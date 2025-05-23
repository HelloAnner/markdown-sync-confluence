package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
)

// Client 表示一个 Confluence API 客户端
type Client struct {
	config     *config.Config
	httpClient *http.Client
}

// Page 表示一个 Confluence 页面
type Page struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Version VersionInfo    `json:"version"`
	Links   map[string]string `json:"_links"`
}

// VersionInfo 表示一个页面的版本信息
type VersionInfo struct {
	Number int `json:"number"`
}

// SearchResult 表示搜索结果
type SearchResult struct {
	Results    []ContentResult `json:"results"`
	Start      int            `json:"start"`
	Limit      int            `json:"limit"`
	Size       int            `json:"size"`
	TotalSize  int            `json:"totalSize"`
}

// ContentResult 表示搜索返回的内容项
type ContentResult struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Excerpt string    `json:"excerpt"`
	Space   SpaceInfo   `json:"space"`
	Links   struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
	Version struct {
		Number    int    `json:"number"`
	} `json:"version"`
}

// SpaceInfo 表示空间信息
type SpaceInfo struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

// SearchOptions 定义搜索选项
type SearchOptions struct {
	SpaceKey string // 限定搜索的空间
	Type     string // 内容类型，如 "page", "blogpost" 等
	Start    int    // 分页起始位置
	Limit    int    // 每页结果数
}

// NewClient 创建一个新的 Confluence 客户端
func NewClient(config *config.Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{},
	}
}

// FindPageInParent 在父页面中查找一个页面
func (c *Client) FindPageInParent(title, parentPageID string) (*Page, error) {
	start := 0
	limit := 100 // 每页获取100个结果
	
	for {
		endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/page?limit=%d&start=%d", 
			c.config.Confluence.URL, parentPageID, limit, start)

		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("error finding page: %s - %s", resp.Status, string(body))
		}

		var result struct {
			Results []Page `json:"results"`
			Size    int    `json:"size"`     // 当前页面结果数
			Start   int    `json:"start"`    // 当前起始位置
			Limit   int    `json:"limit"`    // 每页限制
			Links   struct {
				Next string `json:"next"`    // 下一页链接
			} `json:"_links"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		// 在当前页面中查找目标页面
		for _, page := range result.Results {
			if page.Title == title {
				return &page, nil
			}
		}

		// 如果没有更多结果，退出循环
		if len(result.Results) < limit || result.Links.Next == "" {
			break
		}

		// 更新起始位置，继续获取下一页
		start += limit
	}

	return nil, nil
}

// GetPageInfoByID  按照ID获取页面基本信息 (不包括内容)
func (c *Client) GetPageInfoByID(pageID string) (*Page, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s?expand=version", c.config.Confluence.URL, pageID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting page: %s - %s", resp.Status, string(body))
	}

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &page, nil
}

// GetPageContentByID 获取页面的 HTML 内容
func (c *Client) GetPageContentByID(pageID string) (string, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage", c.config.Confluence.URL, pageID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error getting page content: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Body struct {
			Storage struct {
				Value string `json:"value"`
			} `json:"storage"`
		} `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	return result.Body.Storage.Value, nil
}

// UpdatePage 更新一个存在的页面
func (c *Client) UpdatePage(pageID, title, body, spaceKey string) error {
	currentPage, err := c.GetPageInfoByID(pageID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/rest/api/content/%s", c.config.Confluence.URL, pageID)

	bodyData := map[string]interface{}{
		"id":    pageID,
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": spaceKey},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          body,
				"representation": "storage",
			},
		},
		"version": map[string]int{
			"number": currentPage.Version.Number + 1,
		},
	}

	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error updating page: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("✅ Successfully updated page: %s\n", title)
	fmt.Printf("🔗 Page link: %s/pages/viewpage.action?pageId=%s\n", c.config.Confluence.URL, pageID)

	return nil
}

// CreatePage 创建一个新的页面
func (c *Client) CreatePage(title, body, parentPageID string) (*Page, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content", c.config.Confluence.URL)

	bodyData := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": c.config.Confluence.Space},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          body,
				"representation": "storage",
			},
		},
		"ancestors": []map[string]string{
			{
				"id": parentPageID,
			},
		},
	}

	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating page: %s - %s", resp.Status, string(body))
	}

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	fmt.Printf("✅ Successfully created page: %s\n", title)

	return &page, nil
}

// AttachFile  上传文件到页面
func (c *Client) AttachFile(pageID, filename string, content []byte, contentType string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", c.config.Confluence.URL, pageID)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	_ = writer.WriteField("comment", "Uploaded by markdown-sync-confluence")

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)

	req.Header.Set("X-Atlassian-Token", "nocheck")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error uploading file: %s - %s", resp.Status, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// NormalizeURL 确保 URL 具有正确的协议和尾部斜杠
func NormalizeURL(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL 
	}

	if !strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path += "/"
	}

	return parsedURL.String()
}

// GetAttachments 获取一个页面的所有附件
func (c *Client) GetAttachments(pageID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", c.config.Confluence.URL, pageID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting attachments: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Results []map[string]interface{} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

// SearchPages 使用关键词搜索页面
// 参数:
//   - query: 搜索关键词
//   - options: 搜索选项，包括空间、类型和分页参数
// 返回:
//   - *SearchResult: 搜索结果，包含页面ID和其他信息
//   - error: 错误信息
func (c *Client) SearchPages(query string, options *SearchOptions) (*SearchResult, error) {
	if options == nil {
		options = &SearchOptions{
			Start: 0,
			Limit: 25,
		}
	}

	// 构建 CQL 查询
	cql := fmt.Sprintf("text ~ \"%s\"", query)
	if options.SpaceKey != "" {
		cql += fmt.Sprintf(" AND space = \"%s\"", options.SpaceKey)
	}
	if options.Type != "" {
		cql += fmt.Sprintf(" AND type = \"%s\"", options.Type)
	}

	// 构建请求 URL
	endpoint := fmt.Sprintf("%s/rest/api/content/search?cql=%s&start=%d&limit=%d&expand=space,version,metadata",
		c.config.Confluence.URL,
		url.QueryEscape(cql),
		options.Start,
		options.Limit,
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating search request: %w", err)
	}

	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error searching pages: %s - %s", resp.Status, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding search response: %w", err)
	}

	return &result, nil
} 