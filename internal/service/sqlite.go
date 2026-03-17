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

	queryCtx, cancel := withOptionalTimeout(ctx, request)
	defer cancel()

	result, err := runSQLiteQuery(queryCtx, dbPath, sqlQuery, request)
	if err != nil {
		return toolResultErrorf("%v", err)
	}
	return jsonToolResult(result)
}

func runSQLiteQuery(ctx context.Context, dbPath, sqlQuery string, request mcp.CallToolRequest) (map[string]interface{}, error) {
	if err := sqlitedb.InitDB(dbPath); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer sqlitedb.CloseDB()

	db := sqlitedb.DB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	queryArgs := requestArrayArgs(request, "args")
	if isSQLiteExecStatement(sqlQuery) {
		result, err := db.ExecContext(ctx, sqlQuery, queryArgs...)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
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
		return response, nil
	}

	rows, err := db.QueryContext(ctx, sqlQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
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
			return nil, fmt.Errorf("failed to scan row: %w", err)
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
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if hasMore {
		nextOffset = offset + len(results)
	}

	return map[string]interface{}{
		"type":        "select",
		"data":        results,
		"count":       len(results),
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": nextOffset,
		"truncated":   hasMore,
	}, nil
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
