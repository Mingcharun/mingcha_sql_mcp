# Development Guide

This guide is for maintainers and contributors.

## Repository Layout

```text
cmd/database-mcp/
  main.go

internal/service/
  MCP tool registration
  argument handling
  pagination
  timeout handling
  connection lifecycle

internal/database/
  mysql/
  postgres/
  redis/
  sqlite/

internal/projectconfig/
  config discovery
  parsing
  placeholder expansion
  database config resolution

docs/
packages/npm/
scripts/
```

## Layer Responsibilities

### `cmd/database-mcp`

Keep it limited to:

- process startup
- version output
- stdio MCP serving

### `internal/service`

This is the MCP-facing orchestration layer.

It should handle:

- tool definitions
- argument validation
- pagination and timeout wiring
- formatting responses
- managing shared connection state

It should avoid:

- large parsing subsystems
- database-specific scan logic

### `internal/database`

This is the database implementation layer.

It should handle:

- connection construction
- query execution
- result scanning
- database-specific behavior

### `internal/projectconfig`

This layer exists specifically for project-based auto-detection.

It should handle:

- walking project directories
- choosing likely config files
- parsing supported formats
- resolving placeholders
- mapping config values into database connection models

It should not depend on MCP request objects.

## Contribution Principles

- keep tool behavior predictable for agent workflows
- keep outputs bounded and structured
- prefer explicit parameter names
- add tests when changing parsing or connection behavior
- update docs together with code

## Local Commands

```bash
gofmt -w ./cmd ./internal
go test ./...
go test -race ./...
go vet ./...
./scripts/build.sh
```

## Integration Test Environment Variables

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

If these are not set, the corresponding integration tests are skipped.

## Adding a New MCP Tool

Recommended sequence:

1. decide whether the logic belongs in `internal/database` or `internal/projectconfig`
2. add or update the lower-level API
3. expose it in `internal/service`
4. document parameters and expected behavior
5. add tests

## Adding New Config Detection Rules

When improving project config detection:

1. prefer generic patterns over framework-specific hacks
2. keep file parsing separate from database matching
3. add fixture-style tests for every new supported pattern
4. avoid silently guessing when confidence is low

## Documentation Policy

These changes must update documentation:

- new tools
- renamed tools
- changed tool arguments
- changed return fields
- new supported config sources
- changes to binary names or package names
