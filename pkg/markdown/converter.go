package markdown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

// Converter orchestrates the conversion and publishing process
type Converter struct {
	config         *config.Config
	confluenceClient *confluence.Client
	contentHandler *ContentHandler
	imageHandler   *ImageHandler
	preprocessor   *Preprocessor
	currentPageID  string
}

// NewConverter creates a new markdown-to-confluence converter
func NewConverter(config *config.Config) *Converter {
	confluenceClient := confluence.NewClient(config)
	
	return &Converter{
		config:         config,
		confluenceClient: confluenceClient,
		contentHandler: NewContentHandler(),
		imageHandler:   NewImageHandler(confluenceClient, config),
		preprocessor:   NewPreprocessor(),
	}
}

// Publish converts and publishes a markdown file to Confluence
func (c *Converter) Publish(markdownFile, title, parentPageID string) error {
	// Get markdown file directory (for resolving relative image paths)
	markdownDir := filepath.Dir(markdownFile)
	
	// Read markdown content
	content, err := os.ReadFile(markdownFile)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}
	
	// Preprocess content
	processedContent := c.preprocessor.Process(string(content))
	
	// Use configured parent page ID if not specified via command line
	if parentPageID == "" && c.config.Confluence.ParentPageID != "" {
		parentPageID = c.config.Confluence.ParentPageID
	}
	
	if parentPageID == "" {
		return fmt.Errorf("parent page ID must be specified")
	}
	
	// Find existing page in parent
	existingPage, err := c.confluenceClient.FindPageInParent(title, parentPageID)
	if err != nil {
		fmt.Printf("⚠️ Warning: Error finding existing page: %s\n", err)
	}
	
	if existingPage != nil {
		c.currentPageID = existingPage.ID
	}
	
	// 1. 先转换文本为Confluence格式
	htmlContent, err := c.contentHandler.ConvertToConfluence(processedContent)
	if err != nil {
		return fmt.Errorf("failed to convert to confluence format: %w", err)
	}
	
	// 2. 再处理图片（此时页面ID已确定）
	pageID := c.currentPageID
	if pageID == "" {
		pageID = parentPageID
	}
	
	contentWithImages, err := c.imageHandler.ProcessImages(htmlContent, markdownDir, pageID)
	if err != nil {
		return fmt.Errorf("failed to process images: %w", err)
	}
	
	// Update or create page
	if existingPage != nil {
		fmt.Printf("📝 Updating existing page: %s...\n", title)
		err = c.confluenceClient.UpdatePage(
			existingPage.ID,
			title,
			contentWithImages,
			c.config.Confluence.Space,
		)
		if err != nil {
			return fmt.Errorf("failed to update page: %w", err)
		}
	} else {
		fmt.Printf("📝 Creating new page under parent %s: %s...\n", parentPageID, title)
		newPage, err := c.confluenceClient.CreatePage(
			title,
			contentWithImages,
			parentPageID,
		)
		if err != nil {
			return fmt.Errorf("failed to create page: %w", err)
		}
		c.currentPageID = newPage.ID
	}
	
	return nil
} 