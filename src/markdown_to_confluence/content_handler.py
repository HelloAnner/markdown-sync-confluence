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


class TableTransformer(MarkdownTransformer):
    """表格转换器，解决表格中的HTML标签闭合问题"""

    def post_process(self, html_content: str) -> str:
        """修复表格中的HTML问题"""
        if "<table>" in html_content:
            # 确保表格中的所有标签正确闭合
            # 替换自闭合标签，确保正确闭合
            html_content = re.sub(r'<br\s*/?>(?!</br>)', '<br></br>', html_content)
            
            # 检查并修复所有表格中的HTML
            html_content = re.sub(
                r'(<table>.*?</table>)',
                self._fix_table_html,
                html_content,
                flags=re.DOTALL
            )
        
        return html_content
    
    def _fix_table_html(self, match):
        """修复表格HTML"""
        table_html = match.group(1)
        
        # 确保tbody标签存在
        if "<tbody>" not in table_html:
            table_html = table_html.replace("<table>", "<table><tbody>")
            table_html = table_html.replace("</table>", "</tbody></table>")
        
        # 修复表格行
        rows = re.findall(r'<tr>(.*?)</tr>', table_html, re.DOTALL)
        fixed_table = "<table><tbody>"
        
        for row in rows:
            fixed_row = "<tr>"
            
            # 修复单元格，确保每个单元格都正确闭合
            for cell_type in ['th', 'td']:
                # 使用非贪婪匹配，确保每个标签都有对应的闭合标签
                cell_pattern = f'<{cell_type}(.*?)>(.*?)</{cell_type}>'
                cells = re.findall(cell_pattern, row, re.DOTALL)
                
                for attrs, content in cells:
                    # 确保内部的br标签正确闭合
                    content = re.sub(r'<br\s*/?>(?!</br>)', '<br></br>', content)
                    fixed_row += f'<{cell_type}{attrs}>{content}</{cell_type}>'
            
            # 如果没有找到任何单元格，检查是否存在自闭合的td或未正常解析的单元格
            if "<td" not in fixed_row and "<th" not in fixed_row:
                # 为每个可能的单元格添加正确的闭合标签
                row = re.sub(r'<(td|th)([^>]*)>', r'<\1\2>', row)
                row = re.sub(r'</(td|th)>', r'</\1>', row)
                fixed_row += row
            
            fixed_row += "</tr>"
            fixed_table += fixed_row
        
        fixed_table += "</tbody></table>"
        
        return fixed_table


class FoldingTransformer(MarkdownTransformer):
    """折叠块转换器，将 ---标题--- 和内容转换为 Confluence 的展开宏"""
    
    def transform(self, content: str) -> str:
        """识别并处理折叠块"""
        # 存储折叠块信息
        fold_blocks = []
        
        # 为了避免干扰普通的分隔线(---)，我们需要更精确的模式
        # 1. 首先匹配自定义标题的折叠块：---标题--- 内容 ---标题---
        # 确保标题两侧的---是单独一行的，且标题不为空
        fold_pattern = r'(?:^|\n)---([^-\n]+?)---\s*\n(.*?)\n\s*---\1---(?:\n|$)'
        
        def replace_fold(match):
            title = match.group(1).strip()
            fold_content = match.group(2).strip()
            placeholder = f"FOLD_PLACEHOLDER_{len(fold_blocks)}"
            fold_blocks.append((title, fold_content))
            return f"\n{placeholder}\n"
        
        # 替换折叠块为占位符
        processed_content = re.sub(fold_pattern, replace_fold, content, flags=re.DOTALL)
        
        # 2. 处理旧格式的折叠块（无标题指定）：---折叠--- 内容 ---折叠---
        # 同样确保两侧的标记是单独一行的
        old_fold_pattern = r'(?:^|\n)---折叠---\s*\n(.*?)\n\s*---折叠---(?:\n|$)'
        
        def replace_old_fold(match):
            fold_content = match.group(1).strip()
            placeholder = f"FOLD_PLACEHOLDER_{len(fold_blocks)}"
            fold_blocks.append(("点击展开", fold_content))
            return f"\n{placeholder}\n"
        
        # 替换旧格式折叠块为占位符
        processed_content = re.sub(old_fold_pattern, replace_old_fold, processed_content, flags=re.DOTALL)
        
        self.fold_blocks = fold_blocks
        return processed_content
    
    def post_process(self, html_content: str) -> str:
        """将折叠块占位符替换为 Confluence 展开宏"""
        for i, (title, fold_content) in enumerate(self.fold_blocks):
            placeholder = f"FOLD_PLACEHOLDER_{i}"
            
            # 转换折叠内容为HTML
            fold_html = markdown2.markdown(
                fold_content,
                extras=["fenced-code-blocks", "tables", "header-ids", "code-friendly"]
            )
            
            # 创建Confluence展开宏
            expand_macro = (
                '<ac:structured-macro ac:name="expand">'
                f'<ac:parameter ac:name="title">{html.escape(title)}</ac:parameter>'
                '<ac:rich-text-body>'
                f'{fold_html}'
                '</ac:rich-text-body>'
                '</ac:structured-macro>'
            )
            
            # 替换占位符
            html_content = html_content.replace(f"<p>{placeholder}</p>", expand_macro)
            html_content = html_content.replace(placeholder, expand_macro)
        
        return html_content


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
            FoldingTransformer(),  # 先处理折叠块
            MermaidTransformer(),
            HighlightTransformer(),
            TableTransformer(),
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
