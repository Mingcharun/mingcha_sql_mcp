# 使用场景

本文档不按“数据库种类”介绍，而是按真实使用场景组织，帮助你更快判断应该怎么接入、怎么调用、先用哪些工具。

## 场景一：在 AI IDE 中排查数据库问题

适合：

- Codex
- Claude Desktop
- 其他支持 MCP 的 IDE / Agent 客户端

典型目标：

- 看某张表有哪些字段
- 找某个 Redis key 的值
- 检查 PostgreSQL 的索引
- 本地打开 SQLite 文件排查问题

推荐流程：

1. 先连接目标数据库
2. 优先使用结构类工具而不是直接全表扫描
3. 查询时总是带 `timeout_ms`
4. 大结果集总是带 `max_rows`

示例思路：

- MySQL: 先 `mysql_connect`，再 `mysql_query`
- PostgreSQL: 先 `pgsql_connect`，再 `pgsql_list_tables`、`pgsql_list_columns`
- Redis: 先 `redis_connect`，再 `redis_command`
- SQLite: 直接 `sqlite_query`

推荐提示词：

- “连接到我的 PostgreSQL，并列出 public schema 下的表”
- “查看 Redis 中 `session:123` 的内容”
- “打开这个 SQLite 文件，查一下最近 20 条任务记录”

## 场景二：让 Agent 做只读结构探查

适合：

- 让模型理解数据库结构
- 在生成 SQL 前先建立上下文
- 做接口联调时快速理解表关系

建议做法：

- 不要一开始就 `SELECT *`
- 优先使用元数据类工具
- 先缩小范围，再查询业务数据

推荐工具组合：

- PostgreSQL:
  - `pgsql_list_schemas`
  - `pgsql_list_tables`
  - `pgsql_list_columns`
  - `pgsql_list_indexes`
- MySQL:
  - `mysql_query` 配合 `SHOW TABLES`
  - `mysql_query` 配合 `DESCRIBE <table>`

推荐节奏：

1. 列出 schema 或 database 中的表
2. 查看目标表字段
3. 查看索引或主键
4. 最后再执行有针对性的查询

## 场景三：让 Agent 执行可控的数据修改

适合：

- 测试环境修数据
- 本地环境初始化数据
- 受控执行 DDL / DML

建议：

- 先让 Agent 解释它要执行的 SQL
- 再执行 `mysql_exec` / `pgsql_exec` / `sqlite_query`
- 对插入类语句优先用能返回 ID 的工具

常见做法：

- MySQL 插入并拿回 ID: `mysql_exec_get_id`
- PostgreSQL 插入并 `RETURNING id`: `pgsql_exec`
- SQLite 本地修数据: `sqlite_query`

不建议：

- 在生产环境让 Agent 直接执行不受审查的大范围更新
- 不设 `timeout_ms` 就跑复杂查询
- 不分页就把大表结果直接返回给模型

## 场景四：Redis 调试与脚本验证

适合：

- 查看缓存值
- 验证 key 是否存在
- 测试 Lua 脚本

常见工作流：

1. `redis_connect`
2. `redis_command`
3. 如需脚本验证，再用 `redis_lua`
4. 结束后可调用 `redis_disconnect`

示例：

- 读取 key: `GET session:123`
- 检查 TTL: `TTL session:123`
- 原子更新: `redis_lua`

注意：

- `redis_command` 同时支持原始命令字符串和结构化 `args`
- 如果参数中有空格、引号或数字，优先使用结构化 `args`

## 场景五：SQLite 本地文件排查

适合：

- 桌面应用
- 本地缓存数据库
- 测试数据文件
- 单机工具型项目

特点：

- 不需要常驻连接状态
- 直接通过 `db_path` 指定文件
- 自动区分查询语句和写操作语句

建议：

- 总是确认 `db_path` 指向的是正确文件
- 先执行小范围查询
- 对 `INSERT/UPDATE/DELETE` 操作保留上下文说明，便于复盘

## 场景六：团队内部统一数据库 MCP 服务

适合：

- 多个 Agent 共用一套数据库工具
- IDE、桌面客户端、自动化任务统一走 MCP
- 团队希望降低接入成本

推荐方式：

- 由团队维护统一的 Database MCP 二进制版本
- 在内部文档里固定配置方式
- 对不同环境使用不同配置文件或不同启动实例

推荐规范：

- 测试环境和生产环境不要共用一个 MCP 配置
- 固定发布版本，不直接依赖 `latest`
- 将高风险写操作放到受控环境中

## 场景七：二次开发与扩展

如果你是维护者，而不是单纯使用者，建议阅读：

- [`docs/development.md`](development.md)
- [`docs/tool-reference.md`](tool-reference.md)
- [`docs/release.md`](release.md)

新增能力时的建议顺序：

1. 先在 `internal/database/<db>/` 完成底层逻辑
2. 再在 `internal/service/<db>.go` 注册工具
3. 最后补文档、测试和发布说明
