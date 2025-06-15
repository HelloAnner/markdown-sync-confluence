package markdown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

// Converter 是Markdown到Confluence的转换和发布协调器
type Converter struct {
	config          *config.Config        // 应用配置
	confluenceClient *confluence.Client   // Confluence客户端
	contentHandler  *ContentHandler       // 内容处理器
	imageHandler    *ImageHandler         // 图片处理器
	preprocessor    *Preprocessor         // 预处理器
	currentPageID   string                // 当前正在处理的页面ID
}

// NewConverter 创建一个新的Markdown转Confluence转换器
// 参数:
//   - config: 应用配置
// 返回:
//   - *Converter: 转换器实例
func NewConverter(config *config.Config) *Converter {
	confluenceClient := confluence.NewClient(config)
	
	return &Converter{
		config:          config,
		confluenceClient: confluenceClient,
		contentHandler:  NewContentHandler(),
		imageHandler:    NewImageHandler(confluenceClient, config),
		preprocessor:    NewPreprocessor(),
	}
}

// Publish 将Markdown文件转换并发布到Confluence
// 参数:
//   - markdownFile: Markdown文件路径
//   - title: 页面标题
//   - parentPageID: 父页面ID
// 返回:
//   - error: 处理过程中的错误
func (c *Converter) Publish(markdownFile, title, parentPageID string) error {
	// 获取Markdown文件目录（用于解析相对图片路径）
	markdownDir := filepath.Dir(markdownFile)
	
	// 读取Markdown内容
	content, err := os.ReadFile(markdownFile)
	if err != nil {
		return fmt.Errorf("读取Markdown文件失败: %w", err)
	}
	
	// 预处理内容
	processedContent := c.preprocessor.Process(string(content))
	
	// 如果命令行未指定父页面ID，使用配置中的值
	if parentPageID == "" && c.config.Confluence.ParentPageID != "" {
		parentPageID = c.config.Confluence.ParentPageID
	}
	
	// 父页面ID必须指定
	if parentPageID == "" {
		return fmt.Errorf("必须指定父页面ID")
	}
	
	// 在父页面中查找现有页面
	existingPage, err := c.confluenceClient.FindPageInParent(title, parentPageID)
	if err != nil {
		fmt.Printf("⚠️ 警告: 查找现有页面时出错: %s\n", err)
	}
	
	// 如果页面已存在，记录其ID
	if existingPage != nil {
		c.currentPageID = existingPage.ID
	}
	
	// 1. 先转换文本为Confluence格式
	htmlContent, err := c.contentHandler.ConvertToConfluence(processedContent)
	if err != nil {
		return fmt.Errorf("转换为Confluence格式失败: %w", err)
	}
	
	// 2. 再处理图片（此时页面ID已确定）
	pageID := c.currentPageID
	if pageID == "" {
		pageID = parentPageID
	}
	
	// 处理图片引用并上传图片
	contentWithImages, err := c.imageHandler.ProcessImages(htmlContent, markdownDir, pageID)
	if err != nil {
		return fmt.Errorf("处理图片失败: %w", err)
	}
	
	// 3. 更新或创建页面
	if existingPage != nil {
		// 更新现有页面
		fmt.Printf("📝 正在更新页面: %s...\n", title)
		err = c.confluenceClient.UpdatePage(
			existingPage.ID,
			title,
			contentWithImages,
			c.config.Confluence.Space,
		)
		if err != nil {
			return fmt.Errorf("更新页面失败: %w", err)
		}
		fmt.Printf("✅ 页面更新成功: %s\n", title)
		fmt.Printf("🔗 页面链接: %s/pages/viewpage.action?pageId=%s\n", c.config.Confluence.URL, existingPage.ID)
	} else {
		// 创建新页面
		fmt.Printf("📝 正在父页面 %s 下创建新页面: %s...\n", parentPageID, title)
		newPage, err := c.confluenceClient.CreatePage(
			title,
			contentWithImages,
			parentPageID,
		)
		if err != nil {
			return fmt.Errorf("创建页面失败: %w", err)
		}
		c.currentPageID = newPage.ID
		fmt.Printf("✅ 页面创建成功: %s\n", title)
		fmt.Printf("🔗 页面链接: %s/pages/viewpage.action?pageId=%s\n", c.config.Confluence.URL, newPage.ID)
	}
	
	return nil
}

// PublishContent 将Markdown内容转换并发布到Confluence
// 参数:
//   - content: Markdown内容字符串
//   - title: 页面标题
//   - parentPageID: 父页面ID
// 返回:
//   - error: 处理过程中的错误
func (c *Converter) PublishContent(content, title, parentPageID string) error {
	// 预处理内容
	processedContent := c.preprocessor.Process(content)
	
	// 如果命令行未指定父页面ID，使用配置中的值
	if parentPageID == "" && c.config.Confluence.ParentPageID != "" {
		parentPageID = c.config.Confluence.ParentPageID
	}
	
	// 父页面ID必须指定
	if parentPageID == "" {
		return fmt.Errorf("必须指定父页面ID")
	}
	
	// 在父页面中查找现有页面
	existingPage, err := c.confluenceClient.FindPageInParent(title, parentPageID)
	if err != nil {
		fmt.Printf("⚠️ 警告: 查找现有页面时出错: %s\n", err)
	}
	
	// 如果页面已存在，记录其ID
	if existingPage != nil {
		c.currentPageID = existingPage.ID
	}
	
	// 1. 先转换文本为Confluence格式
	htmlContent, err := c.contentHandler.ConvertToConfluence(processedContent)
	if err != nil {
		return fmt.Errorf("转换为Confluence格式失败: %w", err)
	}
	
	// 2. 再处理图片（此时页面ID已确定）
	pageID := c.currentPageID
	if pageID == "" {
		pageID = parentPageID
	}
	
	// 处理图片引用并上传图片（对于内容字符串，使用空的markdownDir）
	contentWithImages, err := c.imageHandler.ProcessImages(htmlContent, "", pageID)
	if err != nil {
		return fmt.Errorf("处理图片失败: %w", err)
	}
	
	// 3. 更新或创建页面
	if existingPage != nil {
		// 更新现有页面
		fmt.Printf("📝 正在更新页面: %s...\n", title)
		err = c.confluenceClient.UpdatePage(
			existingPage.ID,
			title,
			contentWithImages,
			c.config.Confluence.Space,
		)
		if err != nil {
			return fmt.Errorf("更新页面失败: %w", err)
		}
		fmt.Printf("✅ 页面更新成功: %s\n", title)
		fmt.Printf("🔗 页面链接: %s/pages/viewpage.action?pageId=%s\n", c.config.Confluence.URL, existingPage.ID)
	} else {
		// 创建新页面
		fmt.Printf("📝 正在父页面 %s 下创建新页面: %s...\n", parentPageID, title)
		newPage, err := c.confluenceClient.CreatePage(
			title,
			contentWithImages,
			parentPageID,
		)
		if err != nil {
			return fmt.Errorf("创建页面失败: %w", err)
		}
		c.currentPageID = newPage.ID
		fmt.Printf("✅ 页面创建成功: %s\n", title)
		fmt.Printf("🔗 页面链接: %s/pages/viewpage.action?pageId=%s\n", c.config.Confluence.URL, newPage.ID)
	}
	
	return nil
} 