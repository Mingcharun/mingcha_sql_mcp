# 发布流程

本文档面向维护者，说明如何构建、验证、发布和回归检查 Database MCP。

## 发布前先确认什么

至少确认以下几项：

1. `README.md` 与 `docs/` 已同步到当前实现
2. `packages/npm/package.json` 版本将与 release tag 对齐
3. 二进制命名、npm 下载逻辑和 release 工作流是一致的
4. 主要测试通过

## 发布前命令

```bash
go test ./...
go test -race ./...
go vet ./...
./scripts/build.sh
```

## 本地产物

默认构建产物：

```text
dist/database-mcp
```

## GitHub Actions 工作流

工作流文件：

```text
.github/workflows/release.yml
```

它负责：

1. 根据 tag 触发构建
2. 以 `./cmd/database-mcp` 为入口编译
3. 生成多平台二进制
4. 上传 release 产物
5. 发布 npm 包

## 当前 release 产物命名

- `database-mcp_darwin_arm64`
- `database-mcp_darwin_amd64`
- `database-mcp_linux_amd64`
- `database-mcp_windows_amd64.exe`
- `database-mcp_windows_arm64.exe`

如果这里改了，以下文件通常也要同步改：

- `scripts/install.sh`
- `packages/npm/install.js`
- `packages/npm/bin/database-mcp.js`
- `README.md`

## 推荐发布顺序

1. 确认代码与文档一致
2. 提交当前改动
3. 创建并推送版本 tag
4. 观察 GitHub Actions 是否成功
5. 检查 Release 页面资产
6. 验证 npm 是否能正常安装

## 发布后怎么验证

### 二进制验证

至少验证：

- macOS 一个架构
- Linux amd64
- Windows 一个架构

检查项：

- 能否下载
- 能否执行 `--version`
- MCP Client 能否正常拉起

### npm 验证

至少验证：

```bash
npx -y @mingcharun/database-mcp --version
```

检查项：

- 是否下载到正确平台二进制
- 是否能正常启动
- 错误信息是否可读

## 常见发布风险

### 风险一：二进制文件名变了，但 npm 没同步

症状：

- npm 安装成功
- 启动时报找不到二进制

### 风险二：tag 与 npm 版本不一致

症状：

- npm 下载到不存在的 release 资源

### 风险三：文档还在引用旧命名

症状：

- 用户能安装，但示例无法直接照抄
- 仓库页面仍然暴露旧品牌或旧路径
