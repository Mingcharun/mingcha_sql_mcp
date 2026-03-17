# Database MCP

Database MCP 是一个为 AI Agent、IDE 与桌面 MCP Client 提供统一数据库能力的 Go MCP Server。

它的目标不是“再封装一套数据库 SDK”，而是让模型在一个稳定、可控、可部署的 MCP 进程里完成这些常见任务：

- 连接 MySQL、PostgreSQL、Redis、SQLite
- 查询数据、执行写操作、检查结构与元数据
- 在 Agent 工作流里控制超时、分页与结果规模
- 让 IDE、桌面客户端和团队内部工具复用同一套数据库能力

项目由 Mingcharun 团队维护，仓库的目录、命名、文档与发布流程都已经按长期维护场景重新整理。

## 适合什么场景

### 1. 给 AI IDE 提供数据库上下文

如果你希望 Codex、Claude Desktop 或其他 MCP Client 能直接查询数据库、查看表结构、执行 Redis 命令，这个项目就是一个标准入口。

### 2. 给 Agent 提供受控的数据访问能力

项目内置分页、行数限制、请求超时与连接状态能力，适合放在自动化 Agent、工作流编排器或内部智能助手后面使用。

### 3. 统一多数据库接入方式

如果团队同时维护 MySQL、PostgreSQL、Redis 和 SQLite，Database MCP 可以把这些能力统一暴露给上层工具，而不需要每个 Agent 都单独接一个数据库 SDK。

### 4. 做本地开发排障和结构探查

它很适合做 schema 查看、表字段检查、索引排查、Redis key 操作、SQLite 本地文件调试这类日常工作。

## 当前能力

| 数据源 | 工具数量 | 主要能力 |
| --- | ---: | --- |
| MySQL | 10 | 连接、查询、写操作、存储过程、连接状态 |
| PostgreSQL | 10 | 连接、查询、写操作、schema/table/column/index 元数据、连接状态 |
| Redis | 5 | 连接、命令执行、Lua 脚本、连接状态 |
| SQLite | 1 | 单文件数据库查询与写操作 |
| 合计 | 26 | 统一通过 MCP 暴露 |

### 查询保护能力

查询类工具统一支持以下参数，用于控制 Agent 会话中的资源使用：

- `offset`
- `max_rows`
- `timeout_ms`

查询结果统一返回分页相关字段：

- `count`
- `offset`
- `has_more`
- `next_offset`
- `truncated`

这意味着你可以把它直接接在 Agent 前面，而不用担心一条大查询把会话结果塞爆。

## 快速开始

### 从源码构建

```bash
git clone https://github.com/Mingcharun/mingcha_sql_mcp.git
cd mingcha_sql_mcp
./scripts/build.sh
```

构建产物默认位于：

```text
dist/database-mcp
```

### 通过安装脚本安装

```bash
curl -fsSL https://raw.githubusercontent.com/Mingcharun/mingcha_sql_mcp/main/scripts/install.sh | bash
```

### 通过 npm 运行

```bash
npx -y @mingcharun/database-mcp
```

## MCP Client 配置示例

### Codex

```toml
[mcp_servers.database_mcp]
command = "/absolute/path/to/database-mcp"
```

### Claude Desktop

```json
{
  "mcpServers": {
    "database_mcp": {
      "command": "/absolute/path/to/database-mcp",
      "args": []
    }
  }
}
```

### 通过 npm 配置

```json
{
  "mcpServers": {
    "database_mcp": {
      "command": "npx",
      "args": ["-y", "@mingcharun/database-mcp"]
    }
  }
}
```

## 仓库结构

```text
.
├── cmd/database-mcp/           # 二进制入口
├── internal/service/           # MCP tool 注册、参数解析、响应封装
├── internal/database/          # 各数据库底层实现
│   ├── mysql/
│   ├── postgres/
│   ├── redis/
│   └── sqlite/
├── docs/                       # 项目文档
├── packages/npm/               # npm 分发包装
├── scripts/                    # 构建与安装脚本
└── README.md                   # 仓库总入口
```

## 文档导航

如果你是第一次接触这个项目，建议按下面顺序阅读：

1. 安装与客户端接入：[`docs/installation.md`](docs/installation.md)
2. 场景化使用说明：[`docs/scenarios.md`](docs/scenarios.md)
3. 工具参考：[`docs/tool-reference.md`](docs/tool-reference.md)
4. 开发与扩展：[`docs/development.md`](docs/development.md)
5. 发布流程：[`docs/release.md`](docs/release.md)
6. npm 包维护：[`docs/npm-package.md`](docs/npm-package.md)

## 测试与验证

常用验证命令：

```bash
go test ./...
go test -race ./...
go vet ./...
./scripts/build.sh
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

未配置时，对应的集成测试会自动跳过。

## 设计原则

- 用一个 MCP 进程统一暴露多数据库能力
- 让工具更适合 Agent，而不是只适合人手工调用
- 让查询行为具备超时、分页和结果规模控制
- 让数据库实现与 MCP 层分离，便于长期维护
- 让部署、安装、测试和发布流程尽量可预测
