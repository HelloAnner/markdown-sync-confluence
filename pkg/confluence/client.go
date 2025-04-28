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

// Client represents a Confluence API client
type Client struct {
	config     *config.Config
	httpClient *http.Client
}

// Page represents a Confluence page
type Page struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Version VersionInfo    `json:"version"`
	Links   map[string]string `json:"_links"`
}

// VersionInfo represents the version information of a page
type VersionInfo struct {
	Number int `json:"number"`
}

// NewClient creates a new Confluence client
func NewClient(config *config.Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{},
	}
}

// FindPageInParent finds a page by title in a parent page
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

// GetPageByID gets a page by its ID
func (c *Client) GetPageByID(pageID string) (*Page, error) {
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

// UpdatePage updates an existing page
func (c *Client) UpdatePage(pageID, title, body, spaceKey string) error {
	currentPage, err := c.GetPageByID(pageID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/rest/api/content/%s", c.config.Confluence.URL, pageID)

	// Prepare request body
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

// CreatePage creates a new page
func (c *Client) CreatePage(title, body, parentPageID string) (*Page, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content", c.config.Confluence.URL)

	// Prepare request body
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

// AttachFile uploads a file to a page
func (c *Client) AttachFile(pageID, filename string, content []byte, contentType string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", c.config.Confluence.URL, pageID)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	// Add comment field
	_ = writer.WriteField("comment", "Uploaded by markdown-sync-confluence")

	// Close multipart writer to set the terminating boundary
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// Build and execute the request
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.SetBasicAuth(c.config.Confluence.Username, c.config.Confluence.Password)

	// Check if file already exists, update it if it does
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

// NormalizeURL ensures that URLs have correct protocol and trailing slash
func NormalizeURL(rawURL string) string {
	// Add scheme if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	// Ensure trailing slash
	if !strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path += "/"
	}

	return parsedURL.String()
}

// GetAttachments gets all attachments for a page
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