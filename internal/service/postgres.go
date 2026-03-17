package service

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	postgresdb "github.com/Mingcharun/database-mcp/internal/database/postgres"
)

func registerPostgresTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("pgsql_connect",
			mcp.WithDescription("连接到PostgreSQL服务器"),
			mcp.WithString("host", mcp.Required()),
			mcp.WithNumber("port", mcp.DefaultNumber(5432)),
			mcp.WithString("user", mcp.Required()),
			mcp.WithString("password", mcp.Required()),
			mcp.WithString("database", mcp.Required()),
			mcp.WithString("sslmode", mcp.DefaultString("disable")),
		),
		handlePostgresConnect,
	)

	s.AddTool(
		mcp.NewTool("pgsql_query",
			mcp.WithDescription("执行PostgreSQL SELECT查询"),
			mcp.WithString("sql", mcp.Required()),
			mcp.WithArray("args", mcp.Description("SQL 参数列表，按 $1/$2 顺序传入")),
			mcp.WithNumber("offset", mcp.Description("分页偏移量，默认 0")),
			mcp.WithNumber("max_rows", mcp.Description("最大返回行数，默认 200，0 表示不限制")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresQuery,
	)

	s.AddTool(
		mcp.NewTool("pgsql_exec",
			mcp.WithDescription("执行PostgreSQL INSERT/UPDATE/DELETE操作"),
			mcp.WithString("sql", mcp.Required()),
			mcp.WithArray("args", mcp.Description("SQL 参数列表，按 $1/$2 顺序传入")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，执行超时时间，单位毫秒")),
		),
		handlePostgresExec,
	)

	s.AddTool(
		mcp.NewTool("pgsql_info",
			mcp.WithDescription("获取当前PostgreSQL连接信息"),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresInfo,
	)

	s.AddTool(
		mcp.NewTool("pgsql_list_schemas",
			mcp.WithDescription("列出PostgreSQL数据库中的schema"),
			mcp.WithNumber("offset", mcp.Description("分页偏移量，默认 0")),
			mcp.WithNumber("max_rows", mcp.Description("最大返回行数，默认 200，0 表示不限制")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresListSchemas,
	)

	s.AddTool(
		mcp.NewTool("pgsql_list_tables",
			mcp.WithDescription("列出指定schema下的表"),
			mcp.WithString("schema", mcp.Description("Schema 名称，默认 public")),
			mcp.WithNumber("offset", mcp.Description("分页偏移量，默认 0")),
			mcp.WithNumber("max_rows", mcp.Description("最大返回行数，默认 200，0 表示不限制")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresListTables,
	)

	s.AddTool(
		mcp.NewTool("pgsql_list_columns",
			mcp.WithDescription("列出指定表的列信息"),
			mcp.WithString("table_name", mcp.Required()),
			mcp.WithString("schema", mcp.Description("Schema 名称，默认 public")),
			mcp.WithNumber("offset", mcp.Description("分页偏移量，默认 0")),
			mcp.WithNumber("max_rows", mcp.Description("最大返回行数，默认 200，0 表示不限制")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresListColumns,
	)

	s.AddTool(
		mcp.NewTool("pgsql_list_indexes",
			mcp.WithDescription("列出指定表的索引信息"),
			mcp.WithString("table_name", mcp.Required()),
			mcp.WithString("schema", mcp.Description("Schema 名称，默认 public")),
			mcp.WithNumber("offset", mcp.Description("分页偏移量，默认 0")),
			mcp.WithNumber("max_rows", mcp.Description("最大返回行数，默认 200，0 表示不限制")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，查询超时时间，单位毫秒")),
		),
		handlePostgresListIndexes,
	)

	s.AddTool(
		mcp.NewTool("pgsql_status",
			mcp.WithDescription("获取当前PostgreSQL连接状态"),
		),
		handlePostgresStatus,
	)

	s.AddTool(
		mcp.NewTool("pgsql_disconnect",
			mcp.WithDescription("关闭当前PostgreSQL连接"),
		),
		handlePostgresDisconnect,
	)
}

func getStringParam(args map[string]interface{}, key string, defaultValue string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultValue
}

func getNumberParam(args map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := args[key].(float64); ok {
		return val
	}
	return defaultValue
}

func handlePostgresConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments == nil {
		return toolResultErrorf("缺少必需参数")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return toolResultErrorf("参数格式错误")
	}

	config := postgresdb.Config{
		Host:     getStringParam(args, "host", "localhost"),
		Port:     int(getNumberParam(args, "port", 5432)),
		User:     getStringParam(args, "user", ""),
		Password: getStringParam(args, "password", ""),
		Database: getStringParam(args, "database", ""),
		SSLMode:  getStringParam(args, "sslmode", "disable"),
	}

	client, err := postgresdb.NewClient(config)
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
			"host":     config.Host,
			"port":     config.Port,
			"database": config.Database,
			"user":     config.User,
			"sslmode":  config.SSLMode,
		},
	})
}

func handlePostgresQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return toolResultErrorf("参数格式错误")
	}

	sql := getStringParam(args, "sql", "")
	if sql == "" {
		return toolResultErrorf("SQL语句不能为空")
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.QueryWithOptions(
		queryCtx,
		sql,
		requestOffset(request),
		requestMaxRows(request),
		requestArrayArgs(request, "args")...,
	)
	if err != nil {
		return toolResultErrorf("查询执行失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresExec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return toolResultErrorf("参数格式错误")
	}

	sql := getStringParam(args, "sql", "")
	if sql == "" {
		return toolResultErrorf("SQL语句不能为空")
	}

	queryArgs := requestArrayArgs(request, "args")
	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	var (
		result interface{}
		err    error
	)
	if upperSQL := strings.ToUpper(strings.TrimSpace(sql)); strings.HasPrefix(upperSQL, "INSERT") && strings.Contains(upperSQL, "RETURNING") {
		result, err = client.ExecWithLastInsertId(queryCtx, sql, queryArgs...)
	} else {
		result, err = client.Exec(queryCtx, sql, queryArgs...)
	}
	if err != nil {
		return toolResultErrorf("执行失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.GetInfo(queryCtx)
	if err != nil {
		return toolResultErrorf("获取连接信息失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresListSchemas(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.ListSchemasWithOptions(queryCtx, requestOffset(request), requestMaxRows(request))
	if err != nil {
		return toolResultErrorf("列出 schema 失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.ListTablesWithOptions(
		queryCtx,
		request.GetString("schema", "public"),
		requestOffset(request),
		requestMaxRows(request),
	)
	if err != nil {
		return toolResultErrorf("列出表失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresListColumns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}

	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.ListColumnsWithOptions(
		queryCtx,
		tableName,
		request.GetString("schema", "public"),
		requestOffset(request),
		requestMaxRows(request),
	)
	if err != nil {
		return toolResultErrorf("列出列信息失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresListIndexes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return toolResultErrorf("请先连接到PostgreSQL服务器")
	}

	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := client.ListIndexesWithOptions(
		queryCtx,
		tableName,
		request.GetString("schema", "public"),
		requestOffset(request),
		requestMaxRows(request),
	)
	if err != nil {
		return toolResultErrorf("列出索引信息失败: %v", err)
	}
	return jsonToolResult(result)
}

func handlePostgresStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getPostgresClient()
	if client == nil {
		return jsonToolResult(map[string]interface{}{
			"connected": false,
		})
	}

	config := client.Config()
	return jsonToolResult(map[string]interface{}{
		"connected": true,
		"host":      config.Host,
		"port":      config.Port,
		"database":  config.Database,
		"user":      config.User,
		"sslmode":   config.SSLMode,
	})
}

func handlePostgresDisconnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if previous := swapPostgresClient(nil); previous != nil {
		if err := previous.Close(); err != nil {
			return toolResultErrorf("关闭PostgreSQL连接失败: %v", err)
		}
	}
	return jsonToolResult(map[string]interface{}{
		"disconnected": true,
	})
}
