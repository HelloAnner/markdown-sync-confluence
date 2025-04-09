import markdown2
import html
import re

class ContentHandler:
    """内容处理器，负责将 Markdown 转换为 Confluence 格式"""

    def __init__(self):
        self.markdown_extras = [
            'fenced-code-blocks',
            'tables',
            'header-ids',
            'code-friendly'
        ]

    def _process_code_blocks(self, html_content):
        """处理代码块，使用 Confluence 的代码宏"""
        return re.sub(
            r'<pre><code>(.*?)</code></pre>',
            lambda m: f'<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[{m.group(1)}]]></ac:plain-text-body></ac:structured-macro>',
            html_content,
            flags=re.DOTALL
        )

    def _process_html_links(self, html_content):
        """处理HTML中的链接"""
        def encode_url_in_html(match):
            tag = match.group(1)
            attrs = match.group(2)
            attrs = re.sub(r'href="([^"]*)"', lambda m: f'href="{m.group(1).replace("&", "&amp;")}"', attrs)
            return f'<{tag}{attrs}>'
        
        return re.sub(r'<(\w+)([^>]*)>', encode_url_in_html, html_content)

    def _add_toc_macro(self, html_content):
        """添加目录宏"""
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
        return toc_macro + html_content

    def convert_to_confluence(self, content):
        """将 Markdown 内容转换为 Confluence 格式"""
        # 转换 Markdown 为 HTML
        html_content = markdown2.markdown(
            content,
            extras=self.markdown_extras
        )

        # 处理代码块
        html_content = self._process_code_blocks(html_content)
        
        # 处理链接
        html_content = self._process_html_links(html_content)
        
        # 添加目录宏
        html_content = self._add_toc_macro(html_content)
        
        return html_content 