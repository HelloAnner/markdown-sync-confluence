package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// ContentHandler processes Markdown content and converts it to Confluence format
type ContentHandler struct {
	markdown goldmark.Markdown
	imagePlaceholders map[string]string
}

// NewContentHandler creates a new content handler
func NewContentHandler() *ContentHandler {
	// Configure Goldmark with needed extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Table,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML
		),
	)

	return &ContentHandler{
		markdown: md,
		imagePlaceholders: make(map[string]string),
	}
}

// ConvertToConfluence converts Markdown content to Confluence format
func (ch *ContentHandler) ConvertToConfluence(content string) (string, error) {
	// Process the content with preprocessors
	content = ch.preProcessMermaid(content)
	content = ch.preProcessFolding(content)
	content = ch.preProcessTaskLists(content)


	fmt.Println("content", content)
	// Convert Markdown to HTML
	var htmlContent strings.Builder
	if err := ch.markdown.Convert([]byte(content), &htmlContent); err != nil {
		return "", err
	}

	fmt.Println("htmlContent", htmlContent.String())

	// Post-process HTML for Confluence
	result := htmlContent.String()
	result = ch.postProcessCodeBlocks(result)
	result = ch.postProcessLinks(result)
	result = ch.postProcessMermaid(result)
	result = ch.postProcessFolding(result)
	result = ch.postProcessTables(result)
	result = ch.addTOCMacro(result)

	return result, nil
}

// preProcessMermaid handles mermaid code blocks
func (ch *ContentHandler) preProcessMermaid(content string) string {
	// Preserve and replace mermaid blocks with placeholders
	re := regexp.MustCompile("```mermaid\\s*\\n([\\s\\S]*?)```")
	return re.ReplaceAllStringFunc(content, func(match string) string {
		mermaidContent := re.FindStringSubmatch(match)[1]
		return "MERMAID_PLACEHOLDER:" + mermaidContent + ":MERMAID_PLACEHOLDER"
	})
}

// postProcessMermaid converts mermaid placeholders to Confluence macros
func (ch *ContentHandler) postProcessMermaid(content string) string {
	// Find all mermaid placeholders and convert them to Confluence Markdown macro
	re := regexp.MustCompile("MERMAID_PLACEHOLDER:([\\s\\S]*?):MERMAID_PLACEHOLDER")
	return re.ReplaceAllStringFunc(content, func(match string) string {
		mermaidContent := re.FindStringSubmatch(match)[1]
		// Escape CDATA end markers in content
		mermaidContent = escapeCDATA(mermaidContent)
		return `<ac:structured-macro ac:name="markdown">` +
			`<ac:plain-text-body><![CDATA[` +
			"```mermaid\n" + mermaidContent + "\n```" +
			`]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
}

// escapeCDATA escapes the CDATA end marker sequence ']]>' in content
func escapeCDATA(content string) string {
	// Replace ]]> with ]]&gt; to prevent it from breaking CDATA sections
	return strings.ReplaceAll(content, "]]>", "]]&gt;")
}

// preProcessFolding handles folding/collapsible sections
func (ch *ContentHandler) preProcessFolding(content string) string {
	// Match custom title fold blocks: ---Title--- content ---Title---
	// Go doesn't support backreferences in regex, so we need a different approach
	lines := strings.Split(content, "\n")
	result := []string{}
	inFoldBlock := false
	var currentTitle string
	var foldContent []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Check for fold start
		startMatch := regexp.MustCompile(`^---([^-\n]+?)---\s*$`).FindStringSubmatch(line)
		if !inFoldBlock && len(startMatch) > 1 {
			// Found start of fold block
			currentTitle = strings.TrimSpace(startMatch[1])
			inFoldBlock = true
			foldContent = []string{}
			continue
		}
		
		// Check for fold end with same title
		endMatch := regexp.MustCompile(`^---([^-\n]+?)---\s*$`).FindStringSubmatch(line)
		if inFoldBlock && len(endMatch) > 1 && strings.TrimSpace(endMatch[1]) == currentTitle {
			// Found end of fold block with matching title
			content := strings.Join(foldContent, "\n")
			result = append(result, "FOLD_PLACEHOLDER_TITLE:"+currentTitle+":CONTENT:"+content+":FOLD_PLACEHOLDER")
			inFoldBlock = false
			currentTitle = ""
			continue
		}
		
		// Inside fold block
		if inFoldBlock {
			foldContent = append(foldContent, line)
		} else {
			result = append(result, line)
		}
	}
	
	// Handle incomplete fold block
	if inFoldBlock {
		// Add back original lines if fold wasn't closed properly
		result = append(result, "---"+currentTitle+"---")
		result = append(result, foldContent...)
	}
	
	content = strings.Join(result, "\n")

	// Match old style fold blocks: ---折叠--- content ---折叠---
	// Use the same line-by-line approach
	lines = strings.Split(content, "\n")
	result = []string{}
	inFoldBlock = false
	foldContent = []string{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Check for fold start
		if !inFoldBlock && line == "---折叠---" {
			// Found start of fold block
			inFoldBlock = true
			foldContent = []string{}
			continue
		}
		
		// Check for fold end
		if inFoldBlock && line == "---折叠---" {
			// Found end of fold block
			content := strings.Join(foldContent, "\n")
			result = append(result, "FOLD_PLACEHOLDER_TITLE:点击展开:CONTENT:"+content+":FOLD_PLACEHOLDER")
			inFoldBlock = false
			continue
		}
		
		// Inside fold block
		if inFoldBlock {
			foldContent = append(foldContent, line)
		} else {
			result = append(result, line)
		}
	}
	
	// Handle incomplete fold block
	if inFoldBlock {
		// Add back original lines if fold wasn't closed properly
		result = append(result, "---折叠---")
		result = append(result, foldContent...)
	}
	
	return strings.Join(result, "\n")
}

// postProcessFolding converts folding placeholders to Confluence expand macros
func (ch *ContentHandler) postProcessFolding(content string) string {
	re := regexp.MustCompile("FOLD_PLACEHOLDER_TITLE:([^:]*?):CONTENT:([\\s\\S]*?):FOLD_PLACEHOLDER")
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		title := submatches[1]
		foldContent := submatches[2]
		
		// Convert the content to HTML (needed for nested content)
		var nestedHTML strings.Builder
		if err := ch.markdown.Convert([]byte(foldContent), &nestedHTML); err != nil {
			return match // Return original on error
		}
		
		// Escape CDATA end markers in the nested content
		nestedContent := escapeCDATA(nestedHTML.String())
		
		return `<ac:structured-macro ac:name="expand">` +
			`<ac:parameter ac:name="title">` + title + `</ac:parameter>` +
			`<ac:rich-text-body>` + nestedContent + `</ac:rich-text-body>` +
			`</ac:structured-macro>`
	})
}

// preProcessTaskLists handles task lists/checklists
func (ch *ContentHandler) preProcessTaskLists(content string) string {
	// Match task lists: - [ ] item or - [x] item
	re := regexp.MustCompile(`(?m)^- \[([ x])\] (.+)$`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		checked := submatches[1] == "x"
		text := submatches[2]
		
		status := "incomplete"
		if checked {
			status = "complete"
		}
		
		return "TASK_PLACEHOLDER_STATUS:" + status + ":TEXT:" + text + ":TASK_PLACEHOLDER"
	})
}

// postProcessCodeBlocks converts code blocks to Confluence code macros
func (ch *ContentHandler) postProcessCodeBlocks(content string) string {
	// Find code blocks with language specification
	re := regexp.MustCompile(`<pre><code class="language-([^"]+)">([\s\S]*?)</code></pre>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		language := submatches[1]
		code := submatches[2]
		
		// Unescape HTML entities in code
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		
		// Escape CDATA end markers
		code = escapeCDATA(code)
		
		return `<ac:structured-macro ac:name="code">` +
			`<ac:parameter ac:name="language">` + language + `</ac:parameter>` +
			`<ac:plain-text-body><![CDATA[` + code + `]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
	
	// Find code blocks without language specification
	reNoLang := regexp.MustCompile(`<pre><code>([\s\S]*?)</code></pre>`)
	content = reNoLang.ReplaceAllStringFunc(content, func(match string) string {
		submatches := reNoLang.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		code := submatches[1]
		
		// Unescape HTML entities in code
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		
		// Escape CDATA end markers
		code = escapeCDATA(code)
		
		return `<ac:structured-macro ac:name="code">` +
			`<ac:plain-text-body><![CDATA[` + code + `]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
	
	return content
}

// postProcessLinks handles special attributes in links
func (ch *ContentHandler) postProcessLinks(content string) string {
	// Process links to properly encode special characters
	re := regexp.MustCompile(`<a href="([^"]+)"`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		url := re.FindStringSubmatch(match)[1]
		
		// Ensure ampersands are properly encoded
		encodedURL := strings.ReplaceAll(url, "&amp;", "&")
		encodedURL = strings.ReplaceAll(encodedURL, "&", "&amp;")
		
		return `<a href="` + encodedURL + `"`
	})
}

// postProcessTables fixes table HTML for Confluence
func (ch *ContentHandler) postProcessTables(content string) string {
	// Ensure all tables have tbody
	content = regexp.MustCompile(`<table>\s*<tr>`).ReplaceAllString(content, "<table><tbody><tr>")
	content = regexp.MustCompile(`</tr>\s*</table>`).ReplaceAllString(content, "</tr></tbody></table>")
	
	// Fix BR tags in tables to be properly closed
	content = regexp.MustCompile(`<br(?:\s*/)?>`).ReplaceAllString(content, "<br/>")
	
	return content
}

// addTOCMacro adds table of contents macro if needed
func (ch *ContentHandler) addTOCMacro(content string) string {
	// Check if content has at least one heading
	if regexp.MustCompile(`<h[1-6]`).MatchString(content) {
		// Add TOC if not already present
		if !strings.Contains(content, "ac:name=\"toc\"") {
			tocMacro := `<ac:structured-macro ac:name="toc">` +
				`<ac:parameter ac:name="printable">true</ac:parameter>` +
				`<ac:parameter ac:name="style">disc</ac:parameter>` +
				`<ac:parameter ac:name="maxLevel">3</ac:parameter>` +
				`<ac:parameter ac:name="minLevel">1</ac:parameter>` +
				`</ac:structured-macro>`
			
			// Add TOC before the first heading
			firstHeadingIndex := regexp.MustCompile(`<h[1-6]`).FindStringIndex(content)[0]
			content = content[:firstHeadingIndex] + tocMacro + "\n" + content[firstHeadingIndex:]
		}
	}
	
	return content
} 