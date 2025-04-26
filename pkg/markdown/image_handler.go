package markdown

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

// ImageHandler handles the processing and uploading of images
type ImageHandler struct {
	client      *confluence.Client
	config      *config.Config
	pageID      string
	markdownDir string
	uploaded    map[string]string // Cache for uploaded images
	maxWidth    int
	maxHeight   int
	minScale    float64
}

// client: 客户端
// config: 配置
// pageID: 页面ID
// markdownDir: markdown文件目录
// uploaded: 上传的图片缓存
// maxWidth: 最大宽度
// maxHeight: 最大高度
func NewImageHandler(client *confluence.Client, config *config.Config) *ImageHandler {
	return &ImageHandler{
		client:    client,
		config:    config,
		uploaded:  make(map[string]string),
		maxWidth:  600,
		maxHeight: 400,
		minScale:  0.6,
	}
}

// ProcessImages 处理Markdown图片引用并上传到Confluence
func (h *ImageHandler) ProcessImages(content, markdownDir, pageID string) (string, error) {
	h.markdownDir = markdownDir
	h.pageID = pageID

	// 1. 首先处理HTML中的已转换的<img>标签
	imgRe := regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*\/?>`)
	content = imgRe.ReplaceAllStringFunc(content, func(match string) string {
		// 提取src属性
		srcMatches := regexp.MustCompile(`src="([^"]+)"`).FindStringSubmatch(match)
		if len(srcMatches) < 2 {
			return match
		}
		
		imgSrc := srcMatches[1]
		// 获取alt文本，如果有的话
		altText := ""
		altMatches := regexp.MustCompile(`alt="([^"]*)"`).FindStringSubmatch(match)
		if len(altMatches) >= 2 {
			altText = altMatches[1]
		}
		
		// 调用现有的处理函数
		return h.processImageReference(imgSrc, altText)
	})
	
	// 2. 处理任何剩余的标准Markdown图片 ![alt](path)
	content = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`).ReplaceAllStringFunc(content, h.replaceImage)
	
	// 3. 处理Obsidian-style图片 ![[path]]
	content = regexp.MustCompile(`!\[\[(.*?)\]\]`).ReplaceAllStringFunc(content, h.replaceObsidianImage)

	return content, nil
}

// replaceImage 处理标准Markdown图片格式
func (h *ImageHandler) replaceImage(match string) string {
	re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) < 3 {
		return match
	}

	altText := submatches[1]
	imagePath := submatches[2]

	return h.processImageReference(imagePath, altText)
}

// replaceObsidianImage 处理Obsidian-style图片格式
func (h *ImageHandler) replaceObsidianImage(match string) string {
	re := regexp.MustCompile(`!\[\[(.*?)\]\]`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) < 2 {
		return match
	}

	imagePath := submatches[1]
	return h.processImageReference(imagePath, "")
}

// processImageReference 处理实际的图片处理和替换
func (h *ImageHandler) processImageReference(imagePath, altText string) string {
	fullPath, size := h.processImagePath(imagePath)
	
	// 处理远程URL
	if strings.HasPrefix(fullPath, "http://") || strings.HasPrefix(fullPath, "https://") {
		// 对URL进行转义，确保所有的特殊字符（特别是&字符）都被正确处理
		escapedURL := escapeXMLAttributeValue(fullPath)
		
		if size > 0 {
			return fmt.Sprintf("<ac:image ac:width=\"%d\"><ri:url ri:value=\"%s\"/></ac:image>", 
				size, escapedURL)
		}
		return fmt.Sprintf("<ac:image><ri:url ri:value=\"%s\"/></ac:image>", escapedURL)
	}
	
	// 处理本地文件
	if _, err := os.Stat(fullPath); err == nil {
		imageURL, err := h.uploadImage(fullPath)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to upload image %s: %v\n", fullPath, err)
			return ""
		}
		
		// 对URL进行转义
		escapedURL := escapeXMLAttributeValue(imageURL)
		
		if size > 0 {
			return fmt.Sprintf("<ac:image ac:width=\"%d\"><ri:url ri:value=\"%s\"/></ac:image>", 
				size, escapedURL)
		}
		return fmt.Sprintf("<ac:image><ri:url ri:value=\"%s\"/></ac:image>", escapedURL)
	}
	
	// 图片未找到
	fmt.Printf("⚠️ Warning: Image file not found: %s\n", fullPath)
	return ""
}

// escapeXMLAttributeValue 转义XML属性值中的特殊字符
func escapeXMLAttributeValue(s string) string {
	// 替换XML中的特殊字符
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// processImagePath 处理图片路径并提取尺寸信息
func (h *ImageHandler) processImagePath(imagePath string) (string, int) {
	// 提取尺寸信息如果存在
	size := 0
	if strings.Contains(imagePath, "|") {
		parts := strings.Split(imagePath, "|")
		imagePath = parts[0]
		if len(parts) > 1 {
			if s, err := strconv.Atoi(parts[1]); err == nil {
				size = s
			}
		}
	}
	
	// 直接处理远程URL
	if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
		return imagePath, size
	}
	
	// 标准化路径分隔符
	imagePath = strings.ReplaceAll(imagePath, "\\", "/")
	
	// 如果它是绝对路径，直接使用它
	if filepath.IsAbs(imagePath) {
		return imagePath, size
	}
	
	// 尝试不同的可能路径，如Python版本
	possiblePaths := []string{
		// 1. 直接相对于markdown文件目录
		filepath.Join(h.markdownDir, imagePath),
		
		// 2. 在attachments子目录
		filepath.Join(h.markdownDir, "attachments", imagePath),
	}
	
	// 3. 如果路径包含attachments，从markdown目录重建
	if strings.Contains(imagePath, "/") {
		pathParts := strings.Split(imagePath, "/")
		if len(pathParts) > 0 {
			mdDirPath := filepath.Join(h.markdownDir, filepath.Join(pathParts...))
			possiblePaths = append(possiblePaths, mdDirPath)
		}
	}
	
	// 4. 处理../相对路径
	normPath := filepath.Clean(filepath.Join(h.markdownDir, imagePath))
	possiblePaths = append(possiblePaths, normPath)
	
	// Check each path
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, size
		}
	}
	
	// Print debug info like in Python version
	fmt.Printf("⚠️ Warning: Could not find image file: %s\n", imagePath)
	fmt.Println("Searched paths:")
	for _, path := range possiblePaths {
		fmt.Printf("- %s\n", path)
	}
	
	// Return default path (will likely fail later, but we keep the same behavior)
	return filepath.Join(h.markdownDir, imagePath), size
}

// getContentType determines the MIME type of a file
func (h *ImageHandler) getContentType(path string) string {
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "image/png" // Default to PNG if unknown
	}
	return contentType
}

// uploadImage uploads an image to Confluence
func (h *ImageHandler) uploadImage(imagePath string) (string, error) {
	// Check if we've already uploaded this image
	if url, exists := h.uploaded[imagePath]; exists {
		return url, nil
	}

	// Ensure we're working with absolute paths
	var absImagePath string
	if filepath.IsAbs(imagePath) {
		absImagePath = imagePath
	} else {
		var err error
		absImagePath, err = filepath.Abs(imagePath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	// Verify the image exists
	if _, err := os.Stat(absImagePath); err != nil {
		fmt.Printf("警告: 找不到图片 %s\n", absImagePath)
		return "", fmt.Errorf("image file not found: %w", err)
	}

	// Get the filename and content type
	filename := filepath.Base(absImagePath)
	contentType := h.getContentType(absImagePath)

	// Read the image file
	fileContent, err := os.ReadFile(absImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Upload to Confluence
	result, err := h.client.AttachFile(h.pageID, filename, fileContent, contentType)
	if err != nil {
		// Check if error is due to duplicate file name
		if strings.Contains(err.Error(), "Cannot add a new attachment with same file name") {
			fmt.Printf("⚠️ 注意: 图片 %s 已经存在，正在获取现有图片的URL\n", filename)
			
			// Get attachments for this page to find the existing image
			attachments, attachErr := h.client.GetAttachments(h.pageID)
			if attachErr != nil {
				return "", fmt.Errorf("failed to get attachments: %w", attachErr)
			}
			
			// Find the attachment with matching filename
			for _, attachment := range attachments {
				if title, ok := attachment["title"].(string); ok && title == filename {
					// Extract the download URL
					if links, ok := attachment["_links"].(map[string]interface{}); ok {
						if download, ok := links["download"].(string); ok {
							// Ensure URL is absolute
							imageURL := download
							if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
								baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
								imageURL = fmt.Sprintf("%s%s", baseURL, imageURL)
							}
							
							// Cache the URL (不转义原始缓存，转义发生在使用时)
							h.uploaded[imagePath] = imageURL
							fmt.Printf("✓ 使用现有图片: %s\n", filename)
							fmt.Printf("  图片URL: %s\n", imageURL)
							return imageURL, nil
						}
					}
				}
			}
			
			// If we get here, we couldn't find the matching attachment
			return "", fmt.Errorf("failed to find existing attachment: %s", filename)
		}
		
		// For other errors, return as usual
		fmt.Printf("警告: 上传图片失败: %v\n", err)
		return "", fmt.Errorf("failed to upload to Confluence: %w", err)
	}

	// Debug output for understanding response structure
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Debug: Attachment response: %s\n", jsonData)

	// 从响应中提取URL的多种尝试方式
	var imageURL string
	
	// 尝试方法1: 检查直接的 _links.download 字段
	if links, ok := result["_links"].(map[string]interface{}); ok {
		if download, ok := links["download"].(string); ok {
			imageURL = download
			fmt.Printf("Debug: Found URL in _links.download: %s\n", imageURL)
		}
	}
	
	// 尝试方法2: 检查 results 数组中的第一个结果
	if imageURL == "" && result["results"] != nil {
		if results, ok := result["results"].([]interface{}); ok && len(results) > 0 {
			if firstResult, ok := results[0].(map[string]interface{}); ok {
				if links, ok := firstResult["_links"].(map[string]interface{}); ok {
					if download, ok := links["download"].(string); ok {
						imageURL = download
						fmt.Printf("Debug: Found URL in results[0]._links.download: %s\n", imageURL)
					}
				}
			}
		}
	}
	
	// 尝试方法3: 检查self字段
	if imageURL == "" && result["self"] != nil {
		if self, ok := result["self"].(string); ok {
			// 将self URL转换为下载URL
			imageURL = strings.Replace(self, "/rest/api/", "/download/attachments/", 1)
			fmt.Printf("Debug: Constructed URL from self: %s\n", imageURL)
		}
	}
	
	// 尝试方法4: 检查其他可能的字段名
	if imageURL == "" {
		// 尝试url字段
		if url, ok := result["url"].(string); ok {
			imageURL = url
			fmt.Printf("Debug: Found URL in url field: %s\n", imageURL)
		}
		
		// 尝试downloadUrl字段
		if imageURL == "" {
			if downloadUrl, ok := result["downloadUrl"].(string); ok {
				imageURL = downloadUrl
				fmt.Printf("Debug: Found URL in downloadUrl field: %s\n", imageURL)
			}
		}
	}
	
	// 尝试方法5: 如果有ID，手动构建URL
	if imageURL == "" && result["id"] != nil {
		if _, ok := result["id"].(string); ok {
			baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
			imageURL = fmt.Sprintf("%s/download/attachments/%s/%s", baseURL, h.pageID, filename)
			fmt.Printf("Debug: Constructed URL from ID and filename: %s\n", imageURL)
		}
	}
	
	// 确保URL是绝对路径
	if imageURL != "" {
		if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
			// 确保基础URL没有多余的斜杠
			baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
			imageURL = fmt.Sprintf("%s%s", baseURL, imageURL)
		}
		
		// 缓存并返回URL (不转义原始缓存，转义发生在使用时)
		h.uploaded[imagePath] = imageURL
		fmt.Printf("✓ 成功上传图片: %s\n", filename)
		fmt.Printf("  图片URL: %s\n", imageURL)
		return imageURL, nil
	}
	
	// 如果仍然找不到URL，报错
	fmt.Printf("警告: 无法获取图片 %s 的链接信息，请检查API响应\n", filename)
	return "", fmt.Errorf("failed to get URL for uploaded image")
} 