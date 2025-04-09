#!/usr/bin/env python3

import os
import sys
import re
from pathlib import Path
from atlassian import Confluence
import markdown2
import yaml
import mimetypes
import base64
import warnings
import urllib3
import uuid
import time
from PIL import Image
import argparse
import html

# 忽略 urllib3 的 SSL 警告
warnings.filterwarnings('ignore', category=urllib3.exceptions.InsecureRequestWarning)

class MarkdownToConfluence:
    def __init__(self, config_path=None):
        """初始化转换器"""
        self.config = self._load_config(config_path)
        self.confluence = self._init_confluence()
        self.uploaded_images = {}  # 缓存已上传的图片
        self.current_page_id = None  # 当前页面的ID
        self.max_image_width = 600  # 最大图片宽度
        self.max_image_height = 400  # 最大图片高度
        self.min_scale_ratio = 0.6  # 最小缩放比例

    def _load_config(self, config_path=None):
        """加载配置文件或环境变量"""
        config = {
            'confluence': {
                'url': None,
                'username': None,
                'password': None,
                'space': None,
                'parent_page_id': None
            }
        }

        # 如果指定了配置文件，从文件加载
        if config_path:
            try:
                with open(config_path, 'r', encoding='utf-8') as f:
                    file_config = yaml.safe_load(f)
                    if file_config and 'confluence' in file_config:
                        config.update(file_config)
            except Exception as e:
                print(f"⚠️ 警告: 无法读取配置文件 {config_path}: {str(e)}")
                print("将尝试使用环境变量...")
        
        # 如果没有指定配置文件或配置文件加载失败，尝试从环境变量读取
        if not config_path or not all([
            config['confluence']['url'],
            config['confluence']['username'],
            config['confluence']['password'],
            config['confluence']['space']
        ]):
            # 从环境变量读取配置
            config['confluence'].update({
                'url': os.environ.get('KMS_URL'),
                'username': os.environ.get('KMS_USERNAME'),
                'password': os.environ.get('KMS_PASSWORD'),
                'space': os.environ.get('KMS_SPACE')
            })

        # 验证必要的配置项
        required_keys = ['url', 'username', 'password', 'space']
        missing_keys = [
            key for key in required_keys 
            if not config['confluence'].get(key)
        ]
        
        if missing_keys:
            raise ValueError(
                f"缺少必要的配置项: {', '.join(missing_keys)}\n"
                "请在配置文件中设置这些值，或设置对应的环境变量:\n"
                "KMS_URL, KMS_USERNAME, KMS_PASSWORD, KMS_SPACE"
            )

        return config

    def _init_confluence(self):
        """初始化Confluence客户端"""
        try:
            return Confluence(
                url=self.config['confluence']['url'],
                username=self.config['confluence']['username'],
                password=self.config['confluence']['password'],
                verify_ssl=False
            )
        except Exception as e:
            raise Exception(f"连接Confluence失败: {str(e)}")

    def _get_image_content_type(self, image_path):
        """获取图片的MIME类型"""
        content_type = mimetypes.guess_type(image_path)[0]
        if not content_type:
            # 默认使用 png
            content_type = 'image/png'
        return content_type

    def _generate_temp_title(self, filename):
        """生成唯一的临时页面标题"""
        timestamp = int(time.time())
        random_id = str(uuid.uuid4())[:8]
        return f"temp_{timestamp}_{random_id}_{filename}"

    def _upload_image(self, image_path, space_key):
        """上传图片到Confluence并返回图片URL"""
        if image_path in self.uploaded_images:
            return self.uploaded_images[image_path]

        try:
            # 处理图片路径
            if os.path.isabs(image_path):
                # 绝对路径
                abs_image_path = image_path
            else:
                # 相对路径
                abs_image_path = os.path.abspath(image_path)

            if not os.path.exists(abs_image_path):
                print(f"警告: 找不到图片 {abs_image_path}")
                return None

            # 获取文件名
            filename = os.path.basename(abs_image_path)

            # 上传图片
            if self.current_page_id:
                # 如果页面已存在，直接附加到页面
                try:
                    with open(abs_image_path, 'rb') as f:
                        image_data = f.read()
                    
                    attachment = self.confluence.attach_content(
                        content=image_data,
                        name=filename,
                        content_type=self._get_image_content_type(abs_image_path),
                        page_id=self.current_page_id
                    )
                    # 构建图片URL
                    image_url = f"{self.config['confluence']['url']}/download/attachments/{self.current_page_id}/{filename}"
                except Exception as e:
                    print(f"警告: 上传图片到现有页面失败: {str(e)}")
                    return None
            else:
                # 如果页面不存在，先创建一个临时页面
                temp_title = self._generate_temp_title(filename)
                try:
                    temp_page = self.confluence.create_page(
                        space=space_key,
                        title=temp_title,
                        body="Temporary page for image upload",
                        parent_id=None,
                        type='page',
                        representation='storage'
                    )

                    with open(abs_image_path, 'rb') as f:
                        image_data = f.read()
                    
                    attachment = self.confluence.attach_content(
                        content=image_data,
                        name=filename,
                        content_type=self._get_image_content_type(abs_image_path),
                        page_id=temp_page['id']
                    )
                    # 构建图片URL
                    image_url = f"{self.config['confluence']['url']}/download/attachments/{temp_page['id']}/{filename}"

                    # 缓存上传的图片URL
                    if image_url:
                        self.uploaded_images[image_path] = image_url
                finally:
                    # 确保无论如何都删除临时页面
                    if 'temp_page' in locals() and temp_page and 'id' in temp_page:
                        try:
                            self.confluence.remove_page(temp_page['id'])
                        except Exception as e:
                            print(f"警告: 删除临时页面失败: {str(e)}")

            if image_url:
                self.uploaded_images[image_path] = image_url
                return image_url
            else:
                print(f"警告: 无法获取图片 {filename} 的URL")
                return None

        except Exception as e:
            print(f"警告: 上传图片 {image_path} 失败: {str(e)}")
            return None

    def _get_image_dimensions(self, image_path):
        """获取图片尺寸并计算适当的显示尺寸"""
        try:
            with Image.open(image_path) as img:
                width, height = img.size
                
                # 计算缩放比例
                width_ratio = self.max_image_width / width
                height_ratio = self.max_image_height / height
                # 使用较小的比例，确保图片完全适应限制
                ratio = min(width_ratio, height_ratio)
                
                # 如果图片已经很小，确保至少缩小到原始尺寸的60%
                if ratio > 1:
                    ratio = self.min_scale_ratio
                
                # 计算新的尺寸
                new_width = int(width * ratio)
                new_height = int(height * ratio)
                
                # 确保尺寸为整数
                return max(1, new_width), max(1, new_height)
        except Exception as e:
            print(f"警告: 无法读取图片 {image_path} 的尺寸: {str(e)}")
            return None, None

    def _escape_html(self, text):
        """转义HTML特殊字符"""
        return html.escape(text, quote=True)

    def _process_images(self, content, markdown_dir, space_key):
        """处理Markdown中的图片，上传并替换URL"""
        def process_image_path(image_path):
            """处理图片路径，支持相对路径和 attachments 目录"""
            # 如果是网络图片，直接返回
            if image_path.startswith(('http://', 'https://')):
                return image_path
                
            # 处理 Obsidian 的 attachments 路径
            if 'attachments/' in image_path:
                image_path = image_path.replace('attachments/', '')
                image_path = os.path.join(markdown_dir, 'attachments', image_path)
            # 处理相对路径
            elif not os.path.isabs(image_path):
                # 先尝试直接相对路径
                direct_path = os.path.join(markdown_dir, image_path)
                if os.path.exists(direct_path):
                    image_path = direct_path
                else:
                    # 尝试在 attachments 目录下查找
                    attachments_path = os.path.join(markdown_dir, 'attachments', image_path)
                    if os.path.exists(attachments_path):
                        image_path = attachments_path
                    
            return image_path

        def replace_obsidian_image(match):
            """处理 Obsidian 格式的图片 ![[image]]"""
            image_path = match.group(1)
            if image_path.startswith('Pasted image '):
                # 这是 Obsidian 的粘贴图片
                full_path = process_image_path(image_path)
            else:
                # 其他图片引用
                full_path = process_image_path(image_path)
                
            # 获取图片尺寸
            width, height = self._get_image_dimensions(full_path)
            
            # 上传图片
            image_url = self._upload_image(full_path, space_key)
            if image_url:
                if width and height:
                    return f'<ac:image ac:width="{width}" ac:height="{height}"><ri:url ri:value="{image_url}"/></ac:image>'
                else:
                    return f'<ac:image><ri:url ri:value="{image_url}"/></ac:image>'
            return match.group(0)  # 如果上传失败，保持原样

        def replace_markdown_image(match):
            """处理标准 Markdown 格式的图片 ![alt](path)"""
            alt_text = match.group(1)
            image_path = match.group(2)
            
            # 如果是网络图片，直接使用
            if image_path.startswith(('http://', 'https://')):
                return f'<ac:image><ri:url ri:value="{image_path}"/></ac:image>'
            
            # 处理本地图片路径
            full_path = process_image_path(image_path)
            
            # 获取图片尺寸
            width, height = self._get_image_dimensions(full_path)
            
            # 上传图片
            image_url = self._upload_image(full_path, space_key)
            if image_url:
                if width and height:
                    return f'<ac:image ac:width="{width}" ac:height="{height}"><ri:url ri:value="{image_url}"/></ac:image>'
                else:
                    return f'<ac:image><ri:url ri:value="{image_url}"/></ac:image>'
            return match.group(0)  # 如果上传失败，保持原样

        # 首先处理 Obsidian 格式的图片
        content = re.sub(r'!\[\[(.*?)\]\]', replace_obsidian_image, content)
        
        # 然后处理标准 Markdown 格式的图片
        content = re.sub(r'!\[(.*?)\]\((.*?)\)', replace_markdown_image, content)
        
        return content

    def _process_task_lists(self, content):
        """处理任务列表格式"""
        def replace_task(match):
            checked = match.group(1) == 'x'
            text = match.group(2)
            if checked:
                return f'<ac:task-list><ac:task><ac:task-status>complete</ac:task-status><ac:task-body>{text}</ac:task-body></ac:task></ac:task-list>'
            else:
                return f'<ac:task-list><ac:task><ac:task-status>incomplete</ac:task-status><ac:task-body>{text}</ac:task-body></ac:task></ac:task-list>'

        # 匹配任务列表语法
        pattern = r'- \[([ x])\] (.*?)(?=\n|$)'
        return re.sub(pattern, replace_task, content)

    def _get_page_version(self, page_id):
        """获取页面的当前版本信息"""
        try:
            page = self.confluence.get_page_by_id(page_id)
            return page.get('version', {}).get('number', 0)
        except:
            return 0

    def _find_page_in_parent(self, title, parent_page_id):
        """在指定父页面下查找页面"""
        try:
            # 获取父页面下的所有子页面
            children = self.confluence.get_child_pages(parent_page_id)
            # 在子页面中查找匹配标题的页面
            for child in children:
                if child['title'] == title:
                    return child
            return None
        except Exception as e:
            print(f"警告: 查找页面失败: {str(e)}")
            return None

    def publish(self, markdown_file, title=None, parent_page_id=None):
        """发布Markdown内容到Confluence"""
        try:
            # 获取Markdown文件所在目录（用于解析相对图片路径）
            markdown_dir = os.path.dirname(os.path.abspath(markdown_file))
            
            # 读取Markdown内容
            with open(markdown_file, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # 如果没有指定标题，使用文件名
            if not title:
                title = os.path.splitext(os.path.basename(markdown_file))[0]

            # 使用配置文件中的父页面ID（如果命令行没有指定）
            if not parent_page_id and 'parent_page_id' in self.config['confluence']:
                parent_page_id = self.config['confluence']['parent_page_id']
            
            if not parent_page_id:
                raise ValueError("必须指定父页面ID")

            # 在父页面下查找现有页面
            existing_page = self._find_page_in_parent(title, parent_page_id)
            
            if existing_page:
                self.current_page_id = existing_page['id']
            
            # 处理图片
            content = self._process_images(
                content,
                markdown_dir,
                self.config['confluence']['space']
            )

            # 处理任务列表
            content = self._process_task_lists(content)
            
            # 预处理 Markdown 中的 URL
            def encode_url_in_markdown(match):
                text = match.group(1)
                url = match.group(2)
                # 将 URL 中的 & 替换为 &amp;
                encoded_url = url.replace('&', '&amp;')
                return f'[{text}]({encoded_url})'
            
            content = re.sub(r'\[(.*?)\]\((.*?)\)', encode_url_in_markdown, content)
            
            # 转换Markdown为HTML
            html_content = markdown2.markdown(
                content,
                extras=[
                    'fenced-code-blocks',
                    'tables',
                    'header-ids',
                    'code-friendly'
                ]
            )
            
            # 处理代码块，使用 Confluence 的代码宏
            html_content = re.sub(
                r'<pre><code>(.*?)</code></pre>',
                lambda m: f'<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[{m.group(1)}]]></ac:plain-text-body></ac:structured-macro>',
                html_content,
                flags=re.DOTALL
            )
            
            # 处理链接中的特殊字符
            def encode_url_in_html(match):
                tag = match.group(1)
                attrs = match.group(2)
                # 将 href 属性中的 & 替换为 &amp;
                attrs = re.sub(r'href="([^"]*)"', lambda m: f'href="{m.group(1).replace("&", "&amp;")}"', attrs)
                return f'<{tag}{attrs}>'
            
            html_content = re.sub(r'<(\w+)([^>]*)>', encode_url_in_html, html_content)
            
            # 在内容开头添加目录宏
            toc_macro = (
                '<ac:structured-macro ac:name="toc">\n'
                '<ac:parameter ac:name="printable">true</ac:parameter>\n'
                '<ac:parameter ac:name="style">disc</ac:parameter>\n'
                '<ac:parameter ac:name="maxLevel">5</ac:parameter>\n'
                '<ac:parameter ac:name="minLevel">1</ac:parameter>\n'
                '<ac:parameter ac:name="class">rm-contents</ac:parameter>\n'
                '<ac:parameter ac:name="exclude">^目录$</ac:parameter>\n'
                '<ac:parameter ac:name="type">list</ac:parameter>\n'
                '<ac:parameter ac:name="outline">false</ac:parameter>\n'
                '<ac:parameter ac:name="include">.*</ac:parameter>\n'
                '</ac:structured-macro>\n\n'
            )
            
            html_content = toc_macro + html_content
            
            if existing_page:
                print(f"📝 正在覆盖更新页面: {title}...")
                try:
                    # 获取最新的页面版本信息
                    current_page = self.confluence.get_page_by_id(
                        page_id=existing_page['id'],
                        expand='version'
                    )
                    print(f"ℹ️ 当前页面版本: {current_page['version']['number']}")
                    
                    # 准备更新内容
                    body = {
                        'id': existing_page['id'],
                        'type': 'page',
                        'title': title,
                        'space': {'key': self.config['confluence']['space']},
                        'body': {
                            'storage': {
                                'value': html_content,
                                'representation': 'storage'
                            }
                        },
                        'version': {
                            'number': current_page['version']['number'] + 1
                        }
                    }
                    
                    # 尝试更新页面
                    result = self.confluence.put(
                        f'/rest/api/content/{existing_page["id"]}',
                        data=body
                    )
                    
                    if result:
                        print(f"✅ 已成功更新页面: {title}")
                        print(f"🔗 页面链接: {self.config['confluence']['url']}/pages/viewpage.action?pageId={existing_page['id']}")
                    else:
                        raise Exception("更新页面失败，API 返回为空")
                        
                except Exception as e:
                    print(f"⚠️ 更新页面时出错: {str(e)}")
                    raise e
            else:
                # 创建新页面
                print(f"📝 正在父页面 {parent_page_id} 下创建新页面: {title}...")
                new_page = self.confluence.create_page(
                    space=self.config['confluence']['space'],
                    title=title,
                    body=html_content,
                    parent_id=parent_page_id,
                    type='page',
                    representation='storage'
                )
                self.current_page_id = new_page['id']
                print(f"✅ 已成功创建页面: {title}")
                
        except Exception as e:
            print(f"❌ 发布失败: {str(e)}")
            sys.exit(1)

def main():
    """命令行入口"""
    parser = argparse.ArgumentParser(
        description='将 Markdown 文件发布到 Confluence',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
配置方式:
  1. 使用配置文件:
     md2kms test.md --config config.yml

  2. 使用环境变量:
     export KMS_URL=https://your-domain.atlassian.net
     export KMS_USERNAME=your.email@domain.com
     export KMS_PASSWORD=your-api-token
     export KMS_SPACE=SPACEKEY
     md2kms test.md

示例:
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
    
    args = parser.parse_args()
    
    try:
        converter = MarkdownToConfluence(args.config)
        converter.publish(args.markdown_file, args.title, args.parent)
    except KeyboardInterrupt:
        print("\n⚠️ 操作已取消")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 错误: {str(e)}")
        sys.exit(1)

if __name__ == '__main__':
    main()