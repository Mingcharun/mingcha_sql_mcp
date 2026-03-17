# 工具参考

本文档按“任务类型”而不是按源码文件组织，帮助你快速判断当前该调用哪个工具。

## 通用行为

### 查询保护参数

支持查询的工具通常都支持以下参数：

- `offset`: 从哪一条开始返回，默认 `0`
- `max_rows`: 最多返回多少条，默认 `200`
- `timeout_ms`: 单次调用超时，单位毫秒

### 查询结果的公共字段

查询类结果中常见字段：

- `count`
- `offset`
- `has_more`
- `next_offset`
- `truncated`

这些字段的意义是：

- 结果可分页
- 结果可能被截断
- 调用方可以继续翻页而不是一次拉全量

## 如果你想先“连上数据库”

### MySQL

- `mysql_connect`
- `mysql_status`
- `mysql_disconnect`

### PostgreSQL

- `pgsql_connect`
- `pgsql_status`
- `pgsql_disconnect`

### Redis

- `redis_connect`
- `redis_status`
- `redis_disconnect`

### SQLite

SQLite 不需要显式连接管理，直接通过 `sqlite_query` 传 `db_path` 即可。

## 如果你想“查数据”

### MySQL

- `mysql_query`

适合：

- `SELECT`
- `SHOW`
- `DESCRIBE`

### PostgreSQL

- `pgsql_query`

适合：

- 常规 `SELECT`
- 带参数查询
- 结构化结果返回

### SQLite

- `sqlite_query`

适合：

- 本地 SQLite 文件查询
- 轻量离线数据排查

## 如果你想“写数据或执行 DDL”

### MySQL

- `mysql_exec`
- `mysql_exec_get_id`

使用建议：

- 普通写操作用 `mysql_exec`
- 需要返回自增主键时用 `mysql_exec_get_id`

### PostgreSQL

- `pgsql_exec`

使用建议：

- 普通 DML / DDL 都可以走这个工具
- 如果 `INSERT` 带 `RETURNING`，工具会自动返回插入 ID

### SQLite

- `sqlite_query`

说明：

SQLite 使用单工具模式，会自动识别当前 SQL 是查询还是执行语句。

## 如果你想“看结构和元数据”

### PostgreSQL

这是当前结构探查能力最完整的一组：

- `pgsql_info`
- `pgsql_list_schemas`
- `pgsql_list_tables`
- `pgsql_list_columns`
- `pgsql_list_indexes`

推荐顺序：

1. `pgsql_list_schemas`
2. `pgsql_list_tables`
3. `pgsql_list_columns`
4. `pgsql_list_indexes`

### MySQL

当前没有单独拆出的元数据工具，建议用：

- `mysql_query` + `SHOW TABLES`
- `mysql_query` + `DESCRIBE table_name`
- `mysql_query` + `SHOW INDEX FROM table_name`

## 如果你想“操作 Redis”

### 执行普通命令

- `redis_command`

支持两种输入方式：

- `command`: 命令字符串
- `args`: 结构化参数数组

推荐：

- 简单命令可以用 `command`
- 有空格、引号或复杂参数时优先用 `args`

### 执行 Lua 脚本

- `redis_lua`

支持：

- `script`
- `keys`
- `args`

适合：

- 原子更新
- 调试 Redis 脚本逻辑

## 如果你想“调用 MySQL 存储过程”

- `mysql_call_procedure`
- `mysql_create_procedure`
- `mysql_drop_procedure`
- `mysql_show_procedures`

说明：

- `mysql_call_procedure` 支持多结果集
- `mysql_show_procedures` 支持分页

## 场景建议

### 场景一：先看结构，再查数据

推荐：

- PostgreSQL 优先使用元数据工具
- MySQL 优先先 `SHOW TABLES` / `DESCRIBE`

### 场景二：Agent 需要稳定返回

推荐：

- 总是传 `timeout_ms`
- 总是限制 `max_rows`
- 结果大时用 `next_offset` 继续翻页

### 场景三：调试缓存

推荐：

- 先 `redis_connect`
- 再 `redis_command`
- 如需原子逻辑验证，再用 `redis_lua`

## 完整工具列表

### MySQL

- `mysql_connect`
- `mysql_query`
- `mysql_exec`
- `mysql_exec_get_id`
- `mysql_call_procedure`
- `mysql_create_procedure`
- `mysql_drop_procedure`
- `mysql_show_procedures`
- `mysql_status`
- `mysql_disconnect`

### PostgreSQL

- `pgsql_connect`
- `pgsql_query`
- `pgsql_exec`
- `pgsql_info`
- `pgsql_list_schemas`
- `pgsql_list_tables`
- `pgsql_list_columns`
- `pgsql_list_indexes`
- `pgsql_status`
- `pgsql_disconnect`

### Redis

- `redis_connect`
- `redis_command`
- `redis_lua`
- `redis_status`
- `redis_disconnect`

### SQLite

- `sqlite_query`
