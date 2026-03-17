package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	postgresdb "github.com/Mingcharun/database-mcp/internal/database/postgres"
	redisdb "github.com/Mingcharun/database-mcp/internal/database/redis"
)

const DefaultQueryMaxRows = 200

var (
	postgresClient  *postgresdb.Client
	redisConnection *redisdb.Client
	sharedClientsMu sync.RWMutex
)

// New creates a configured MCP server with all database tools registered.
func New(serviceName, serviceVersion string) *server.MCPServer {
	mcpServer := server.NewMCPServer(
		serviceName,
		serviceVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	registerMySQLTools(mcpServer)
	registerPostgresTools(mcpServer)
	registerRedisTools(mcpServer)
	registerSQLiteTools(mcpServer)

	return mcpServer
}

func jsonToolResult(payload interface{}) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

func toolResultErrorf(format string, args ...interface{}) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

func requestArrayArgs(request mcp.CallToolRequest, key string) []interface{} {
	rawArgs := request.GetArguments()
	if rawArgs == nil {
		return nil
	}
	argsVal, ok := rawArgs[key]
	if !ok {
		return nil
	}
	argsSlice, ok := argsVal.([]interface{})
	if !ok {
		return nil
	}
	return argsSlice
}

func requestStringArgs(request mcp.CallToolRequest, key string) []string {
	rawArgs := request.GetArguments()
	if rawArgs == nil {
		return nil
	}
	argsVal, ok := rawArgs[key]
	if !ok {
		return nil
	}
	argsSlice, ok := argsVal.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(argsSlice))
	for _, item := range argsSlice {
		value, ok := item.(string)
		if !ok {
			return nil
		}
		result = append(result, value)
	}
	return result
}

func requestMaxRows(request mcp.CallToolRequest) int {
	maxRows := request.GetInt("max_rows", DefaultQueryMaxRows)
	if maxRows < 0 {
		return DefaultQueryMaxRows
	}
	return maxRows
}

func requestOffset(request mcp.CallToolRequest) int {
	offset := request.GetInt("offset", 0)
	if offset < 0 {
		return 0
	}
	return offset
}

func withOptionalTimeout(ctx context.Context, request mcp.CallToolRequest) (context.Context, context.CancelFunc) {
	timeoutMs := request.GetInt("timeout_ms", 0)
	if timeoutMs <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
}

func getPostgresClient() *postgresdb.Client {
	sharedClientsMu.RLock()
	defer sharedClientsMu.RUnlock()
	return postgresClient
}

func swapPostgresClient(client *postgresdb.Client) *postgresdb.Client {
	sharedClientsMu.Lock()
	defer sharedClientsMu.Unlock()
	previous := postgresClient
	postgresClient = client
	return previous
}

func getRedisConnection() *redisdb.Client {
	sharedClientsMu.RLock()
	defer sharedClientsMu.RUnlock()
	return redisConnection
}

func swapRedisConnection(client *redisdb.Client) *redisdb.Client {
	sharedClientsMu.Lock()
	defer sharedClientsMu.Unlock()
	previous := redisConnection
	redisConnection = client
	return previous
}
