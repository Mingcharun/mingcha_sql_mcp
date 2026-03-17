package service

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	mysqldb "github.com/Mingcharun/database-mcp/internal/database/mysql"
)

func registerMySQLTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("mysql_connect",
			mcp.WithDescription("Connect to MySQL database with dynamic connection parameters"),
			mcp.WithString("username", mcp.Required(), mcp.Description("MySQL username")),
			mcp.WithString("password", mcp.Required(), mcp.Description("MySQL password")),
			mcp.WithString("addr", mcp.Required(), mcp.Description("MySQL server address (host:port)")),
			mcp.WithString("database_name", mcp.Required(), mcp.Description("Database name to connect to")),
			mcp.WithBoolean("debug", mcp.Description("Enable debug mode (default: false)")),
			mcp.WithNumber("max_open_conns", mcp.Description("Maximum number of open connections (default: 100)")),
			mcp.WithNumber("max_idle_conns", mcp.Description("Maximum number of idle connections (default: 50)")),
			mcp.WithNumber("conn_max_lifetime_hours", mcp.Description("Connection maximum lifetime in hours (default: 4)")),
		),
		handleMySQLConnect,
	)

	s.AddTool(
		mcp.NewTool("mysql_query",
			mcp.WithDescription("Execute MySQL query operations (SELECT/SHOW/DESCRIBE, etc.)"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL query to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset, default 0")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum rows to return (default: 200, 0 for unlimited)")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional query timeout in milliseconds")),
		),
		handleMySQLQuery,
	)

	s.AddTool(
		mcp.NewTool("mysql_exec",
			mcp.WithDescription("Execute MySQL DML/DDL operations (INSERT/UPDATE/DELETE/CREATE TABLE/ALTER TABLE/DROP TABLE, etc.)"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL statement to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional execution timeout in milliseconds")),
		),
		handleMySQLExec,
	)

	s.AddTool(
		mcp.NewTool("mysql_exec_get_id",
			mcp.WithDescription("Execute MySQL INSERT operation and return the last inserted ID"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL INSERT statement to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional execution timeout in milliseconds")),
		),
		handleMySQLExecGetID,
	)

	s.AddTool(
		mcp.NewTool("mysql_call_procedure",
			mcp.WithDescription("Call MySQL stored procedure"),
			mcp.WithString("procedure_name", mcp.Required(), mcp.Description("Name of the stored procedure to call")),
			mcp.WithArray("args", mcp.Description("Arguments to pass to the stored procedure")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum rows to return per result set (default: 200, 0 for unlimited)")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional execution timeout in milliseconds")),
		),
		handleMySQLCallProcedure,
	)

	s.AddTool(
		mcp.NewTool("mysql_create_procedure",
			mcp.WithDescription("Create MySQL stored procedure"),
			mcp.WithString("procedure_sql", mcp.Required(), mcp.Description("Complete CREATE PROCEDURE SQL statement")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional execution timeout in milliseconds")),
		),
		handleMySQLCreateProcedure,
	)

	s.AddTool(
		mcp.NewTool("mysql_drop_procedure",
			mcp.WithDescription("Drop MySQL stored procedure"),
			mcp.WithString("procedure_name", mcp.Required(), mcp.Description("Name of the stored procedure to drop")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional execution timeout in milliseconds")),
		),
		handleMySQLDropProcedure,
	)

	s.AddTool(
		mcp.NewTool("mysql_show_procedures",
			mcp.WithDescription("Show list of stored procedures in the current database"),
			mcp.WithString("database_name", mcp.Description("Database name (if not provided, uses current connection database)")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset, default 0")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum rows to return (default: 200, 0 for unlimited)")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional query timeout in milliseconds")),
		),
		handleMySQLShowProcedures,
	)

	s.AddTool(
		mcp.NewTool("mysql_status",
			mcp.WithDescription("Show current MySQL connection status"),
		),
		handleMySQLStatus,
	)

	s.AddTool(
		mcp.NewTool("mysql_disconnect",
			mcp.WithDescription("Close the current MySQL connection"),
		),
		handleMySQLDisconnect,
	)
}

func handleMySQLConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, err := request.RequireString("username")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	password, err := request.RequireString("password")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	addr, err := request.RequireString("addr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	databaseName, err := request.RequireString("database_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	config := mysqldb.ConnectionConfig{
		Username:        username,
		Password:        password,
		Addr:            addr,
		DatabaseName:    databaseName,
		Debug:           request.GetBool("debug", false),
		MaxOpenConns:    request.GetInt("max_open_conns", 5),
		MaxIdleConns:    request.GetInt("max_idle_conns", 2),
		ConnMaxLifetime: time.Duration(request.GetFloat("conn_max_lifetime_hours", 4.0)) * time.Hour,
	}

	_ = mysqldb.CloseDB()
	if err := mysqldb.InitDB(config); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to connect to MySQL: %v", err)), nil
	}

	return jsonToolResult(map[string]interface{}{
		"type":    "connection",
		"success": true,
		"message": fmt.Sprintf("Successfully connected to MySQL database '%s' at %s", databaseName, addr),
	})
}

func handleMySQLQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.QueryContext(
		queryCtx,
		sql,
		requestOffset(request),
		requestMaxRows(request),
		requestArrayArgs(request, "args")...,
	)
	if err != nil {
		return toolResultErrorf("Query execution failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLExec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.ExecContext(queryCtx, sql, requestArrayArgs(request, "args")...)
	if err != nil {
		return toolResultErrorf("Execution failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLExecGetID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.ExecWithLastIDContext(queryCtx, sql, requestArrayArgs(request, "args")...)
	if err != nil {
		return toolResultErrorf("Execution failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLCallProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	procName, err := request.RequireString("procedure_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.CallProcedureContext(
		queryCtx,
		procName,
		requestMaxRows(request),
		requestArrayArgs(request, "args")...,
	)
	if err != nil {
		return toolResultErrorf("Procedure call failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLCreateProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	procedureSQL, err := request.RequireString("procedure_sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.CreateProcedureContext(queryCtx, procedureSQL)
	if err != nil {
		return toolResultErrorf("Create procedure failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLDropProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	procName, err := request.RequireString("procedure_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.DropProcedureContext(queryCtx, procName)
	if err != nil {
		return toolResultErrorf("Drop procedure failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLShowProcedures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysqldb.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}

	databaseName := request.GetString("database_name", "")
	if databaseName == "" {
		return mcp.NewToolResultError("database_name parameter is required"), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := mysqldb.ShowProceduresContext(
		queryCtx,
		databaseName,
		requestOffset(request),
		requestMaxRows(request),
	)
	if err != nil {
		return toolResultErrorf("Show procedures failed: %v", err)
	}
	return jsonToolResult(result)
}

func handleMySQLStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	config := mysqldb.CurrentConfig()
	if config == nil {
		return jsonToolResult(map[string]interface{}{
			"connected": false,
			"database":  "",
			"addr":      "",
		})
	}

	return jsonToolResult(map[string]interface{}{
		"connected":         mysqldb.IsConnected(),
		"database":          config.DatabaseName,
		"addr":              config.Addr,
		"username":          config.Username,
		"max_open_conns":    config.MaxOpenConns,
		"max_idle_conns":    config.MaxIdleConns,
		"conn_max_lifetime": config.ConnMaxLifetime.String(),
		"debug":             config.Debug,
	})
}

func handleMySQLDisconnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := mysqldb.CloseDB(); err != nil {
		return toolResultErrorf("close MySQL connection failed: %v", err)
	}
	return jsonToolResult(map[string]interface{}{
		"disconnected": true,
	})
}
