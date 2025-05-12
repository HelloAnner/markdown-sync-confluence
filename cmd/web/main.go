package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/HelloAnner/markdown-sync-confluence/pkg/config"
	"github.com/HelloAnner/markdown-sync-confluence/pkg/confluence"
)

// createConverter 为每个请求创建新的转换器
func createConverter(username, password string) (*confluence.Converter, error) {
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

func main() {
	// 支持从命令行参数获取启动端口
	port := flag.String("port", "8080", "启动端口")
	flag.Parse()

	// 静态文件服务
	http.Handle("/", http.FileServer(http.Dir("web")))

	// 通过ID获取文件名
	http.HandleFunc("/api/name", func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")
		password := r.Header.Get("X-Password")
		
		converter, err := createConverter(username, password)
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
	})

	// API 处理
	http.HandleFunc("/api/convert", func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-Username")
		password := r.Header.Get("X-Password")
		
		converter, err := createConverter(username, password)
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
	})

	// 启动服务器
	log.Printf("Server starting on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
} 