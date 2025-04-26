# Markdown to Confluence Sync

A tool to convert and publish Markdown files to Confluence pages.

## Features

- Convert Markdown files to Confluence format
- Automatically handle images (upload and reference)
- Support for code highlighting
- Support for Mermaid diagrams
- Support for collapsible/folding sections
- Support for task lists
- Table of Contents generation

## Installation

### From Source

```bash
git clone https://github.com/HelloAnner/markdown-sync-confluence.git
cd markdown-sync-confluence
go build
```

## Usage

### Basic Usage

```bash
# Using file name as the page title
./markdown-sync-confluence test.md --url https://your-domain.atlassian.net --username your.email@domain.com --password your-api-token --space SPACEKEY --parent 123456

# Specifying a custom page title
./markdown-sync-confluence test.md --title "My Custom Title" --url https://your-domain.atlassian.net --username your.email@domain.com --password your-api-token --space SPACEKEY --parent 123456
```

### Configuration Options

1. Command line arguments (highest priority):
   ```
   --url, -u        Confluence URL
   --username       Confluence username/email
   --password       Confluence API token
   --space          Confluence space key
   --parent, -p     Parent page ID
   --title, -t      Page title (defaults to file name)
   --config, -c     Path to config file
   ```

2. Environment variables:
   ```
   export KMS_URL=https://your-domain.atlassian.net
   export KMS_USERNAME=your.email@domain.com
   export KMS_PASSWORD=your-api-token
   export KMS_SPACE=SPACEKEY
   ```

3. Configuration file (lowest priority):
   Create a `config.yml` file with the following structure:
   ```yaml
   confluence:
     url: 'https://your-domain.atlassian.net'
     username: 'your.email@domain.com'
     password: 'your-api-token'
     space: 'SPACEKEY'
     parent_page_id: '123456'
   ```

   Then run:
   ```
   ./markdown-sync-confluence test.md --config config.yml
   ```

## Special Markdown Features

### Mermaid Diagrams

```
​```mermaid
graph TD
    A[Start] --> B{Is it?}
    B -->|Yes| C[OK]
    B -->|No| D[End]
​```
```

### Collapsible Sections

```
---Title---
Content here
---Title---
```

### Task Lists

```
- [ ] Incomplete task
- [x] Completed task
```

## License

MIT
