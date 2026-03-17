# Installation

This guide explains how to install Database MCP, connect it to MCP clients, and verify that it is working.

## Choose an Installation Method

### Install a Local Binary

Best for:

- long-term local use
- desktop MCP clients
- stable paths and predictable startup

```bash
curl -fsSL https://raw.githubusercontent.com/Mingcharun/mingcha_sql_mcp/main/scripts/install.sh | bash
```

Default install location:

```text
~/go/bin/database-mcp
```

### Build From Source

Best for:

- contributors
- internal builds
- custom patches

```bash
git clone https://github.com/Mingcharun/mingcha_sql_mcp.git
cd mingcha_sql_mcp
./scripts/build.sh
```

Output:

```text
dist/database-mcp
```

### Use npm

Best for:

- machines without a Go toolchain
- quick client setup
- teams that prefer Node-based distribution

```bash
npx -y @mingcharun/database-mcp
```

## Configure an MCP Client

### Codex

```toml
[mcp_servers.database_mcp]
command = "/absolute/path/to/database-mcp"
```

Using npm:

```toml
[mcp_servers.database_mcp]
command = "npx"
args = ["-y", "@mingcharun/database-mcp"]
```

### Claude Desktop

Binary-based setup:

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

npm-based setup:

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

## Verify the Installation

### Check Version Output

```bash
/absolute/path/to/database-mcp --version
```

Expected output:

```text
Database MCP Server v1.0.0
Integrated: MySQL, PostgreSQL, Redis, SQLite
```

### Check Tool Registration

```bash
npm install -g @modelcontextprotocol/inspector
mcp-inspector /absolute/path/to/database-mcp
```

If you can see the database tools, the MCP process itself is working.

## If Your Project Uses Config Files Instead of System Environment Variables

This is a first-class use case.

You can use:

- `project_detect_database_configs`
- `mysql_connect_from_project`
- `pgsql_connect_from_project`
- `redis_connect_from_project`
- `sqlite_query_from_project`

Typical flow:

1. Point the MCP tool at the project root
2. Let Database MCP detect config files
3. Connect using the project-aware tool
4. Continue with normal query or metadata tools

Supported common config sources:

- `.env`
- `.env.local`
- `application.yml`
- `application.yaml`
- `application.properties`
- `config.json`
- `config.toml`

## Recommended Setup by Scenario

### Local Development

Prefer source build if you are actively modifying the codebase.

### Daily Use in an IDE

Prefer a local binary with an absolute path.

### Quick Trial Without Go

Prefer npm.

### Team-Wide Internal Use

Prefer a fixed release version and a documented config template.

## Troubleshooting

### The client cannot find the command

Check:

- absolute path correctness
- executable permissions
- npm availability if using `npx`

### The MCP server starts, but database tools fail

That usually means the MCP process is healthy, but the database config or network path is not.

Recommended sequence:

1. verify `--version`
2. run config detection if the project stores credentials in files
3. run a connect tool
4. then query

### npm install works, but startup fails

Check:

- platform support
- GitHub release asset availability
- network access to release downloads
