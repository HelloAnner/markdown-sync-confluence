import os
import re
import mimetypes
from PIL import Image

class ImageHandler:
    """图片处理器，负责处理和上传图片"""

    def __init__(self, confluence_client, config):
        self.confluence = confluence_client
        self.config = config
        self.uploaded_images = {}  # 缓存已上传的图片
        self.max_image_width = 600  # 最大图片宽度
        self.max_image_height = 400  # 最大图片高度
        self.min_scale_ratio = 0.6  # 最小缩放比例

    def _get_image_content_type(self, image_path):
        """获取图片的MIME类型"""
        content_type = mimetypes.guess_type(image_path)[0]
        if not content_type:
            content_type = 'image/png'
        return content_type

    def _get_image_dimensions(self, image_path):
        """获取图片尺寸并计算适当的显示尺寸"""
        try:
            with Image.open(image_path) as img:
                width, height = img.size
                
                width_ratio = self.max_image_width / width
                height_ratio = self.max_image_height / height
                ratio = min(width_ratio, height_ratio)
                
                if ratio > 1:
                    ratio = self.min_scale_ratio
                
                new_width = int(width * ratio)
                new_height = int(height * ratio)
                
                return max(1, new_width), max(1, new_height)
        except Exception as e:
            print(f"警告: 无法读取图片 {image_path} 的尺寸: {str(e)}")
            return None, None

    def _upload_image(self, image_path, page_id):
        """上传图片到Confluence并返回图片URL"""
        if image_path in self.uploaded_images:
            return self.uploaded_images[image_path]

        try:
            if os.path.isabs(image_path):
                abs_image_path = image_path
            else:
                abs_image_path = os.path.abspath(image_path)

            if not os.path.exists(abs_image_path):
                print(f"警告: 找不到图片 {abs_image_path}")
                return None

            filename = os.path.basename(abs_image_path)

            try:
                with open(abs_image_path, 'rb') as f:
                    image_data = f.read()
                
                attachment = self.confluence.attach_content(
                    content=image_data,
                    name=filename,
                    content_type=self._get_image_content_type(abs_image_path),
                    page_id=page_id
                )
                
                if attachment and '_links' in attachment:
                    image_url = attachment['_links'].get('download', '')
                    if image_url:
                        if not image_url.startswith(('http://', 'https://')):
                            image_url = f"{self.config['confluence']['url']}{image_url}"
                        self.uploaded_images[image_path] = image_url
                        return image_url
                
                print(f"警告: 无法获取图片 {filename} 的链接信息")
                return None

            except Exception as e:
                print(f"警告: 上传图片失败: {str(e)}")
                return None

        except Exception as e:
            print(f"警告: 处理图片 {image_path} 失败: {str(e)}")
            return None

    def process_image_path(self, image_path):
        """处理图片路径，支持相对路径和尺寸信息"""
        # 处理尺寸信息
        size = None
        if '|' in image_path:
            image_path, size_str = image_path.split('|', 1)
            try:
                size = int(size_str)
            except ValueError:
                pass

        # 处理图片路径
        if image_path.startswith(('http://', 'https://')):
            return image_path, size

        # 标准化路径分隔符
        image_path = image_path.replace('\\', '/')
        
        # 如果是绝对路径，直接返回
        if os.path.isabs(image_path):
            return image_path, size

        # 尝试多个可能的路径
        possible_paths = [
            # 1. 直接相对于 markdown 文件目录
            os.path.join(self.markdown_dir, image_path),
            
            # 2. 检查 attachments 子目录
            os.path.join(self.markdown_dir, 'attachments', image_path),
            
            # 3. 如果路径包含 attachments，则从 markdown 目录重新构建
            os.path.join(self.markdown_dir, *image_path.split('/')) if '/' in image_path else None,
            
            # 4. 处理 '../' 相对路径
            os.path.normpath(os.path.join(self.markdown_dir, image_path))
        ]

        # 尝试所有可能的路径
        for path in possible_paths:
            if path and os.path.exists(path):
                return os.path.abspath(path), size
        
        # 如果都找不到，返回原始路径
        print(f"⚠️ 警告: 找不到图片文件: {image_path}")
        print(f"搜索的路径:")
        for path in possible_paths:
            if path:
                print(f"- {path}")
        return os.path.join(self.markdown_dir, image_path), size

    def process_images(self, content, markdown_dir, page_id):
        """处理Markdown中的图片，上传并替换URL"""
        self.markdown_dir = markdown_dir
        self.page_id = page_id

        def replace_image(match):
            # 处理两种不同的匹配模式
            if len(match.groups()) == 1:  # Obsidian 格式 ![[path]]
                image_path = match.group(1)
                alt_text = ""
            else:  # Markdown 格式 ![alt](path)
                alt_text = match.group(1)
                image_path = match.group(2)
            
            full_path, size = self.process_image_path(image_path)
            
            if full_path.startswith(('http://', 'https://')):
                if size:
                    return f'<ac:image ac:width="{size}"><ri:url ri:value="{full_path}"/></ac:image>'
                return f'<ac:image><ri:url ri:value="{full_path}"/></ac:image>'
            
            try:
                if os.path.exists(full_path):
                    image_url = self._upload_image(full_path, page_id)
                    if image_url:
                        if size:
                            return f'<ac:image ac:width="{size}"><ri:url ri:value="{image_url}"/></ac:image>'
                        return f'<ac:image><ri:url ri:value="{image_url}"/></ac:image>'
            except Exception as e:
                print(f"⚠️ 警告: 处理图片时出错 {full_path}: {str(e)}")
            
            return ''  # 如果处理失败，返回空字符串

        # 处理标准Markdown图片语法 ![alt](path)
        content = re.sub(r'!\[(.*?)\]\((.*?)\)', replace_image, content)
        
        # 处理Obsidian图片语法 ![[path]]
        content = re.sub(r'!\[\[(.*?)\]\]', replace_image, content)
        
        return content 