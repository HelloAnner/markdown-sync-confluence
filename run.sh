#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
LOG_DIR="$ROOT_DIR/logs"
DIST_DIR="$ROOT_DIR/dist/cli"
CLI_BINARY_NAME="md2kms"

require_go() {
	if ! command -v go >/dev/null 2>&1; then
		echo "❌ 未检测到 Go，请先安装 Go 1.21+"
		exit 1
	fi
}

prepare_directories() {
	mkdir -p "$BIN_DIR" "$LOG_DIR"
}

get_lan_ips() {
	local ips=()

	if command -v ip >/dev/null 2>&1; then
		while IFS= read -r ip_addr; do
			if [[ -n "$ip_addr" ]]; then
				ips+=("$ip_addr")
			fi
		done < <(ip -o -4 addr show scope global 2>/dev/null | awk '{print $4}' | cut -d'/' -f1)
	fi

	if (( ${#ips[@]} == 0 )) && command -v ifconfig >/dev/null 2>&1; then
		while IFS= read -r ip_addr; do
			if [[ -n "$ip_addr" ]]; then
				ips+=("$ip_addr")
			fi
		done < <(ifconfig 2>/dev/null | awk '/inet / && $2 !~ /^127\./ {print $2}')
	fi

	if (( ${#ips[@]} == 0 )) && command -v hostname >/dev/null 2>&1; then
		local host_ips
		host_ips=$(hostname -I 2>/dev/null || true)
		for ip_addr in $host_ips; do
			if [[ -n "$ip_addr" && "$ip_addr" != "127.0.0.1" ]]; then
				ips+=("$ip_addr")
			fi
		done
	fi

	printf '%s\n' "${ips[@]}"
}

start_web_service() {
	require_go
	prepare_directories

	echo "🛠️ 正在编译 Web 服务..."
	go build -o "$BIN_DIR/kms-web" ./cmd/web

	local port
	read -r -p "请输入 Web 服务端口 (默认 8080): " port
	if [[ -z "$port" ]]; then
		port="8080"
	fi

	if ! [[ "$port" =~ ^[0-9]+$ ]] || (( port < 1 || port > 65535 )); then
		echo "❌ 端口号无效，请输入 1-65535 的数字"
		exit 1
	fi

	local log_file="$LOG_DIR/web-${port}.log"
	local url="http://localhost:${port}"

	touch "$log_file"
	: >"$log_file"

	echo "🚀 正在后台启动 Web 服务 (端口: $port)"
	echo "🌐 本地访问地址: $url"
	echo "📝 日志写入: $log_file"

	nohup "$BIN_DIR/kms-web" --port "$port" >>"$log_file" 2>&1 </dev/null &
	local web_pid=$!

	sleep 1
	if ! kill -0 "$web_pid" >/dev/null 2>&1; then
		echo "❌ Web 服务启动失败，请检查日志 $log_file"
		exit 1
	fi

	echo "✅ Web 服务已启动 (PID: $web_pid)"

	local lan_ips=()
	while IFS= read -r addr; do
		if [[ -n "$addr" ]]; then
			lan_ips+=("$addr")
		fi
	done < <(get_lan_ips)

	if (( ${#lan_ips[@]} > 0 )); then
		echo "📡 局域网访问地址:"
		for addr in "${lan_ips[@]}"; do
			echo "  - http://$addr:$port"
		done
	else
		echo "⚠️ 未检测到局域网 IP，可使用本地地址访问"
	fi

	echo "ℹ️ 服务将持续运行，若需停止请执行: kill $web_pid"
}

package_cli() {
	require_go
	mkdir -p "$DIST_DIR"

	echo "🧹 清理旧的 CLI 构建产物..."
	rm -rf "$DIST_DIR"/*

	local targets=(
		"darwin amd64"
		"darwin arm64"
		"linux amd64"
		"linux arm64"
	)

	echo "📦 开始为以下平台构建 CLI:";
	for target in "${targets[@]}"; do
		echo "  - $target"
	done

	for target in "${targets[@]}"; do
		IFS=' ' read -r os arch <<<"$target"
		local artifact_name="${CLI_BINARY_NAME}-${os}-${arch}"
		local build_dir="$DIST_DIR/$artifact_name"
		mkdir -p "$build_dir"

		echo "⚙️ 构建 $artifact_name ..."
		CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o "$build_dir/$CLI_BINARY_NAME" ./cmd/command

		echo "🗜️ 打包 $artifact_name.tar.gz"
		(
			cd "$build_dir" && tar -czf "$DIST_DIR/$artifact_name.tar.gz" "$CLI_BINARY_NAME"
		)

		rm -rf "$build_dir"
		echo "✅ 完成 $artifact_name"
	done

	echo "🎉 所有 CLI 包已生成，位置: $DIST_DIR"
}

show_menu() {
	echo "======================================"
	echo " KMS Markdown Converter 启动助手"
	echo "======================================"
	echo "1) 启动 Web 服务 (后台运行)"
	echo "2) 打包 CLI (macOS/Linux 多架构)"
	echo "q) 退出"
	echo "======================================"
	read -r -p "请选择操作 [1/2/q]: " choice

	case "$choice" in
		1)
			start_web_service
			;;
		2)
			package_cli
			;;
		q|Q)
			echo "✅ 已退出"
			exit 0
			;;
		*)
			echo "❌ 无效选项"
			exit 1
			;;
	esac
}

show_menu
