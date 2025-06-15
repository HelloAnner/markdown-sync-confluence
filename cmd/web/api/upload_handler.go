package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/ai"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/markdown"
)

// UploadHandler 处理上传相关的请求
type UploadHandler struct{}

// NewUploadHandler 创建上传处理器
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// createMarkdownConverter 为每个请求创建新的markdown转换器
func (h *UploadHandler) createMarkdownConverter(username, password string) (*markdown.Converter, error) {
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

	return markdown.NewConverter(cfg), nil
}

// UploadRequest 上传请求的结构体
type UploadRequest struct {
	Content      string `json:"content"`      // Markdown内容
	Title        string `json:"title"`        // 页面标题
	ParentPageID string `json:"parentPageId"` // 父页面ID
}

// UploadResponse 上传响应的结构体
type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	PageID  string `json:"pageId,omitempty"`
	PageURL string `json:"pageUrl,omitempty"`
}

// OptimizeRequest AI优化请求的结构体
type OptimizeRequest struct {
	Content string `json:"content"` // Markdown内容
	Prompt  string `json:"prompt"`  // AI提示词
}

// OptimizeResponse AI优化响应的结构体
type OptimizeResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	OptimizedContent  string `json:"optimizedContent,omitempty"`
}

// HandleUpload 处理上传请求
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取认证信息
	username := r.Header.Get("X-Username")
	password := r.Header.Get("X-Password")
	
	converter, err := h.createMarkdownConverter(username, password)
	if err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 解析请求内容
	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// 验证必要字段
	if req.Content == "" {
		h.sendErrorResponse(w, "Content is required", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		h.sendErrorResponse(w, "Title is required", http.StatusBadRequest)
		return
	}

	if req.ParentPageID == "" {
		h.sendErrorResponse(w, "Parent page ID is required", http.StatusBadRequest)
		return
	}

	// 发布到Confluence
	err = h.publishToConfluence(converter, req.Content, req.Title, req.ParentPageID)
	if err != nil {
		h.sendErrorResponse(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	response := UploadResponse{
		Success: true,
		Message: "Page published successfully",
		PageURL: fmt.Sprintf("https://kms.fineres.com/pages/viewpage.action?pageId=%s", req.ParentPageID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUploadFile 处理文件上传请求
func (h *UploadHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取认证信息
	username := r.Header.Get("X-Username")
	password := r.Header.Get("X-Password")
	
	converter, err := h.createMarkdownConverter(username, password)
	if err != nil {
		h.sendErrorResponse(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 解析multipart form
	err = r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		h.sendErrorResponse(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		h.sendErrorResponse(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 验证文件类型
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".md") {
		h.sendErrorResponse(w, "Only .md files are allowed", http.StatusBadRequest)
		return
	}

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		h.sendErrorResponse(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// 获取其他参数
	title := r.FormValue("title")
	parentPageID := r.FormValue("parentPageId")

	// 如果没有指定标题，使用文件名
	if title == "" {
		title = filepath.Base(header.Filename)
		extension := filepath.Ext(title)
		title = title[0 : len(title)-len(extension)]
	}

	// 验证必要字段
	if parentPageID == "" {
		h.sendErrorResponse(w, "Parent page ID is required", http.StatusBadRequest)
		return
	}

	// 发布到Confluence
	err = h.publishToConfluence(converter, string(content), title, parentPageID)
	if err != nil {
		h.sendErrorResponse(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	response := UploadResponse{
		Success: true,
		Message: "File uploaded and published successfully",
		PageURL: fmt.Sprintf("https://kms.fineres.com/pages/viewpage.action?pageId=%s", parentPageID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// publishToConfluence 发布内容到Confluence
func (h *UploadHandler) publishToConfluence(converter *markdown.Converter, content, title, parentPageID string) error {
	// 创建临时文件来模拟文件上传
	// 注意：这里我们直接传递内容字符串，而不是创建实际文件
	// 需要修改markdown.Converter的Publish方法以支持直接传递内容
	
	// 由于原始的Publish方法需要文件路径，我们需要创建一个支持直接内容的版本
	// 这里暂时使用现有的方法结构，但需要在pkg/markdown中添加新的方法
	
	return converter.PublishContent(content, title, parentPageID)
}

// sendErrorResponse 发送错误响应
func (h *UploadHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := UploadResponse{
		Success: false,
		Message: message,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleOptimize 处理AI优化请求
func (h *UploadHandler) HandleOptimize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取认证信息（虽然AI优化不需要Confluence认证，但为了保持一致性）
	username := r.Header.Get("X-Username")
	password := r.Header.Get("X-Password")
	
	if username == "" || password == "" {
		h.sendOptimizeErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// 解析请求内容
	var req OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendOptimizeErrorResponse(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// 验证必要字段
	if req.Content == "" {
		h.sendOptimizeErrorResponse(w, "Content is required", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		h.sendOptimizeErrorResponse(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// 调用AI优化
	optimizedContent, err := ai.ChatWithPrompt(req.Content, req.Prompt)
	if err != nil {
		h.sendOptimizeErrorResponse(w, fmt.Sprintf("AI optimization failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	response := OptimizeResponse{
		Success:          true,
		Message:          "Content optimized successfully",
		OptimizedContent: optimizedContent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendOptimizeErrorResponse 发送AI优化错误响应
func (h *UploadHandler) sendOptimizeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := OptimizeResponse{
		Success: false,
		Message: message,
	}
	
	json.NewEncoder(w).Encode(response)
} 