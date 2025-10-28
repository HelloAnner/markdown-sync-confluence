# 简化版 Makefile

.PHONY: build run clean help

build:
	cd cmd/web && go build -o main

run:
	./run.sh

clean:
	./clean.sh

help:
	@echo "用法: make [目标]"
	@echo ""
	@echo "目标:"
	@echo "  build   编译 Web 服务"
	@echo "  run     通过脚本启动/构建"
	@echo "  clean   清理所有日志与构建产物"
