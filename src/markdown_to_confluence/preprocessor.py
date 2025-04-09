import re

class MarkdownPreprocessor:
    """Markdown 预处理器，处理 front matter 和基础格式化"""
    
    @staticmethod
    def strip_front_matter(content):
        """移除 Markdown 文件开头的 YAML front matter"""
        pattern = r'^---\s*\n(.*?)\n---\s*\n'
        if content.startswith('---'):
            match = re.match(pattern, content, re.DOTALL)
            if match:
                return content[match.end():]
        return content

    @staticmethod
    def process_task_lists(content):
        """处理任务列表格式"""
        def replace_task(match):
            checked = match.group(1) == 'x'
            text = match.group(2)
            if checked:
                return f'<ac:task-list><ac:task><ac:task-status>complete</ac:task-status><ac:task-body>{text}</ac:task-body></ac:task></ac:task-list>'
            else:
                return f'<ac:task-list><ac:task><ac:task-status>incomplete</ac:task-status><ac:task-body>{text}</ac:task-body></ac:task></ac:task-list>'

        pattern = r'- \[([ x])\] (.*?)(?=\n|$)'
        return re.sub(pattern, replace_task, content)

    @staticmethod
    def preprocess_urls(content):
        """预处理 Markdown 中的 URL"""
        def encode_url_in_markdown(match):
            text = match.group(1)
            url = match.group(2)
            encoded_url = url.replace('&', '&amp;')
            return f'[{text}]({encoded_url})'
        
        return re.sub(r'\[(.*?)\]\((.*?)\)', encode_url_in_markdown, content)

    def process(self, content):
        """处理 Markdown 内容"""
        content = self.strip_front_matter(content)
        content = self.process_task_lists(content)
        content = self.preprocess_urls(content)
        return content 