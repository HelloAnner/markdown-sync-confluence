# 简化版 Makefile

.PHONY: build docker clean

build:
	cd cmd/web && go build -o main

docker:
	docker-compose up --build -d

clean:
	rm -f cmd/web/main

help:
	@echo "用法: make [目标]"
	@echo ""
	@echo "目标:"
	@echo "  build   编译项目"
	@echo "  docker  构建并启动 Docker 容器"
	@echo "  clean   清理编译产物"