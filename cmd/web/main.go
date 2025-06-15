package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/HelloAnner/markdown-sync-confluence/cmd/web/api"
)

func main() {
	// 支持从命令行参数获取启动端口
	port := flag.String("port", "8080", "启动端口")
	flag.Parse()

	// 创建处理器
	downloadHandler := api.NewDownloadHandler()
	uploadHandler := api.NewUploadHandler()

	// 静态文件服务
	http.Handle("/", http.FileServer(http.Dir("web")))

	// 下载相关API
	http.HandleFunc("/api/name", downloadHandler.HandleGetName)
	http.HandleFunc("/api/convert", downloadHandler.HandleConvert)

	// 上传相关API
	http.HandleFunc("/api/upload", uploadHandler.HandleUpload)
	http.HandleFunc("/api/upload-file", uploadHandler.HandleUploadFile)
	http.HandleFunc("/api/optimize", uploadHandler.HandleOptimize)

	// 启动服务器
	log.Printf("Server starting on http://localhost:%s", *port)
	log.Printf("Download API: /api/name, /api/convert")
	log.Printf("Upload API: /api/upload, /api/upload-file, /api/optimize")
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal(err)
	}
} 