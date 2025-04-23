#!/usr/bin/env python3

import os
import sys
import shutil
import PyInstaller.__main__

# 常量定义
BINARY_NAME = 'md2kms'
CONFIG_FILE = 'config.yml'
SOURCE_DIR = 'src/markdown_to_confluence'
MAIN_SCRIPT = f'{SOURCE_DIR}/main.py'

def build():
    """构建独立的二进制文件"""
    # 清理之前的构建
    for dir_name in ['build', 'dist']:
        if os.path.exists(dir_name):
            shutil.rmtree(dir_name)

    # 确保配置文件会被包含在打包中
    config_path = os.path.join(SOURCE_DIR, CONFIG_FILE)
    if not os.path.exists(config_path):
        with open(config_path, 'w') as f:
            f.write("""confluence:
  url: https://your-domain.atlassian.net
  username: your.email@domain.com
  password: your-api-token
  space: SPACEKEY
  parent_page_id: null
""")

    # PyInstaller 参数
    args = [
        MAIN_SCRIPT,  # 主脚本
        f"--name={BINARY_NAME}",  # 输出文件名
        "--onefile",  # 打包成单个文件
        "--clean",  # 清理临时文件
        f"--add-data={config_path}:markdown_to_confluence",  # 包含配置文件
        "--hidden-import=PIL._tkinter",  # 确保 PIL 相关依赖被包含
        "--hidden-import=PIL._imagingtk",
    ]

    # 根据操作系统添加图标和其他选项
    if sys.platform.startswith('win'):
        args.append('--console')  # Windows下显示控制台
    elif sys.platform.startswith('darwin'):
        args.append('--console')  # macOS下显示终端

    # 运行 PyInstaller
    PyInstaller.__main__.run(args)

    print("\n✅ 构建完成!")
    if sys.platform.startswith('win'):
        binary_path = f'dist\\{BINARY_NAME}.exe'
    else:
        binary_path = f'dist/{BINARY_NAME}'

    print(f"二进制文件位置: {binary_path}")
    print("\n使用方法:")
    print("1. 将可执行文件复制到系统PATH目录")
    print(f"2. 运行命令: {BINARY_NAME} --help")

if __name__ == '__main__':
    build()
