package markdown

import (
	"regexp"
	"strings"
)

// Preprocessor handles front matter and other pre-processing steps
type Preprocessor struct{}

// NewPreprocessor creates a new preprocessor
func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

// Process applies all preprocessing steps to the markdown content
func (p *Preprocessor) Process(content string) string {
    content = p.StripFrontMatter(content)
    content = p.PreprocessURLs(content)
    return content
}

// StripFrontMatter removes YAML front matter from Markdown content
func (p *Preprocessor) StripFrontMatter(content string) string {
    // Normalize line endings for reliable processing
    normalized := strings.ReplaceAll(content, "\r\n", "\n")
    // Remove leading BOM if present
    if strings.HasPrefix(normalized, "\uFEFF") {
        normalized = strings.TrimPrefix(normalized, "\uFEFF")
    }

    // Allow optional leading blank lines before frontmatter
    lines := strings.Split(normalized, "\n")
    start := 0
    for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
        start++
    }

    // Frontmatter must start with a line that is exactly '---'
    if start < len(lines) && strings.TrimSpace(lines[start]) == "---" {
        // Find the closing '---' or '...' line
        end := start + 1
        for end < len(lines) {
            trimmed := strings.TrimSpace(lines[end])
            if trimmed == "---" || trimmed == "..." {
                break
            }
            end++
        }

        if end < len(lines) { // Found a terminator
            // Skip the closing line
            next := end + 1
            // Optionally skip a single blank line after frontmatter
            if next < len(lines) && strings.TrimSpace(lines[next]) == "" {
                next++
            }
            result := strings.Join(lines[next:], "\n")
            // Keep original line endings if they were CRLF in the input
            if strings.Contains(content, "\r\n") {
                return strings.ReplaceAll(result, "\n", "\r\n")
            }
            return result
        }
    }

    return content
}

// PreprocessURLs encodes special characters in URLs
func (p *Preprocessor) PreprocessURLs(content string) string {
	// Find Markdown links and encode & in URLs
	re := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		
		text := parts[1]
		url := parts[2]
		
		// Encode ampersands in URL
		url = strings.ReplaceAll(url, "&", "&amp;")
		
		return "[" + text + "](" + url + ")"
	})
} 
 
