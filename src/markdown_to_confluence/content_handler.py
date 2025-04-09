import markdown2
import html
import re
from typing import List, Callable, Dict, Any


class MarkdownTransformer:
    """Markdown 转换器基类"""

    def transform(self, content: str) -> str:
        """转换内容"""
        return content


class MermaidTransformer(MarkdownTransformer):
    """Mermaid 图表转换器"""

    def transform(self, content: str) -> str:
        mermaid_blocks = []

        def save_mermaid(match):
            code_content = match.group(1)
            placeholder = f"MERMAID_PLACEHOLDER_{len(mermaid_blocks)}"
            mermaid_blocks.append(code_content)
            return f"```{placeholder}```"

        # 保存 mermaid 代码块并替换为占位符
        content = re.sub(
            r"```mermaid\n(.*?)```", save_mermaid, content, flags=re.DOTALL
        )

        self.mermaid_blocks = mermaid_blocks
        return content

    def post_process(self, html_content: str) -> str:
        """处理保存的 mermaid 块"""
        for i, mermaid_content in enumerate(self.mermaid_blocks):
            placeholder = f"MERMAID_PLACEHOLDER_{i}"
            mermaid_macro = (
                '<ac:structured-macro ac:name="markdown">'
                "<ac:plain-text-body><![CDATA["
                f"```mermaid\n{mermaid_content}\n```"
                "]]></ac:plain-text-body>"
                "</ac:structured-macro>"
            )
            html_content = html_content.replace(
                f"<p><code>{placeholder}</code></p>", mermaid_macro
            )
            html_content = html_content.replace(
                f"<code>{placeholder}</code>", mermaid_macro
            )
        return html_content


class HighlightTransformer(MarkdownTransformer):
    """高亮语法转换器"""

    # Obsidian 高亮颜色到 HTML 颜色的映射
    COLOR_MAP = {
        "#BBFABBA6": "#E3FCE3",  # 浅绿色
        "#FFB8EBA6": "#FFE3F1",  # 浅粉色
        "#FF8F8FA6": "#FFE3E3",  # 浅红色
        "#FBFB8FA6": "#FFFAE5",  # 浅黄色
        "#ABF7F7A6": "#E3FCFC",  # 浅蓝色
        None: "#FFF3B8",  # 默认颜色
    }

    def transform(self, content: str) -> str:
        def replace_highlight(match):
            text = match.group(2)  # 高亮文本
            style = match.group(1)  # 样式属性

            # 提取背景色
            color = None
            if style:
                color_match = re.search(r"background:\s*(#[A-Fa-f0-9]+)", style)
                if color_match:
                    color = color_match.group(1)

            # 获取对应的 HTML 颜色
            html_color = self.COLOR_MAP.get(color, self.COLOR_MAP[None])

            # 返回带背景色的 span 标签
            return f'<span style="background-color: {html_color};">{text}</span>'

        # 转换高亮语法
        content = re.sub(
            r'<mark(?:\s+style="([^"]*)")?>([^<]+)</mark>', replace_highlight, content
        )
        return content


class ContentHandler:
    """内容处理器，负责将 Markdown 转换为 Confluence 格式"""

    def __init__(self):
        self.markdown_extras = [
            "fenced-code-blocks",
            "tables",
            "header-ids",
            "code-friendly",
            "fenced-code-attributes",
        ]
        # 初始化转换器列表
        self.transformers: List[MarkdownTransformer] = [
            MermaidTransformer(),
            HighlightTransformer(),
        ]

    def _process_code_blocks(self, html_content: str) -> str:
        """处理普通代码块，使用 Confluence 的代码宏"""
        return re.sub(
            r'<pre><code(?:\s+class=".*?")?>([^<]+)</code></pre>',
            lambda m: f'<ac:structured-macro ac:name="code"><ac:plain-text-body><![CDATA[{m.group(1)}]]></ac:plain-text-body></ac:structured-macro>',
            html_content,
            flags=re.DOTALL,
        )

    def _process_html_links(self, html_content: str) -> str:
        """处理HTML中的链接"""
        def encode_url_in_html(match):
            tag = match.group(1)
            attrs = match.group(2)
            attrs = re.sub(r'href="([^"]*)"', lambda m: f'href="{m.group(1).replace("&", "&amp;")}"', attrs)
            return f'<{tag}{attrs}>'

        return re.sub(r'<(\w+)([^>]*)>', encode_url_in_html, html_content)

    def _add_toc_macro(self, html_content: str) -> str:
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

    def convert_to_confluence(self, content: str) -> str:
        """将 Markdown 内容转换为 Confluence 格式"""
        # 预处理：应用所有转换器的预处理
        for transformer in self.transformers:
            content = transformer.transform(content)

        # 转换 Markdown 为 HTML
        html_content = markdown2.markdown(
            content,
            extras=self.markdown_extras
        )

        # 处理普通代码块
        html_content = self._process_code_blocks(html_content)

        # 后处理：应用所有转换器的后处理
        for transformer in self.transformers:
            if hasattr(transformer, "post_process"):
                html_content = transformer.post_process(html_content)

        # 处理链接
        html_content = self._process_html_links(html_content)

        # 添加目录宏
        html_content = self._add_toc_macro(html_content)

        return html_content 
