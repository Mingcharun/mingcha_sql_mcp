package redis

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

// Config Redis 配置结构。
type Config struct {
	Addr                  string `json:"addr"`
	Password              string `json:"password"`
	DB                    int    `json:"db"`
	SSLInsecureSkipVerify *bool  `json:"ssl_insecure_skip_verify,omitempty"`
}

// Client Redis 客户端包装器。
type Client struct {
	client *goredis.Client
	config Config
}

// NewClient 创建新的 Redis 客户端。
func NewClient(config Config) *Client {
	options := &goredis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}

	// 配置 TLS，如果指定了 ssl_insecure_skip_verify 为 true 时跳过SSL验证
	if config.SSLInsecureSkipVerify != nil && *config.SSLInsecureSkipVerify == true {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rdb := goredis.NewClient(options)

	return &Client{
		client: rdb,
		config: config,
	}
}

// Close 关闭Redis连接
func (r *Client) Close() error {
	return r.client.Close()
}

// Config 返回当前连接配置副本。
func (r *Client) Config() Config {
	return r.config
}

// Ping 测试Redis连接
func (r *Client) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// ExecuteCommand 执行Redis命令
func (r *Client) ExecuteCommand(ctx context.Context, cmdArgs []interface{}) (interface{}, error) {
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := r.client.Do(ctx, cmdArgs...)
	result, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("redis command failed: %w", err)
	}

	return result, nil
}

// ExecuteLuaScript 执行Lua脚本
func (r *Client) ExecuteLuaScript(ctx context.Context, script string, keys []string, args []interface{}) (interface{}, error) {
	cmd := r.client.Eval(ctx, script, keys, args...)
	result, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("lua script execution failed: %w", err)
	}

	return result, nil
}

// ParseCommand 解析 Redis 命令字符串。
func ParseCommand(cmdStr string) ([]interface{}, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil, fmt.Errorf("empty command")
	}

	parts, err := splitCommand(cmdStr)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	args := make([]interface{}, len(parts))
	for i, part := range parts {
		// 尝试解析为数字
		if intVal, err := strconv.Atoi(part); err == nil {
			args[i] = intVal
		} else if floatVal, err := strconv.ParseFloat(part, 64); err == nil {
			args[i] = floatVal
		} else {
			args[i] = part
		}
	}

	return args, nil
}

func splitCommand(cmdStr string) ([]string, error) {
	var (
		parts   []string
		current bytes.Buffer
		quote   rune
		escape  bool
	)

	flush := func() {
		if current.Len() == 0 {
			return
		}
		parts = append(parts, current.String())
		current.Reset()
	}

	for _, r := range cmdStr {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\':
			escape = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escape {
		return nil, fmt.Errorf("unterminated escape sequence")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted string")
	}
	flush()

	return parts, nil
}

// FormatResult 将 Redis 结果格式化为 JSON 友好的文本。
func FormatResult(result interface{}) (string, error) {
	switch v := result.(type) {
	case nil:
		return "null", nil
	case string:
		return fmt.Sprintf("\"%s\"", v), nil
	case []byte:
		return fmt.Sprintf("\"%s\"", string(v)), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case []interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal slice: %w", err)
		}
		return string(jsonBytes), nil
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal map: %w", err)
		}
		return string(jsonBytes), nil
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(jsonBytes), nil
	}
}
