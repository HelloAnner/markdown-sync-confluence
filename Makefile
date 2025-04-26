 # Makefile for cross-platform builds of md2kms

# 版本号
VERSION := $(shell git describe --tags --always || echo "dev")

# 基本设置
BINARY_NAME := md2kms
GO := go
GOFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

# 输出目录
OUT_DIR := build

# 各目标平台 GOOS/GOARCH 组合
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# 默认目标：编译当前平台
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

# 创建输出目录
$(OUT_DIR):
	mkdir -p $(OUT_DIR)

# 清理编译产物
.PHONY: clean
clean:
	rm -rf $(OUT_DIR)
	rm -f $(BINARY_NAME)

# 全平台编译
.PHONY: all
all: clean $(OUT_DIR) $(PLATFORMS)

# 为每个平台编译
.PHONY: $(PLATFORMS)
$(PLATFORMS): %: $(OUT_DIR)
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval SUFFIX := $(if $(filter windows,$(GOOS)),.exe,))
	@echo "编译 $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(SUFFIX) .

# 仅编译 macOS 版本
.PHONY: darwin
darwin: $(OUT_DIR) darwin/amd64 darwin/arm64

# 仅编译 Linux 版本
.PHONY: linux
linux: $(OUT_DIR) linux/amd64 linux/arm64

# 仅编译 Windows 版本
.PHONY: windows
windows: $(OUT_DIR) windows/amd64

# 帮助信息
.PHONY: help
help:
	@echo "可用的编译目标:"
	@echo "  make          - 编译当前平台的二进制文件"
	@echo "  make all      - 编译所有平台的二进制文件"
	@echo "  make darwin   - 仅编译 macOS 平台 (amd64, arm64)"
	@echo "  make linux    - 仅编译 Linux 平台 (amd64, arm64)"
	@echo "  make windows  - 仅编译 Windows 平台 (amd64)"
	@echo "  make clean    - 清理编译产物"