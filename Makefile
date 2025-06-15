# Makefile for cross-platform builds of md2kms

# 版本信息
VERSION := $(shell git describe --tags --always || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date +%Y%m%d-%H%M%S)

# 基本设置
BINARY_NAME := md2kms
GO := go
GOFLAGS := -trimpath -buildmode=pie
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildTime=$(BUILD_TIME) \
	-extldflags '-static'

# 输出目录
OUT_DIR := dist

# 目标平台定义
PLATFORMS := darwin/amd64 darwin/arm64 \
	linux/amd64 linux/arm64 \w
	windows/amd64

# 默认目标
.PHONY: build all darwin linux windows clean help

build:
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .

$(OUT_DIR):
	@mkdir -p $(OUT_DIR)

clean:
	@rm -rf $(OUT_DIR)
	@rm -f $(BINARY_NAME)*

all: clean $(OUT_DIR) $(PLATFORMS)

$(PLATFORMS): %: $(OUT_DIR)
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval SUFFIX := $(if $(filter windows,$(GOOS)),.exe,))
	@echo ">> 编译 $(GOOS)/$(GOARCH)..."
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(OUT_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(SUFFIX) .

# 平台组编译
darwin: $(OUT_DIR) darwin/amd64 darwin/arm64
linux: $(OUT_DIR) linux/amd64 linux/arm64 linux/arm/v7
windows: $(OUT_DIR) windows/amd64

# 帮助信息
help:
	@echo "用法: make [目标]"
	@echo ""
	@echo "目标:"
	@echo "  build       编译当前平台 (默认目标)"
	@echo "  all         编译所有平台"
	@echo "  darwin      编译macOS平台 (Intel/Apple Silicon)"
	@echo "  linux       编译Linux平台 (amd64/arm64/armv7)"
	@echo "  windows     编译Windows平台 (amd64)"
	@echo "  clean       清理编译产物"
	@echo ""
	@echo "编译选项:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  BUILD_TIME=$(BUILD_TIME)"
	@echo ""
	@echo "输出目录: $(OUT_DIR)/"