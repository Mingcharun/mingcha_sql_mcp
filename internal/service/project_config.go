package service

import (
	"context"
	"fmt"
	"time"

	mysqldb "github.com/Mingcharun/database-mcp/internal/database/mysql"
	postgresdb "github.com/Mingcharun/database-mcp/internal/database/postgres"
	redisdb "github.com/Mingcharun/database-mcp/internal/database/redis"
	projectconfig "github.com/Mingcharun/database-mcp/internal/projectconfig"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerProjectConfigTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("project_detect_database_configs",
			mcp.WithDescription("Scan a project directory and detect database configs from common config files"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project root directory")),
			mcp.WithString("config_file", mcp.Description("Optional specific config file path, absolute or relative to project_path")),
		),
		handleProjectDetectDatabaseConfigs,
	)

	s.AddTool(
		mcp.NewTool("mysql_connect_from_project",
			mcp.WithDescription("Detect MySQL config from project files and connect automatically"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project root directory")),
			mcp.WithString("config_file", mcp.Description("Optional specific config file path, absolute or relative to project_path")),
			mcp.WithBoolean("debug", mcp.Description("Enable debug mode (default: false)")),
			mcp.WithNumber("max_open_conns", mcp.Description("Maximum number of open connections (default: 5)")),
			mcp.WithNumber("max_idle_conns", mcp.Description("Maximum number of idle connections (default: 2)")),
			mcp.WithNumber("conn_max_lifetime_hours", mcp.Description("Connection maximum lifetime in hours (default: 4)")),
		),
		handleMySQLConnectFromProject,
	)

	s.AddTool(
		mcp.NewTool("pgsql_connect_from_project",
			mcp.WithDescription("Detect PostgreSQL config from project files and connect automatically"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project root directory")),
			mcp.WithString("config_file", mcp.Description("Optional specific config file path, absolute or relative to project_path")),
		),
		handlePostgresConnectFromProject,
	)

	s.AddTool(
		mcp.NewTool("redis_connect_from_project",
			mcp.WithDescription("Detect Redis config from project files and connect automatically"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project root directory")),
			mcp.WithString("config_file", mcp.Description("Optional specific config file path, absolute or relative to project_path")),
		),
		handleRedisConnectFromProject,
	)

	s.AddTool(
		mcp.NewTool("sqlite_query_from_project",
			mcp.WithDescription("Detect SQLite database path from project files and execute SQL automatically"),
			mcp.WithString("project_path", mcp.Required(), mcp.Description("Path to the project root directory")),
			mcp.WithString("config_file", mcp.Description("Optional specific config file path, absolute or relative to project_path")),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL query to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset, default 0")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum rows to return (default: 200, 0 for unlimited)")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional query timeout in milliseconds")),
		),
		handleSQLiteQueryFromProject,
	)
}

func handleProjectDetectDatabaseConfigs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := projectconfig.Detect(projectPath, request.GetString("config_file", ""))
	if err != nil {
		return toolResultErrorf("project config detection failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLConnectFromProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	match, err := projectconfig.ResolveMySQL(projectPath, request.GetString("config_file", ""))
	if err != nil {
		return toolResultErrorf("failed to resolve MySQL config: %v", err)
	}

	config := match.Config
	config.Debug = request.GetBool("debug", false)
	config.MaxOpenConns = request.GetInt("max_open_conns", 5)
	config.MaxIdleConns = request.GetInt("max_idle_conns", 2)
	config.ConnMaxLifetime = time.Duration(request.GetFloat("conn_max_lifetime_hours", 4.0)) * time.Hour

	_ = mysqldb.CloseDB()
	if err := mysqldb.InitDB(config); err != nil {
		return toolResultErrorf("failed to connect to MySQL: %v", err)
	}

	return jsonToolResult(map[string]interface{}{
		"type":    "connection",
		"success": true,
		"message": fmt.Sprintf("Successfully connected to MySQL database '%s' at %s", config.DatabaseName, config.Addr),
		"source":  match.Source,
	})
}

func handlePostgresConnectFromProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	match, err := projectconfig.ResolvePostgres(projectPath, request.GetString("config_file", ""))
	if err != nil {
		return toolResultErrorf("failed to resolve PostgreSQL config: %v", err)
	}

	client, err := postgresdb.NewClient(match.Config)
	if err != nil {
		return toolResultErrorf("PostgreSQL连接失败: %v", err)
	}
	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return toolResultErrorf("PostgreSQL连接测试失败: %v", err)
	}

	if previous := swapPostgresClient(client); previous != nil {
		_ = previous.Close()
	}

	return jsonToolResult(map[string]interface{}{
		"status":  "success",
		"message": "PostgreSQL连接成功",
		"config": map[string]interface{}{
			"host":     match.Config.Host,
			"port":     match.Config.Port,
			"database": match.Config.Database,
			"user":     match.Config.User,
			"sslmode":  match.Config.SSLMode,
		},
		"source": match.Source,
	})
}

func handleRedisConnectFromProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	match, err := projectconfig.ResolveRedis(projectPath, request.GetString("config_file", ""))
	if err != nil {
		return toolResultErrorf("failed to resolve Redis config: %v", err)
	}

	if previous := swapRedisConnection(nil); previous != nil {
		_ = previous.Close()
	}

	client := redisdb.NewClient(match.Config)
	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return toolResultErrorf("连接失败: %v", err)
	}
	swapRedisConnection(client)

	return jsonToolResult(map[string]interface{}{
		"status": "connected",
		"addr":   match.Config.Addr,
		"db":     match.Config.DB,
		"source": match.Source,
	})
}

func handleSQLiteQueryFromProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectPath, err := request.RequireString("project_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sqlQuery, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	match, err := projectconfig.ResolveSQLite(projectPath, request.GetString("config_file", ""))
	if err != nil {
		return toolResultErrorf("failed to resolve SQLite config: %v", err)
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := runSQLiteQuery(queryCtx, match.DBPath, sqlQuery, request)
	if err != nil {
		return toolResultErrorf("%v", err)
	}

	result["source"] = match.Source
	return jsonToolResult(result)
}
