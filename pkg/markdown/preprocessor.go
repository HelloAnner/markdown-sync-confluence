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
	// Match YAML front matter pattern
	pattern := regexp.MustCompile(`^---\s*\n(.*?)\n---\s*\n`)
	
	if strings.HasPrefix(content, "---") {
		if match := pattern.FindStringSubmatch(content); len(match) > 0 {
			return content[len(match[0]):]
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
 