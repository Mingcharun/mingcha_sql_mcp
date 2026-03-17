# Database MCP

[English](README.md) | 简体中文

[![Go](https://img.shields.io/badge/Go-Database%20MCP-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Protocol](https://img.shields.io/badge/Protocol-MCP-111827)](https://modelcontextprotocol.io/)
[![Runtime](https://img.shields.io/badge/Use%20Case-AI%20Agents%20%26%20IDEs-2563EB)](https://github.com/Mingcharun/mingcha_sql_mcp)

一个面向 AI Agent、IDE 助手与桌面 MCP Client 的开源数据库 MCP Server。

> 这个项目面向真实项目环境设计：数据库连接信息通常写在仓库配置文件里，而不只是系统环境变量中。

| 从这里开始 | 继续阅读 |
| --- | --- |
| [安装说明](docs/installation.md) | [架构说明](docs/architecture.md) |
| [工具参考](docs/tool-reference.md) | [开发指南](docs/development.md) |
| [发布指南](docs/release.md) | [English](README.md) |

Database MCP 为模型提供统一、结构化的数据库能力入口，当前覆盖：

| 数据源 | 主要能力 |
| --- | --- |
| MySQL | 连接、查询、写操作、存储过程、连接状态 |
| PostgreSQL | 连接、查询、写操作、元数据、连接状态 |
| Redis | 连接、命令执行、Lua、连接状态 |
| SQLite | 查询、写操作、项目配置自动定位 |

它重点解决一个非常现实的问题：

很多团队并不会把数据库连接信息放在系统环境变量里，而是写在项目配置文件中，例如 `.env`、`application.yml`、`application.properties`、`config.json`、`config.toml`。

Database MCP 可以直接从项目目录中识别这些配置，并帮助 AI 自动建立连接。

## 这个项目解决什么问题

大多数 “AI 访问数据库” 的工作流，都会卡在两个地方：

1. 模型没有一个安全、结构化的数据库执行接口
2. 模型不知道数据库配置到底写在哪个项目文件里

Database MCP 同时解决这两个问题：

- 它把数据库能力暴露成 MCP 工具
- 它让 AI 可以从项目配置文件中识别数据库连接信息

这使它非常适合 PHP、Go、Java、Spring、Node、Python，以及混合型仓库场景。

## 主要特性

- 为 MySQL、PostgreSQL、Redis、SQLite 提供统一 MCP 接口
- 内置分页与超时控制，更适合 Agent 对话式调用
- 提供 `connect`、`status`、`disconnect` 这一类连接生命周期工具
- 支持从项目配置文件自动识别数据库连接信息
- 提供 `*_from_project` 工具，减少 AI 手工拼接连接参数
- 仓库结构清晰，适合长期维护与开源协作

## “自动调用”到底是怎么发生的

最容易误解的一点是：

Database MCP 不会自己去“监控仓库”或者“主动读取你的代码”。

实际运行链路是：

1. MCP 客户端启动 `database-mcp`
2. AI 看见这个 MCP 暴露了哪些工具
3. AI 根据你的请求选择下一步调用哪个工具
4. 如果项目把数据库配置写在文件里，AI 可以先调用：
   `project_detect_database_configs`
5. MCP 返回识别到的配置结果后，AI 再调用：
   `mysql_connect_from_project`
   `pgsql_connect_from_project`
   `redis_connect_from_project`
   或 `sqlite_query_from_project`
6. 建立连接后，再继续执行查询、元数据探查或写操作

所以所谓“自动”，本质上是：

- AI 负责理解意图
- MCP 负责提供能力
- 客户端负责执行工具调用
- Database MCP 负责解析配置、建立连接并返回结果

## 当前支持的项目配置来源

目前已支持常见项目配置来源，例如：

- `.env`
- `.env.local`
- `application.yml`
- `application.yaml`
- `application.properties`
- `config.json`
- `config.toml`

同时支持一些常见配置模式：

- host / port / username / password 字段
- DSN / URL 形式连接串
- Spring datasource 配置
- `${DB_HOST}`、`${DB_PORT:5432}` 这类占位符展开

## 工具覆盖

| 分类 | 工具数量 |
| --- | ---: |
| MySQL | 11 |
| PostgreSQL | 11 |
| Redis | 6 |
| SQLite | 2 |
| 项目配置探测 | 1 |
| 总计 | 31 |

## 快速开始

### 从源码构建

```bash
git clone https://github.com/Mingcharun/mingcha_sql_mcp.git
cd mingcha_sql_mcp
./scripts/build.sh
```

产物位置：

```text
dist/database-mcp
```

### 使用安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/Mingcharun/mingcha_sql_mcp/main/scripts/install.sh | bash
```

### 使用 npm 运行

```bash
npx -y @mingcharun/database-mcp
```

### 面向项目配置的第一步调用

如果数据库配置写在项目目录里，推荐先调用：

```text
project_detect_database_configs
```

然后再继续调用对应的项目感知工具：

- `mysql_connect_from_project`
- `pgsql_connect_from_project`
- `redis_connect_from_project`
- `sqlite_query_from_project`

## MCP 客户端配置示例

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

### npm 方式

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

## 常见使用方式

### 当你已经知道数据库连接信息

直接使用普通连接工具：

- `mysql_connect`
- `pgsql_connect`
- `redis_connect`
- `sqlite_query`

### 当数据库配置写在项目文件里

使用项目感知流程：

1. `project_detect_database_configs`
2. 再调用以下之一：
   `mysql_connect_from_project`
   `pgsql_connect_from_project`
   `redis_connect_from_project`
3. 后续继续查询、结构探查或写操作

如果是 SQLite：

1. `project_detect_database_configs`
2. `sqlite_query_from_project`

## 仓库结构

```text
.
├── cmd/database-mcp/           # 二进制入口
├── internal/service/           # MCP 工具注册与 handler
├── internal/database/          # 各数据库实现
├── internal/projectconfig/     # 项目配置探测与解析
├── docs/                       # 项目文档
├── packages/npm/               # npm 分发包装
├── scripts/                    # 构建与安装脚本
├── README.md                   # 英文首页
└── README.zh-CN.md             # 中文首页
```

## 文档导航

- 安装说明：[`docs/installation.md`](docs/installation.md)
- 架构说明：[`docs/architecture.md`](docs/architecture.md)
- 工具参考：[`docs/tool-reference.md`](docs/tool-reference.md)
- 开发指南：[`docs/development.md`](docs/development.md)
- 发布指南：[`docs/release.md`](docs/release.md)

## 验证命令

```bash
go test ./...
go test -race ./...
go vet ./...
./scripts/build.sh
```

## 设计原则

- 让数据库能力真正适合 Agent 调用
- 让响应规模可控、结果结构清晰
- 将 MCP 编排层与数据库实现层解耦
- 支持真实项目中的配置文件形态
- 让仓库保持整洁、易维护、易贡献
