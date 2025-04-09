# Markdown to Confluence Converter

A powerful tool for converting Markdown files to Confluence pages. Supports image uploads, code block formatting, task lists, and various Markdown features.

## Features

- Standard Markdown syntax support
- Obsidian format support (e.g., `![[image]]` syntax)
- Automatic image processing and upload
- Automatic table of contents generation
- Code block highlighting
- Task list support
- Smart image size adjustment
- Configuration file and environment variable support
- Command-line interface

## Installation

### Environment Setup

```bash
# Install uv (recommended Python package manager)
curl -LsSf https://astral.sh/uv/install.sh | sh

# Create and activate virtual environment
uv venv
source .venv/bin/activate  # Linux/macOS
# or
.venv\Scripts\activate  # Windows
```

### Install from Source

```bash
# Clone repository
git clone <repository-url>
cd markdown-to-confluence

# Install dependencies using uv
uv pip3 install -e .
```

### Build Executable

```bash
# Install build tools using uv
uv pip3 install pyinstaller

# Run build script
uv run python3 build.py
```

The executable will be generated in the `dist` directory:
- Windows: `dist/md2kms.exe`
- macOS/Linux: `dist/md2kms`

## Configuration

### Option 1: Configuration File

Create a `config.yml` file:

```yaml
confluence:
  url: "https://your-domain.atlassian.net"
  username: "your.email@domain.com"
  password: "your-api-token"
  space: "SPACEKEY"
  parent_page_id: "123456"  # Optional
```

### Option 2: Environment Variables

```bash
export KMS_URL="https://your-domain.atlassian.net"
export KMS_USERNAME="your.email@domain.com"
export KMS_PASSWORD="your-api-token"
export KMS_SPACE="SPACEKEY"
```

## Usage

### Basic Usage

```bash
# Using configuration file
uv run python3 -m markdown_to_confluence your-file.md --config config.yml

# Using environment variables
uv run python3 -m markdown_to_confluence your-file.md

# Specify title and parent page
uv run python3 -m markdown_to_confluence your-file.md "Page Title" "123456"
```

### Command Line Arguments

- First argument: Path to the Markdown file
- Second argument: Confluence page title (optional, defaults to file name)
- Third argument: Parent page ID (optional, uses config value if not specified)
- `--config`: Path to configuration file (optional, uses environment variables if not specified)

### Image Handling

The tool supports two image reference formats:

1. Standard Markdown format:
```markdown
![alt text](path/to/image.png)
```

2. Obsidian format:
```markdown
![[image.png]]
```

Image files can be located in:
- Same directory as the Markdown file
- `attachments` subdirectory
- Absolute path
- Network URL

### Special Features

1. Automatic Table of Contents:
   - Added automatically at the beginning of the page
   - Supports multiple heading levels
   - Clickable navigation

2. Code Block Processing:
   - Automatic conversion to Confluence code macro
   - Preserves formatting and indentation

3. Task Lists:
   - Supports checkbox syntax
   - Converts to Confluence task lists

4. Image Optimization:
   - Automatic resizing of large images
   - Maintains aspect ratio
   - Supports maximum width and height limits

## Project Benefits

1. Efficiency Improvement:
   - Write documents in familiar Markdown
   - One-click publishing to Confluence
   - Automatic format conversion

2. Standardization:
   - Unified document format
   - Consistent image handling
   - Standardized code presentation

3. Workflow Optimization:
   - Local document version control
   - Team collaboration friendly
   - Reduced manual formatting work

4. Extensibility:
   - Custom configuration support
   - Integration with other toolchains
   - Batch processing support

## Important Notes

1. Runtime Environment:
   - Uses `uv` as package manager
   - Recommended to run in virtual environment
   - Ensure Python version compatibility

2. Image Upload:
   - Recommended to place images in `attachments` directory
   - Avoid special characters in image names
   - Supported formats: PNG, JPG, JPEG, GIF

3. Security:
   - Don't hardcode authentication information
   - Use environment variables or config file
   - Properly secure API Token

4. Performance:
   - Large documents may require longer processing time
   - Image upload speed depends on network conditions
   - Test locally before publishing

## Common Issues

1. Image Upload Failures:
   - Check file path correctness
   - Verify supported image format
   - Check network connectivity

2. Authentication Errors:
   - Verify configuration information
   - Check API Token validity
   - Verify user permissions

3. Format Issues:
   - Ensure correct Markdown syntax
   - Check special character escaping
   - Verify image reference format

## Contributing

Issues and Pull Requests are welcome!

1. Fork the project
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT License 