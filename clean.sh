#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TARGETS=(
	"$ROOT_DIR/dist"
	"$ROOT_DIR/logs"
	"$ROOT_DIR/bin"
	"$ROOT_DIR/run"
)

echo "🧹 开始清理构建及日志目录..."

for path in "${TARGETS[@]}"; do
	if [[ -e "$path" ]]; then
		echo "🗑️ 删除 $path"
		rm -rf "$path"
	else
		echo "ℹ️ 跳过 $path (不存在)"
	fi
done

echo "✅ 清理完成"
