#!/usr/bin/env python3

import os
import sys
import argparse
from markdown_to_confluence.config import ConfigLoader
from markdown_to_confluence.confluence_client import ConfluenceClient
from markdown_to_confluence.content_handler import ContentHandler
from markdown_to_confluence.image_handler import ImageHandler
from markdown_to_confluence.preprocessor import MarkdownPreprocessor

class MarkdownToConfluence:
    def __init__(self, config_path=None, cli_config=None):
        """初始化转换器"""
        # 先加载命令行配置，确保最高优先级
        self.config = ConfigLoader.load_config(config_path, cli_config)
        self.confluence_client = ConfluenceClient(self.config)
        self.content_handler = ContentHandler()
        self.image_handler = ImageHandler(self.confluence_client.client, self.config)
        self.preprocessor = MarkdownPreprocessor()
        self.current_page_id = None

    def publish(self, markdown_file, title=None, parent_page_id=None):
        """发布Markdown内容到Confluence"""
        try:
            # 获取Markdown文件所在目录（用于解析相对图片路径）
            markdown_dir = os.path.dirname(os.path.abspath(markdown_file))
            
            # 读取Markdown内容
            with open(markdown_file, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # 预处理内容
            content = self.preprocessor.process(content)
            
            # 如果没有指定标题，使用文件名
            if not title:
                title = os.path.splitext(os.path.basename(markdown_file))[0]

            # 使用配置文件中的父页面ID（如果命令行没有指定）
            if not parent_page_id and 'parent_page_id' in self.config['confluence']:
                parent_page_id = self.config['confluence']['parent_page_id']
            
            if not parent_page_id:
                raise ValueError("必须指定父页面ID")

            # 在父页面下查找现有页面
            existing_page = self.confluence_client.find_page_in_parent(title, parent_page_id)
            
            if existing_page:
                self.current_page_id = existing_page['id']
            
            # 处理图片
            content = self.image_handler.process_images(
                content,
                markdown_dir,
                self.current_page_id if self.current_page_id else parent_page_id
            )
            
            # 转换为 Confluence 格式
            html_content = self.content_handler.convert_to_confluence(content)
            
            if existing_page:
                print(f"📝 正在覆盖更新页面: {title}...")
                self.confluence_client.update_page(
                    existing_page['id'],
                    title,
                    html_content,
                    self.config['confluence']['space']
                )
            else:
                # 创建新页面
                print(f"📝 正在父页面 {parent_page_id} 下创建新页面: {title}...")
                new_page = self.confluence_client.create_page(
                    title,
                    html_content,
                    parent_page_id
                )
                self.current_page_id = new_page['id']
                
        except Exception as e:
            print(f"❌ 发布失败: {str(e)}")
            sys.exit(1)

def main():
    """命令行入口"""
    parser = argparse.ArgumentParser(
        description='将 Markdown 文件发布到 Confluence',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
配置方式（按优先级从高到低）:
  1. 使用命令行参数:
     md2kms test.md --url https://your-domain.atlassian.net --username your.email@domain.com --password your-token --space SPACEKEY

  2. 使用环境变量:
     export KMS_URL=https://your-domain.atlassian.net
     export KMS_USERNAME=your.email@domain.com
     export KMS_PASSWORD=your-api-token
     export KMS_SPACE=SPACEKEY
     md2kms test.md

  3. 使用配置文件:
     md2kms test.md --config config.yml

示例:
  # 使用命令行参数
  md2kms test.md --url https://your-domain.atlassian.net --username your.email@domain.com --password your-token --space SPACEKEY --parent 123456

  # 使用文件名作为页面标题
  md2kms test.md --parent 123456

  # 指定页面标题
  md2kms test.md --title "我的文档" --parent 123456
"""
    )
    
    parser.add_argument(
        'markdown_file',
        help='要发布的 Markdown 文件路径'
    )
    
    parser.add_argument(
        '--title', '-t',
        help='Confluence 页面标题（默认使用文件名）'
    )
    
    parser.add_argument(
        '--parent', '-p',
        help='父页面 ID（如果未指定，将使用配置文件中的值）'
    )
    
    parser.add_argument(
        '--config', '-c',
        help='配置文件路径（如果未指定，将使用环境变量）'
    )

    # Confluence 配置参数
    parser.add_argument(
        '--url',
        help='Confluence URL (例如: https://your-domain.atlassian.net)'
    )
    
    parser.add_argument(
        '--username',
        help='Confluence 用户名/邮箱'
    )
    
    parser.add_argument(
        '--password',
        help='Confluence API Token'
    )
    
    parser.add_argument(
        '--space',
        help='Confluence Space Key'
    )
    
    args = parser.parse_args()
    
    try:
        # 创建命令行参数配置字典，只包含非None的值
        cli_config = {
            'confluence': {k: v for k, v in {
                'url': args.url,
                'username': args.username,
                'password': args.password,
                'space': args.space
            }.items() if v is not None}
        }
        
        converter = MarkdownToConfluence(args.config, cli_config)
        converter.publish(args.markdown_file, args.title, args.parent)
    except KeyboardInterrupt:
        print("\n⚠️ 操作已取消")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 错误: {str(e)}")
        sys.exit(1)

if __name__ == '__main__':
    main()