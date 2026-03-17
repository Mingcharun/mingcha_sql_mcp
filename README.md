# Database MCP

Database MCP is an open-source MCP server for database access in AI agents, IDE assistants, and desktop MCP clients.

It provides one consistent MCP interface for:

- MySQL
- PostgreSQL
- Redis
- SQLite

The project is designed for a very practical reality: many teams do not keep database credentials in system environment variables. They keep them in project files such as `.env`, `application.yml`, `application.properties`, `config.json`, or `config.toml`. Database MCP can detect those configurations directly from a project directory and help the agent connect automatically.

## What Problem This Project Solves

When people say "let the AI inspect my database", there are usually two different problems hidden inside:

1. How does the AI get database access in a safe, structured way?
2. How does the AI know where the database credentials are stored?

Database MCP solves both:

- It exposes database operations as MCP tools.
- It can scan common project config files and detect database connection settings.

This makes it useful for PHP projects, Go services, Java and Spring applications, Node projects, Python backends, and mixed monorepos.

## Key Features

- Unified MCP interface for MySQL, PostgreSQL, Redis, and SQLite
- Query pagination and timeout controls for agent-friendly responses
- Connection lifecycle tools such as `connect`, `status`, and `disconnect`
- Project config detection from common config files
- Direct "connect from project" tools so the agent does not need to manually reconstruct credentials
- Open-source Go implementation with a clean `cmd / internal / docs / packages / scripts` layout

## How Automatic Calling Actually Works

This is the most important concept in the whole project:

Database MCP does not "watch your code" by itself.

The actual flow is:

1. Your MCP client starts Database MCP.
2. The AI sees that Database MCP exposes tools.
3. The AI decides which tool to call based on your request.
4. If you ask it to inspect a project database, it can first call:
   `project_detect_database_configs`
5. After it sees the detected config, it can call:
   `mysql_connect_from_project`
   `pgsql_connect_from_project`
   `redis_connect_from_project`
   or `sqlite_query_from_project`
6. Once connected, it can continue with normal query or metadata tools.

So the automation comes from the AI choosing MCP tools in sequence, not from the MCP process autonomously acting on the repository.

In other words:

- The AI reads intent from the conversation.
- The MCP exposes capabilities.
- The client executes the tool calls.
- Database MCP performs the actual detection and connection work.

## Supported Config Sources

Current project-based detection supports common config formats such as:

- `.env`
- `.env.local`
- `application.yml`
- `application.yaml`
- `application.properties`
- `config.json`
- `config.toml`

It also supports a number of common patterns:

- direct host / port / username / password fields
- DSN / URL style connection strings
- Spring style datasource configuration
- placeholder expansion such as `${DB_HOST}` and `${DB_PORT:5432}`

## Tool Coverage

| Category | Tools |
| --- | ---: |
| MySQL | 11 |
| PostgreSQL | 11 |
| Redis | 6 |
| SQLite | 2 |
| Project Config Detection | 1 |
| Total | 31 |

## Quick Start

### Build From Source

```bash
git clone https://github.com/Mingcharun/mingcha_sql_mcp.git
cd mingcha_sql_mcp
./scripts/build.sh
```

Binary output:

```text
dist/database-mcp
```

### Install With Script

```bash
curl -fsSL https://raw.githubusercontent.com/Mingcharun/mingcha_sql_mcp/main/scripts/install.sh | bash
```

### Run With npm

```bash
npx -y @mingcharun/database-mcp
```

## MCP Client Configuration

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

### npm-Based Configuration

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

## Typical Usage Flow

### If credentials are already known

Use the normal connection tools:

- `mysql_connect`
- `pgsql_connect`
- `redis_connect`
- `sqlite_query`

### If credentials live inside the project

Use the project-aware flow:

1. `project_detect_database_configs`
2. `mysql_connect_from_project` or `pgsql_connect_from_project` or `redis_connect_from_project`
3. query / metadata / write tools

For SQLite:

1. `project_detect_database_configs`
2. `sqlite_query_from_project`

## Repository Layout

```text
.
├── cmd/database-mcp/           # binary entrypoint
├── internal/service/           # MCP tool registration and handlers
├── internal/database/          # database implementations
├── internal/projectconfig/     # project config detection and parsing
├── docs/                       # project documentation
├── packages/npm/               # npm distribution wrapper
├── scripts/                    # build and install scripts
└── README.md                   # project entry document
```

## Documentation

Start here depending on your role:

- Installation and client setup: [`docs/installation.md`](docs/installation.md)
- Architecture and automatic tool calling: [`docs/architecture.md`](docs/architecture.md)
- Tool reference: [`docs/tool-reference.md`](docs/tool-reference.md)
- Development guide: [`docs/development.md`](docs/development.md)
- Release guide: [`docs/release.md`](docs/release.md)

## Validation

Common verification commands:

```bash
go test ./...
go test -race ./...
go vet ./...
./scripts/build.sh
```

## Design Principles

- Make database access usable for agents, not just humans
- Keep responses bounded and predictable
- Separate MCP orchestration from database implementation details
- Support real-world project config layouts
- Keep the repository clean, maintainable, and open-source friendly
