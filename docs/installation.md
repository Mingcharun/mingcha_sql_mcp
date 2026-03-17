# 安装与接入

本文档面向使用者，重点解决三件事：

1. 怎么把 Database MCP 安装到本机或服务器
2. 怎么接入常见 MCP Client
3. 怎么验证它已经可用

## 先选适合你的安装方式

### 方式一：本地直接安装二进制

适合：

- 想最快开始使用
- 希望本机有固定可执行文件
- 用桌面 MCP Client 做长期接入

命令：

```bash
curl -fsSL https://raw.githubusercontent.com/Mingcharun/mingcha_sql_mcp/main/scripts/install.sh | bash
```

默认安装位置：

```text
~/go/bin/database-mcp
```

### 方式二：从源码构建

适合：

- 需要二次开发
- 想自己控制构建流程
- 希望在 CI 或内部环境中构建

命令：

```bash
git clone https://github.com/Mingcharun/mingcha_sql_mcp.git
cd mingcha_sql_mcp
./scripts/build.sh
```

构建产物：

```text
dist/database-mcp
```

### 方式三：通过 npm 直接运行

适合：

- 本机没有 Go 环境
- 希望通过 `npx` 快速接入
- 统一走 Node 侧分发

命令：

```bash
npx -y @mingcharun/database-mcp
```

## 按客户端接入

### Codex

直接使用本地二进制：

```toml
[mcp_servers.database_mcp]
command = "/absolute/path/to/database-mcp"
```

如果希望走 npm：

```toml
[mcp_servers.database_mcp]
command = "npx"
args = ["-y", "@mingcharun/database-mcp"]
```

### Claude Desktop

本地二进制方式：

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

npm 方式：

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

### 团队统一部署

适合：

- 一组开发者共用统一版本
- 团队内部有固定工作机或跳板机
- 希望避免每个人自行构建

建议：

- 固定二进制版本
- 把配置模板写进团队文档
- 分离测试环境和生产环境的 MCP 配置

## 安装后怎么验证

### 检查版本输出

```bash
/absolute/path/to/database-mcp --version
```

预期输出类似：

```text
Database MCP Server v1.0.0
Integrated: MySQL, PostgreSQL, Redis, SQLite
```

### 用 Inspector 看工具是否注册成功

```bash
npm install -g @modelcontextprotocol/inspector
mcp-inspector /absolute/path/to/database-mcp
```

如果能看到 MySQL、PostgreSQL、Redis、SQLite 对应工具列表，说明服务本身已经可用。

## 按使用场景给建议

### 场景一：本地开发调试

建议优先用源码构建：

- 便于联调
- 可以直接改代码
- 方便跑测试

### 场景二：桌面客户端长期使用

建议优先用固定路径的本地二进制：

- 启动更稳定
- 不依赖 npm 缓存
- 更容易排查路径问题

### 场景三：无 Go 环境的快速接入

建议优先用 npm：

- 启动快
- 适合临时接入
- 适合不想维护本地 Go 工具链的用户

### 场景四：团队内部统一版本

建议：

- 固定 release 版本
- 不依赖 `latest`
- 在升级前先跑回归验证

## 常见问题

### 1. 客户端提示找不到命令

优先检查：

- `command` 是否写成了绝对路径
- 二进制是否有执行权限
- npm 方式下网络是否正常

### 2. 服务能启动，但数据库工具调用失败

这通常不是 MCP 进程问题，而是数据库连接参数、网络或权限问题。建议：

1. 先验证 `--version`
2. 再在客户端里只调用连接工具
3. 成功后再查表、执行命令

### 3. npm 安装成功但运行失败

优先检查：

- 平台与架构是否受支持
- GitHub Release 是否存在对应二进制
- 本地网络是否能拉取 release 资源
