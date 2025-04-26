package markdown

import (
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

// ImageHandler 处理图片的上传和处理
type ImageHandler struct {
	client      *confluence.Client // Confluence客户端
	config      *config.Config     // 应用配置
	pageID      string             // 当前页面ID
	markdownDir string             // Markdown文件所在目录
	uploaded    map[string]string  // 已上传图片的缓存，键为本地路径，值为Confluence URL
	maxWidth    int                // 图片最大宽度
	maxHeight   int                // 图片最大高度
	minScale    float64            // 最小缩放比例
}

// NewImageHandler 创建一个新的图片处理器
// 参数:
//   - client: Confluence客户端
//   - config: 应用配置
// 返回:
//   - *ImageHandler: 图片处理器实例
func NewImageHandler(client *confluence.Client, config *config.Config) *ImageHandler {
	return &ImageHandler{
		client:    client,
		config:    config,
		uploaded:  make(map[string]string),
		maxWidth:  600, // 默认最大宽度
		maxHeight: 400, // 默认最大高度
		minScale:  0.6, // 默认最小缩放比例
	}
}

// ProcessImages 处理HTML内容中的图片并上传到Confluence
// 参数:
//   - content: 要处理的HTML内容
//   - markdownDir: Markdown文件所在目录（用于解析相对路径）
//   - pageID: Confluence页面ID
// 返回:
//   - string: 处理后的内容
//   - error: 处理过程中的错误
func (h *ImageHandler) ProcessImages(content, markdownDir, pageID string) (string, error) {
	// 设置上下文信息
	h.markdownDir = markdownDir
	h.pageID = pageID

	// 1. 处理HTML中的<img>标签
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
		
		// 处理图片引用
		return h.processImageReference(imgSrc, altText)
	})
	
	// 2. 处理Markdown格式的图片引用 ![alt](path)
	content = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`).ReplaceAllStringFunc(content, h.replaceImage)
	
	// 3. 处理Obsidian格式的图片引用 ![[path]]
	content = regexp.MustCompile(`!\[\[(.*?)\]\]`).ReplaceAllStringFunc(content, h.replaceObsidianImage)

	return content, nil
}

// replaceImage 处理标准Markdown格式的图片引用
// 参数:
//   - match: 匹配到的Markdown图片字符串，格式为![alt](path)
// 返回:
//   - string: 替换后的Confluence XML格式图片标签
func (h *ImageHandler) replaceImage(match string) string {
	re := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) < 3 {
		return match // 格式不匹配则返回原始内容
	}

	altText := submatches[1]   // 图片替代文本
	imagePath := submatches[2] // 图片路径

	return h.processImageReference(imagePath, altText)
}

// replaceObsidianImage 处理Obsidian格式的图片引用
// 参数:
//   - match: 匹配到的Obsidian图片字符串，格式为![[path]]
// 返回:
//   - string: 替换后的Confluence XML格式图片标签
func (h *ImageHandler) replaceObsidianImage(match string) string {
	re := regexp.MustCompile(`!\[\[(.*?)\]\]`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) < 2 {
		return match // 格式不匹配则返回原始内容
	}

	imagePath := submatches[1] // 图片路径
	return h.processImageReference(imagePath, "") // Obsidian格式没有alt文本
}

// processImageReference 处理图片引用并生成Confluence XML
// 参数:
//   - imagePath: 图片路径（可能是相对路径或URL）
//   - altText: 图片替代文本
// 返回:
//   - string: Confluence XML格式的图片标签
func (h *ImageHandler) processImageReference(imagePath, altText string) string {
	// 处理图片路径并获取尺寸信息
	fullPath, size := h.processImagePath(imagePath)
	
	// 处理远程URL
	if strings.HasPrefix(fullPath, "http://") || strings.HasPrefix(fullPath, "https://") {
		// 转义URL中的XML特殊字符
		escapedURL := escapeXMLAttributeValue(fullPath)
		
		// 根据是否有尺寸生成适当的XML
		if size > 0 {
			return fmt.Sprintf("<ac:image ac:width=\"%d\"><ri:url ri:value=\"%s\"/></ac:image>", 
				size, escapedURL)
		}
		return fmt.Sprintf("<ac:image><ri:url ri:value=\"%s\"/></ac:image>", escapedURL)
	}
	
	// 处理本地文件
	if _, err := os.Stat(fullPath); err == nil {
		// 上传图片到Confluence
		imageURL, err := h.uploadImage(fullPath)
		if err != nil {
			fmt.Printf("⚠️ 警告: 图片上传失败 %s: %v\n", fullPath, err)
			return ""
		}
		
		// 转义URL中的XML特殊字符
		escapedURL := escapeXMLAttributeValue(imageURL)
		
		// 根据是否有尺寸生成适当的XML
		if size > 0 {
			return fmt.Sprintf("<ac:image ac:width=\"%d\"><ri:url ri:value=\"%s\"/></ac:image>", 
				size, escapedURL)
		}
		return fmt.Sprintf("<ac:image><ri:url ri:value=\"%s\"/></ac:image>", escapedURL)
	}
	
	// 图片文件未找到
	fmt.Printf("⚠️ 警告: 图片文件未找到: %s\n", fullPath)
	return ""
}

// escapeXMLAttributeValue 转义XML属性值中的特殊字符
// 参数:
//   - s: 需要转义的字符串
// 返回:
//   - string: 转义后的字符串
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
// 参数:
//   - imagePath: 图片路径（可能包含尺寸信息）
// 返回:
//   - string: 处理后的完整图片路径
//   - int: 图片尺寸（如果指定了的话）
func (h *ImageHandler) processImagePath(imagePath string) (string, int) {
	// 提取尺寸信息（如果存在）
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
	
	// 如果是绝对路径，直接使用
	if filepath.IsAbs(imagePath) {
		return imagePath, size
	}
	
	// 尝试多种可能的相对路径
	possiblePaths := []string{
		// 1. 直接相对于markdown文件目录
		filepath.Join(h.markdownDir, imagePath),
		
		// 2. 在attachments子目录
		filepath.Join(h.markdownDir, "attachments", imagePath),
	}
	
	// 3. 如果路径包含/，尝试从markdown目录重建完整路径
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
	
	// 检查每个可能的路径
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, size // 返回第一个存在的路径
		}
	}
	
	// 打印调试信息，提示所有尝试过的路径
	fmt.Printf("⚠️ 警告: 图片文件未找到: %s\n", imagePath)
	fmt.Println("尝试过的路径:")
	for _, path := range possiblePaths {
		fmt.Printf("- %s\n", path)
	}
	
	// 返回默认路径（后续会处理失败）
	return filepath.Join(h.markdownDir, imagePath), size
}

// getContentType 确定文件的MIME类型
// 参数:
//   - path: 文件路径
// 返回:
//   - string: 文件的MIME类型
func (h *ImageHandler) getContentType(path string) string {
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "image/png" // 默认为PNG
	}
	return contentType
}

// uploadImage 上传图片到Confluence
// 参数:
//   - imagePath: 图片文件的本地路径
// 返回:
//   - string: 上传后的图片URL
//   - error: 上传过程中的错误
func (h *ImageHandler) uploadImage(imagePath string) (string, error) {
	// 检查缓存中是否已有此图片
	if url, exists := h.uploaded[imagePath]; exists {
		return url, nil
	}

	// 确保使用绝对路径
	var absImagePath string
	if filepath.IsAbs(imagePath) {
		absImagePath = imagePath
	} else {
		var err error
		absImagePath, err = filepath.Abs(imagePath)
		if err != nil {
			return "", fmt.Errorf("获取绝对路径失败: %w", err)
		}
	}

	// 确认图片文件存在
	if _, err := os.Stat(absImagePath); err != nil {
		fmt.Printf("⚠️ 警告: 图片未找到 %s\n", absImagePath)
		return "", fmt.Errorf("图片文件未找到: %w", err)
	}

	// 获取文件名和内容类型
	filename := filepath.Base(absImagePath)
	contentType := h.getContentType(absImagePath)

	// 读取图片文件内容
	fileContent, err := os.ReadFile(absImagePath)
	if err != nil {
		return "", fmt.Errorf("读取图片文件失败: %w", err)
	}

	// 上传到Confluence
	result, err := h.client.AttachFile(h.pageID, filename, fileContent, contentType)
	if err != nil {
		// 处理重复文件名错误
		if strings.Contains(err.Error(), "Cannot add a new attachment with same file name") {
			fmt.Printf("ℹ️ 提示: 图片 %s 已存在，正在获取现有图片的URL\n", filename)
			
			// 获取当前页面的所有附件
			attachments, attachErr := h.client.GetAttachments(h.pageID)
			if attachErr != nil {
				return "", fmt.Errorf("获取附件列表失败: %w", attachErr)
			}
			
			// 查找匹配文件名的附件
			for _, attachment := range attachments {
				if title, ok := attachment["title"].(string); ok && title == filename {
					// 提取下载URL
					if links, ok := attachment["_links"].(map[string]interface{}); ok {
						if download, ok := links["download"].(string); ok {
							// 确保URL是绝对路径
							imageURL := download
							if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
								baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
								imageURL = fmt.Sprintf("%s%s", baseURL, imageURL)
							}
							
							// 缓存URL
							h.uploaded[imagePath] = imageURL
							fmt.Printf("✓ 使用现有图片: %s\n", filename)
							fmt.Printf("  图片URL: %s\n", imageURL)
							return imageURL, nil
						}
					}
				}
			}
			
			// 找不到匹配的附件
			return "", fmt.Errorf("未找到现有附件: %s", filename)
		}
		
		// 其他上传错误
		fmt.Printf("⚠️ 警告: 上传图片失败: %v\n", err)
		return "", fmt.Errorf("上传到Confluence失败: %w", err)
	}

	// 从响应中提取URL
	var imageURL string
	
	// 方法1: 检查_links.download字段
	if links, ok := result["_links"].(map[string]interface{}); ok {
		if download, ok := links["download"].(string); ok {
			imageURL = download
		}
	}
	
	// 方法2: 检查results数组中的第一个结果
	if imageURL == "" && result["results"] != nil {
		if results, ok := result["results"].([]interface{}); ok && len(results) > 0 {
			if firstResult, ok := results[0].(map[string]interface{}); ok {
				if links, ok := firstResult["_links"].(map[string]interface{}); ok {
					if download, ok := links["download"].(string); ok {
						imageURL = download
					}
				}
			}
		}
	}
	
	// 方法3: 尝试从ID和文件名构建URL
	if imageURL == "" && result["id"] != nil {
		if _, ok := result["id"].(string); ok {
			baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
			imageURL = fmt.Sprintf("%s/download/attachments/%s/%s", baseURL, h.pageID, filename)
		}
	}
	
	// 确保URL是绝对路径
	if imageURL != "" {
		if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
			baseURL := strings.TrimSuffix(h.config.Confluence.URL, "/")
			imageURL = fmt.Sprintf("%s%s", baseURL, imageURL)
		}
		
		// 缓存并返回URL
		h.uploaded[imagePath] = imageURL
		fmt.Printf("✓ 图片上传成功: %s\n", filename)
		fmt.Printf("  图片URL: %s\n", imageURL)
		return imageURL, nil
	}
	
	// 找不到URL
	fmt.Printf("⚠️ 警告: 无法获取图片 %s 的URL，请检查API响应\n", filename)
	return "", fmt.Errorf("无法获取已上传图片的URL")
} 