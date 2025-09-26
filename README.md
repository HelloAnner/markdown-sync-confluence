# KMS Markdown Converter

一个用于将 KMS 页面转换为 Markdown 格式的工具。

## 快速开始

### 运行辅助脚本

仓库提供 `run.sh` 脚本来统一管理 Web 服务和 CLI 构建：

```bash
chmod +x run.sh   # 首次使用时设置执行权限
./run.sh
```

脚本提供以下选项：

- **1)** 启动 Web 服务（实时日志）。脚本会编译 `cmd/web`，询问端口号（默认为 8080），并在当前终端直接运行服务；所有输出同时写入 `logs/web-<port>.log`，方便在服务器环境中查看历史日志。脚本会打印可点击的访问链接（例如 `http://localhost:8080`），结束脚本（Ctrl+C）即会停止服务。
- **2)** 打包 CLI 工具。脚本会为 macOS (amd64/arm64) 与 Linux (amd64/arm64) 交叉编译 CLI，并生成压缩包到 `dist/cli` 目录。

### 清理构建产物

使用 `clean.sh` 可以清理 Web 编译产物和日志、CLI 压缩包等内容：

```bash
chmod +x clean.sh   # 首次使用时设置执行权限
./clean.sh
```

### 基础依赖

- Go 1.21+

## 目录简介

- `cmd/web`：Web 服务入口。
- `cmd/command`：命令行工具入口。
- `pkg`：核心业务逻辑与辅助模块。
- `web`：前端静态资源。
- `dist/cli`：脚本生成的 CLI 压缩包（执行后出现）。
- `logs`：Web 服务日志（执行后出现）。

## 功能特性

- KMS 页面转换为 Markdown
- 在线预览和编辑
- 文件下载
- 用户认证
