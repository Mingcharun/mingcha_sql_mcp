package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	sqlitedb "github.com/Mingcharun/database-mcp/internal/database/sqlite"
)

func registerSQLiteTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("sqlite_query",
			mcp.WithDescription("Execute SQL query on SQLite database"),
			mcp.WithString("db_path", mcp.Required(), mcp.Description("Path to the SQLite database file")),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL query to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset, default 0")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum rows to return (default: 200, 0 for unlimited)")),
			mcp.WithNumber("timeout_ms", mcp.Description("Optional query timeout in milliseconds")),
		),
		handleSQLiteQuery,
	)
}

func handleSQLiteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbPath, err := request.RequireString("db_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sqlQuery, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := sqlitedb.InitDB(dbPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to connect to database: %v", err)), nil
	}
	defer sqlitedb.CloseDB()

	db := sqlitedb.DB()
	if db == nil {
		return mcp.NewToolResultError("database not initialized"), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	queryArgs := requestArrayArgs(request, "args")
	if isSQLiteExecStatement(sqlQuery) {
		result, err := db.ExecContext(queryCtx, sqlQuery, queryArgs...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get rows affected: %v", err)), nil
		}
		response := map[string]interface{}{
			"type":         "modification",
			"rowsAffected": rowsAffected,
		}
		if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(sqlQuery)), "INSERT") {
			if lastID, err := result.LastInsertId(); err == nil {
				response["lastInsertId"] = lastID
			}
		}
		return jsonToolResult(response)
	}

	rows, err := db.QueryContext(queryCtx, sqlQuery, queryArgs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get columns: %v", err)), nil
	}

	var (
		results    []map[string]interface{}
		hasMore    bool
		offset     = requestOffset(request)
		maxRows    = requestMaxRows(request)
		skipped    int
		nextOffset int
	)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scan row: %v", err)), nil
		}

		if skipped < offset {
			skipped++
			continue
		}
		if maxRows > 0 && len(results) >= maxRows {
			hasMore = true
			continue
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Row iteration error: %v", err)), nil
	}

	if hasMore {
		nextOffset = offset + len(results)
	}

	return jsonToolResult(map[string]interface{}{
		"type":        "select",
		"data":        results,
		"count":       len(results),
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": nextOffset,
		"truncated":   hasMore,
	})
}

func isSQLiteExecStatement(sqlText string) bool {
	sqlUpper := strings.TrimSpace(strings.ToUpper(sqlText))
	execPrefixes := []string{
		"INSERT",
		"UPDATE",
		"DELETE",
		"CREATE",
		"ALTER",
		"DROP",
		"REPLACE",
		"BEGIN",
		"COMMIT",
		"ROLLBACK",
		"VACUUM",
		"ANALYZE",
		"ATTACH",
		"DETACH",
		"REINDEX",
	}

	for _, prefix := range execPrefixes {
		if strings.HasPrefix(sqlUpper, prefix) {
			return true
		}
	}
	return false
}
