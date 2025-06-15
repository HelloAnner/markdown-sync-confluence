package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

// DownloadHandler 处理下载相关的请求
type DownloadHandler struct{}

// NewDownloadHandler 创建下载处理器
func NewDownloadHandler() *DownloadHandler {
	return &DownloadHandler{}
}

// createConverter 为每个请求创建新的转换器
func (h *DownloadHandler) createConverter(username, password string) (*confluence.Converter, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	cfg := &config.Config{
		Confluence: config.ConfluenceConfig{
			URL:      "https://kms.fineres.com",
			Username: username,
			Password: password,
			Space:    "DR",
		},
	}

	return confluence.NewConverter(cfg), nil
}

// HandleGetName 通过ID获取文件名
func (h *DownloadHandler) HandleGetName(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	password := r.Header.Get("X-Password")
	
	converter, err := h.createConverter(username, password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	confluenceURL := r.URL.Query().Get("url")
	if confluenceURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}

	parts := strings.Split(confluenceURL, "=")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	pageID := parts[len(parts)-1]
	
	fileName, err := converter.GetFileName(pageID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file name: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"fileName": fileName,
	})
}

// HandleConvert 处理转换请求
func (h *DownloadHandler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	password := r.Header.Get("X-Password")
	
	converter, err := h.createConverter(username, password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取参数
	confluenceURL := r.URL.Query().Get("url")
	viewMode := r.URL.Query().Get("view") == "true"

	if confluenceURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}

	// 从 URL 中提取页面 ID
	parts := strings.Split(confluenceURL, "=")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	pageID := parts[len(parts)-1]

	// 获取页面内容并转换为 Markdown
	markdown, err := converter.ToMarkdown(pageID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert page: %v", err), http.StatusInternalServerError)
		return
	}

	// 根据模式返回不同的响应
	if viewMode {
		// 返回 JSON 格式的内容
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"content": markdown,
		})
	} else {
		// 返回文件下载
		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.md", pageID))
		w.Write([]byte(markdown))
	}
} 