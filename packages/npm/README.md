# `@mingcharun/database-mcp`

这是 Database MCP 的 npm 分发包。

它适合这些场景：

- 本机没有 Go 环境
- 希望用 `npx` 直接启动
- 团队内部统一通过 npm 分发 MCP 可执行文件

## 快速使用

```bash
npx -y @mingcharun/database-mcp
```

## 在 MCP Client 中配置

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

## 它不会做什么

这个 npm 包不会在本地编译 Go 源码。

它会在安装阶段：

1. 读取 npm 包版本
2. 判断操作系统和架构
3. 从 GitHub Release 下载对应二进制
4. 通过 `bin/database-mcp.js` 启动真实可执行文件

## 维护文档

如果你在维护这个包，而不是在使用它，请继续阅读：

- [`docs/npm-package.md`](../../docs/npm-package.md)
- [`docs/release.md`](../../docs/release.md)
