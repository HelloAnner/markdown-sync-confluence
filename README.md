# KMS Markdown Converter

一个用于将 KMS 页面转换为 Markdown 格式的工具。

## 使用 Docker 运行

### 方法 1：使用 docker-compose（推荐）

1. 确保已安装 Docker 和 docker-compose
2. 在项目根目录下运行：
   ```bash
   docker-compose up -d
   ```
3. 访问 http://localhost:8080

### 方法 2：直接使用 Docker

1. 构建镜像：
   ```bash
   docker build -t kms-markdown-converter .
   ```

   如果希望直接构建不同的平台,方便部署:
   ```
   docker buildx build --platform linux/amd64 -t kms2md-linux  .
   docker save -o kms2md.tar kms2md-linux:latest

   # 其他机器上
   docker load < kms2md.tar
   docker run -p 48061:8080 -d --name kms2md kms2md-linux
   ```


2. 运行容器：
   ```bash
   docker run -d -p 8080:8080 kms-markdown-converter
   ```

3. 访问 http://localhost:8080

## 开发环境

- Go 1.21+
- Vue.js 3
- Tailwind CSS

## 功能特性

- KMS 页面转换为 Markdown
- 在线预览和编辑
- 文件下载
- 用户认证
