package service

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	redisdb "github.com/Mingcharun/database-mcp/internal/database/redis"
)

func registerRedisTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("redis_connect",
			mcp.WithDescription("连接到Redis服务器"),
			mcp.WithString("addr", mcp.Required(), mcp.Description("Redis服务器地址 (例如: 127.0.0.1:6379)")),
			mcp.WithString("password", mcp.Description("Redis密码")),
			mcp.WithNumber("db", mcp.DefaultNumber(0), mcp.Description("Redis数据库编号")),
			mcp.WithBoolean("ssl_insecure_skip_verify", mcp.Description("是否跳过SSL证书验证，设置为true时启用跳过验证(默认不设置)")),
		),
		handleRedisConnect,
	)

	s.AddTool(
		mcp.NewTool("redis_command",
			mcp.WithDescription("执行任意Redis命令"),
			mcp.WithString("command", mcp.Description("Redis命令字符串 (例如: SET key value 或 GET key)")),
			mcp.WithArray("args", mcp.Description("结构化命令参数列表，优先于 command 使用")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，命令超时时间，单位毫秒")),
		),
		handleRedisCommand,
	)

	s.AddTool(
		mcp.NewTool("redis_lua",
			mcp.WithDescription("执行Lua脚本"),
			mcp.WithString("script", mcp.Required(), mcp.Description("Lua脚本代码")),
			mcp.WithArray("keys", mcp.Description("脚本中使用的键名列表")),
			mcp.WithArray("args", mcp.Description("脚本参数列表")),
			mcp.WithNumber("timeout_ms", mcp.Description("可选，脚本执行超时时间，单位毫秒")),
		),
		handleRedisLua,
	)

	s.AddTool(
		mcp.NewTool("redis_status",
			mcp.WithDescription("获取当前Redis连接状态"),
		),
		handleRedisStatus,
	)

	s.AddTool(
		mcp.NewTool("redis_disconnect",
			mcp.WithDescription("关闭当前Redis连接"),
		),
		handleRedisDisconnect,
	)
}

func handleRedisConnect(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	addr, err := req.RequireString("addr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	config := redisdb.Config{
		Addr:     addr,
		Password: req.GetString("password", ""),
		DB:       req.GetInt("db", 0),
	}

	if rawArgs := req.GetArguments(); rawArgs != nil {
		if sslSkipVerify, exists := rawArgs["ssl_insecure_skip_verify"]; exists {
			if skipVerify, ok := sslSkipVerify.(bool); ok {
				config.SSLInsecureSkipVerify = &skipVerify
			}
		}
	}

	if previous := swapRedisConnection(nil); previous != nil {
		_ = previous.Close()
	}

	client := redisdb.NewClient(config)
	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return mcp.NewToolResultError(fmt.Sprintf("连接失败: %v", err)), nil
	}
	swapRedisConnection(client)

	return jsonToolResult(map[string]interface{}{
		"status": "connected",
		"addr":   addr,
		"db":     config.DB,
	})
}

func handleRedisCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getRedisConnection()
	if client == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	command, err := req.RequireString("command")
	args := requestArrayArgs(req, "args")
	if len(args) == 0 {
		if err != nil {
			return mcp.NewToolResultError("必须提供 command 或 args 参数"), nil
		}
		args, err = redisdb.ParseCommand(command)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("解析命令失败: %v", err)), nil
		}
	}

	queryCtx, cancel := withOptionalTimeout(ctx, req)
	defer cancel()

	result, err := client.ExecuteCommand(queryCtx, args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行命令失败: %v", err)), nil
	}

	formattedResult, err := redisdb.FormatResult(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("格式化结果失败: %v", err)), nil
	}

	return mcp.NewToolResultText(formattedResult), nil
}

func handleRedisLua(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getRedisConnection()
	if client == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	script, err := req.RequireString("script")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryCtx, cancel := withOptionalTimeout(ctx, req)
	defer cancel()

	result, err := client.ExecuteLuaScript(queryCtx, script, requestStringArgs(req, "keys"), requestArrayArgs(req, "args"))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行Lua脚本失败: %v", err)), nil
	}

	formattedResult, err := redisdb.FormatResult(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("格式化结果失败: %v", err)), nil
	}

	return mcp.NewToolResultText(formattedResult), nil
}

func handleRedisStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client := getRedisConnection()
	if client == nil {
		return jsonToolResult(map[string]interface{}{
			"connected": false,
		})
	}

	config := client.Config()
	skipVerify := false
	if config.SSLInsecureSkipVerify != nil {
		skipVerify = *config.SSLInsecureSkipVerify
	}

	return jsonToolResult(map[string]interface{}{
		"connected":                true,
		"addr":                     config.Addr,
		"db":                       config.DB,
		"ssl_insecure_skip_verify": skipVerify,
	})
}

func handleRedisDisconnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if previous := swapRedisConnection(nil); previous != nil {
		if err := previous.Close(); err != nil {
			return toolResultErrorf("关闭Redis连接失败: %v", err)
		}
	}
	return jsonToolResult(map[string]interface{}{
		"disconnected": true,
	})
}
