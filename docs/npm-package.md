# npm 包维护说明

本文档说明 `packages/npm` 的职责、边界和维护要点。

## 这个目录到底做什么

`packages/npm` 不是主业务实现，它只是 Go 二进制的 npm 分发包装层。

它负责：

- 提供 `npx @mingcharun/database-mcp`
- 在安装时识别操作系统和架构
- 从 GitHub Release 下载对应二进制
- 把下载到的二进制包装成 npm 可执行入口

## 关键文件

- `packages/npm/package.json`
- `packages/npm/install.js`
- `packages/npm/bin/database-mcp.js`
- `packages/npm/README.md`

## 工作原理

### `package.json`

负责声明：

- 包名
- 版本
- bin 入口
- postinstall 钩子

### `install.js`

负责：

- 判断平台和架构
- 计算 release 文件名
- 从 GitHub Release 下载二进制
- 设置执行权限

### `bin/database-mcp.js`

负责：

- 作为 npm 的启动入口
- 找到已经下载好的二进制
- 将命令行参数透传给真实二进制

## 维护时最容易出错的地方

### 1. 版本号不同步

`package.json` 的版本必须和 GitHub release tag 对齐，否则安装脚本会去下载不存在的文件。

### 2. 产物命名不同步

如果 release 里的二进制命名变了，以下几处必须同步：

- `packages/npm/install.js`
- `packages/npm/bin/database-mcp.js`
- `scripts/install.sh`
- `docs/release.md`
- `README.md`

### 3. 仓库地址不同步

如果仓库地址变化，以下几处必须更新：

- `package.json`
- `install.js`
- `README.md`
- `docs/installation.md`

## 建议维护流程

1. 先保证 GitHub Release 构建成功
2. 再验证 release 资产命名
3. 再检查 npm 包版本
4. 最后发布 npm

## 最低验证项

发布前至少验证：

```bash
npx -y @mingcharun/database-mcp --version
```

如果这里失败，优先检查：

- npm 版本号
- release 是否存在
- 下载地址是否正确
- 平台文件名是否匹配
