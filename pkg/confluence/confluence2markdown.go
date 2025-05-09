package confluence

import (
	"fmt"
	"os"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
)

type Converter struct {
	config           *config.Config
	confluenceClient *Client
	contentHandler   *ContentHandler
}

func NewConverter(config *config.Config) *Converter {
	confluenceClient := NewClient(config)
	contentHandler := NewContentHandler()
	return &Converter{config: config, confluenceClient: confluenceClient, contentHandler: contentHandler}
}

func (c *Converter) Download(searchWord string ) error {

	// 搜索
	searchOptions := &SearchOptions{
		SpaceKey: "DR",
		Type:     "page",
		Start:    0,
		Limit:    200,
	}

	searchResult, err := c.confluenceClient.SearchPages(searchWord, searchOptions)
	if err != nil {
		fmt.Printf("❌ Error: %s\n", err)
		return err
	}

	for _, page := range searchResult.Results {
		pageContent, err := c.confluenceClient.GetPageContentByID(page.ID)
		if err != nil {
			fmt.Printf("❌ Error: %s\n", err)
			return err
		}

		markdownContent, err := c.contentHandler.ConvertToMarkdown(pageContent)
		if err != nil {
			fmt.Printf("❌ Error: %s\n", err)
			return err
		}

		// 创建目录
		dir := "docs"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}

		// 保存到当前目录
		// 文件名简单编码,去除/
		fileName := strings.ReplaceAll(page.Title, "/", "_") + ".md"
		err = os.WriteFile(dir + "/" + fileName, []byte(markdownContent), 0644)
		if err != nil {
			fmt.Printf("❌ Error: %s\n", err)
			return err
		}
	}

	return nil
}
