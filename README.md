# Markdown to Confluence Converter

[English](README_EN.md) | 简体中文

一个强大的工具，用于将 Markdown 文件转换并发布到 Confluence 页面。支持图片上传、代码块格式化、任务列表等多种 Markdown 特性。

## 特性

- 支持标准 Markdown 语法
- 支持 Obsidian 格式（如 `![[image]]` 语法）
- 自动处理和上传图片
- 自动生成目录
- 支持代码块高亮
- 支持任务列表
- 智能图片尺寸调整
- 支持配置文件和环境变量
- 命令行界面

## 安装

### 环境准备

```bash
# 安装 uv（推荐的 Python 包管理工具）
curl -LsSf https://astral.sh/uv/install.sh | sh

# 创建并激活虚拟环境
uv venv
source .venv/bin/activate  # Linux/macOS
# 或
.venv\Scripts\activate  # Windows
```

### 从源码安装

```bash
# 克隆仓库
git clone <repository-url>
cd markdown-to-confluence

# 使用 uv 安装依赖
uv sync
```

### 打包为可执行文件

```bash
# 运行打包脚本
uv run python3 build.py
```

打包后的可执行文件将在 `dist` 目录中生成：
- Windows: `dist/md2kms.exe`
- macOS/Linux: `dist/md2kms`

## 配置

### 方式一：配置文件

创建 `config.yml` 文件：

```yaml
confluence:
  url: "https://your-domain.atlassian.net"
  username: "your.email@domain.com"
  password: "your-api-token"
  space: "SPACEKEY"
  parent_page_id: "123456"  # 可选
```

### 方式二：环境变量

```bash
export KMS_URL="https://your-domain.atlassian.net"
export KMS_USERNAME="your.email@domain.com"
export KMS_PASSWORD="your-api-token"
export KMS_SPACE="SPACEKEY"
```

## 使用方法

### 基本用法

```bash
# 使用配置文件
uv run python3 -m markdown_to_confluence your-file.md --config config.yml

# 使用环境变量
uv run python3 -m markdown_to_confluence your-file.md

# 指定标题和父页面
uv run python3 -m markdown_to_confluence your-file.md "页面标题" "123456"
```

### 命令行参数

- 第一个参数: 要发布的 Markdown 文件路径
- 第二个参数: Confluence 页面标题（可选，默认使用文件名）
- 第三个参数: 父页面 ID（可选，如未指定则使用配置中的值）
- `--config`: 配置文件路径（可选，如未指定则使用环境变量）

### 图片处理

工具支持两种图片引用格式：

1. 标准 Markdown 格式：
```markdown
![alt text](path/to/image.png)
```

2. Obsidian 格式：
```markdown
![[image.png]]
```

图片文件可以放在：
- Markdown 文件同目录下
- `attachments` 子目录中
- 使用绝对路径
- 使用网络 URL

### 特殊功能

1. 自动目录生成：
   - 自动在页面开头添加目录
   - 支持多级标题
   - 可点击导航

2. 代码块处理：
   - 自动转换为 Confluence 代码宏
   - 保持代码格式和缩进

3. 任务列表：
   - 支持复选框语法
   - 自动转换为 Confluence 任务列表

4. 图片优化：
   - 自动调整过大的图片尺寸
   - 保持宽高比
   - 支持最大宽度和高度限制

## 项目意义

1. 提升效率：
   - 使用熟悉的 Markdown 编写文档
   - 一键发布到 Confluence
   - 自动处理格式转换

2. 标准化：
   - 统一的文档格式
   - 一致的图片处理
   - 规范的代码展示

3. 工作流优化：
   - 支持本地文档版本控制
   - 便于团队协作
   - 减少手动排版工作

4. 扩展性：
   - 支持自定义配置
   - 可集成到其他工具链
   - 支持批量处理

## 注意事项

1. 运行环境：
   - 使用 `uv` 作为包管理工具
   - 建议在虚拟环境中运行
   - 确保 Python 版本兼容性

2. 图片上传：
   - 建议将图片放在 `attachments` 目录中
   - 图片名称不要包含特殊字符
   - 支持的图片格式：PNG, JPG, JPEG, GIF

3. 安全性：
   - 不要在代码中硬编码认证信息
   - 建议使用环境变量或配置文件
   - API Token 需要妥善保管

4. 性能：
   - 大型文档可能需要较长处理时间
   - 图片上传速度取决于网络状况
   - 建议在本地测试后再发布

## 常见问题

1. 图片上传失败：
   - 检查文件路径是否正确
   - 确认图片格式是否支持
   - 验证网络连接状态

2. 认证错误：
   - 确认配置信息正确
   - 检查 API Token 是否有效
   - 验证用户权限

3. 格式问题：
   - 确保 Markdown 语法正确
   - 检查特殊字符转义
   - 验证图片引用格式

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建特性分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
