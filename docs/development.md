# 开发说明

本文档面向维护者和二次开发者，说明项目结构、设计边界、扩展方式和测试要求。

## 仓库结构

```text
cmd/database-mcp/
  main.go

internal/service/
  common.go
  mysql.go
  postgres.go
  redis.go
  sqlite.go

internal/database/
  mysql/
  postgres/
  redis/
  sqlite/

docs/
packages/npm/
scripts/
```

## 设计边界

### `cmd/database-mcp`

职责：

- 进程入口
- 版本输出
- 启动 stdio MCP 服务

这里不应放：

- 业务逻辑
- 参数解析细节
- 数据库访问实现

### `internal/service`

职责：

- 定义 MCP tool
- 读取和校验参数
- 处理通用分页、超时、结果封装
- 管理共享连接状态

这里不应放：

- 重型 SQL 扫描逻辑
- 复杂数据库连接实现

### `internal/database`

职责：

- 数据库连接
- 查询和执行逻辑
- 行扫描、结果转换
- 数据库特有行为封装

这里应尽量保持：

- 对外 API 清晰
- 与 MCP 协议无直接耦合
- 可测试

## 命名约定

- 目录名使用小写语义名
- Go 包名与目录保持一致
- 类型名尽量使用中性业务词，例如 `Client`、`Config`
- Tool 名保留数据库前缀，例如 `mysql_query`
- 避免把个人、历史项目名写入包名、文件名、变量名

## 新增一个工具时的建议流程

1. 先确认它属于哪个数据库实现层
2. 在 `internal/database/<db>/` 增加底层能力
3. 在 `internal/service/<db>.go` 注册 tool
4. 统一接入 `offset`、`max_rows`、`timeout_ms`
5. 补测试
6. 更新文档

## 测试策略

### 单元与集成混合

本项目当前以“轻量单元测试 + 环境变量驱动的集成测试”为主。

### 推荐命令

```bash
go test ./...
go test -race ./...
go vet ./...
```

### 集成测试环境变量

MySQL:

- `MYSQL_TEST_USER`
- `MYSQL_TEST_PASSWORD`
- `MYSQL_TEST_ADDR`
- `MYSQL_TEST_DATABASE`

PostgreSQL:

- `POSTGRES_TEST_HOST`
- `POSTGRES_TEST_PORT`
- `POSTGRES_TEST_USER`
- `POSTGRES_TEST_PASSWORD`
- `POSTGRES_TEST_DATABASE`
- `POSTGRES_TEST_SSLMODE`

Redis:

- `REDIS_TEST_ADDR`
- `REDIS_TEST_PASSWORD`
- `REDIS_TEST_DB`

未配置时自动跳过对应集成测试。

## 开发时应优先关注的问题

### 1. Agent 可用性

不要只从“数据库操作是否成功”判断代码质量，还要考虑：

- 结果是否过大
- 是否容易让模型误用
- 错误信息是否足够清楚
- 是否容易超时

### 2. 响应一致性

同类工具尽量保持：

- 相似参数命名
- 相似的错误风格
- 相似的分页字段

### 3. 连接生命周期

新增连接工具时要考虑：

- 是否需要状态查询
- 是否需要断开连接
- 是否会覆盖旧连接
- 是否存在并发访问问题

## 本地开发建议

### 修改代码后至少执行

```bash
gofmt -w ./cmd ./internal
go test ./...
```

### 在提交前执行

```bash
go test -race ./...
go vet ./...
./scripts/build.sh
```

## 文档同步要求

以下变更必须更新文档：

- 新增 MCP tool
- 修改 tool 参数
- 修改返回字段
- 改变构建产物名
- 改变 npm 包名
- 改变发布流程
