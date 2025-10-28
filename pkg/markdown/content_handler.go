package markdown

import (
    "regexp"
    "strconv"
    "strings"

    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/extension"
    "github.com/yuin/goldmark/parser"
    "github.com/yuin/goldmark/renderer/html"
)

// ContentHandler 处理Markdown内容并将其转换为Confluence格式
type ContentHandler struct {
	markdown          goldmark.Markdown    // Markdown解析器
	imagePlaceholders map[string]string    // 图片占位符映射
}

// NewContentHandler 创建一个新的内容处理器
// 返回:
//   - *ContentHandler: 内容处理器实例
func NewContentHandler() *ContentHandler {
	// 配置Goldmark，启用所需扩展
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub风格Markdown支持
			extension.Footnote,      // 脚注支持
			extension.Table,         // 表格支持
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // 自动生成标题ID
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),    // 启用硬换行
			html.WithXHTML(),        // 使用XHTML格式
			html.WithUnsafe(),       // 允许原始HTML
		),
	)

	return &ContentHandler{
		markdown:          md,
		imagePlaceholders: make(map[string]string),
	}
}

// ConvertToConfluence 将Markdown内容转换为Confluence格式
// 参数:
//   - content: Markdown内容
// 返回:
//   - string: 转换后的Confluence格式内容
//   - error: 处理过程中的错误
func (ch *ContentHandler) ConvertToConfluence(content string) (string, error) {
	// 使用预处理器处理内容
	content = ch.preProcessMermaid(content)    // 预处理Mermaid图表
	content = ch.preProcessFolding(content)    // 预处理折叠块
	content = ch.preProcessTaskLists(content)  // 预处理任务列表

	// 将Markdown转换为HTML
	var htmlContent strings.Builder
	if err := ch.markdown.Convert([]byte(content), &htmlContent); err != nil {
		return "", err
	}

	// 对HTML进行后处理以适配Confluence
	result := htmlContent.String()
	result = ch.postProcessCodeBlocks(result)  // 处理代码块
	result = ch.postProcessLinks(result)       // 处理链接
	result = ch.postProcessMermaid(result)     // 处理Mermaid图表
	result = ch.postProcessFolding(result)     // 处理折叠块
    result = ch.postProcessTables(result)      // 处理表格
    result = ch.postProcessMarkHighlights(result) // 处理 <mark> 高亮
    result = ch.addTOCMacro(result)            // 添加目录宏

	return result, nil
}

// preProcessMermaid 处理Mermaid代码块
// 参数:
//   - content: Markdown内容
// 返回:
//   - string: 预处理后的内容，其中Mermaid块被替换为占位符
func (ch *ContentHandler) preProcessMermaid(content string) string {
	// 保留并替换mermaid块为占位符
	re := regexp.MustCompile("```mermaid\\s*\\n([\\s\\S]*?)```")
	return re.ReplaceAllStringFunc(content, func(match string) string {
		mermaidContent := re.FindStringSubmatch(match)[1]
		return "MERMAID_PLACEHOLDER:" + mermaidContent + ":MERMAID_PLACEHOLDER"
	})
}

// postProcessMermaid 将Mermaid占位符转换为Confluence宏
// 参数:
//   - content: 包含Mermaid占位符的内容
// 返回:
//   - string: 处理后的内容，其中占位符被替换为Confluence宏
func (ch *ContentHandler) postProcessMermaid(content string) string {
	// 查找所有mermaid占位符并将其转换为Confluence Markdown宏
	re := regexp.MustCompile("MERMAID_PLACEHOLDER:([\\s\\S]*?):MERMAID_PLACEHOLDER")
	return re.ReplaceAllStringFunc(content, func(match string) string {
		mermaidContent := re.FindStringSubmatch(match)[1]
		// 转义CDATA结束标记
		mermaidContent = escapeCDATA(mermaidContent)
		return `<ac:structured-macro ac:name="markdown">` +
			`<ac:plain-text-body><![CDATA[` +
			"```mermaid\n" + mermaidContent + "\n```" +
			`]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
}

// escapeCDATA 转义内容中的CDATA结束标记序列']]>'
// 参数:
//   - content: 需要转义的内容
// 返回:
//   - string: 转义后的内容
func escapeCDATA(content string) string {
	// 将]]>替换为]]&gt;以防止破坏CDATA部分
	return strings.ReplaceAll(content, "]]>", "]]&gt;")
}

// preProcessFolding 处理折叠/可折叠部分
// 参数:
//   - content: Markdown内容
// 返回:
//   - string: 预处理后的内容，其中折叠块被替换为占位符
func (ch *ContentHandler) preProcessFolding(content string) string {
	// 匹配自定义标题折叠块: ---标题--- 内容 ---标题---
	// Go不支持正则表达式中的反向引用，因此需要采用不同的方法
	lines := strings.Split(content, "\n")
	result := []string{}
	inFoldBlock := false
	var currentTitle string
	var foldContent []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// 检查折叠块开始
		startMatch := regexp.MustCompile(`^---([^-\n]+?)---\s*$`).FindStringSubmatch(line)
		if !inFoldBlock && len(startMatch) > 1 {
			// 找到折叠块开始
			currentTitle = strings.TrimSpace(startMatch[1])
			inFoldBlock = true
			foldContent = []string{}
			continue
		}
		
		// 检查具有相同标题的折叠块结束
		endMatch := regexp.MustCompile(`^---([^-\n]+?)---\s*$`).FindStringSubmatch(line)
		if inFoldBlock && len(endMatch) > 1 && strings.TrimSpace(endMatch[1]) == currentTitle {
			// 找到带有匹配标题的折叠块结束
			content := strings.Join(foldContent, "\n")
			result = append(result, "FOLD_PLACEHOLDER_TITLE:"+currentTitle+":CONTENT:"+content+":FOLD_PLACEHOLDER")
			inFoldBlock = false
			currentTitle = ""
			continue
		}
		
		// 折叠块内部
		if inFoldBlock {
			foldContent = append(foldContent, line)
		} else {
			result = append(result, line)
		}
	}
	
	// 处理不完整的折叠块
	if inFoldBlock {
		// 如果折叠块未正确关闭，则添加回原始行
		result = append(result, "---"+currentTitle+"---")
		result = append(result, foldContent...)
	}
	
	content = strings.Join(result, "\n")

	// 匹配旧样式折叠块: ---折叠--- 内容 ---折叠---
	// 使用相同的逐行方法
	lines = strings.Split(content, "\n")
	result = []string{}
	inFoldBlock = false
	foldContent = []string{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// 检查折叠块开始
		if !inFoldBlock && line == "---折叠---" {
			// 找到折叠块开始
			inFoldBlock = true
			foldContent = []string{}
			continue
		}
		
		// 检查折叠块结束
		if inFoldBlock && line == "---折叠---" {
			// 找到折叠块结束
			content := strings.Join(foldContent, "\n")
			result = append(result, "FOLD_PLACEHOLDER_TITLE:点击展开:CONTENT:"+content+":FOLD_PLACEHOLDER")
			inFoldBlock = false
			continue
		}
		
		// 折叠块内部
		if inFoldBlock {
			foldContent = append(foldContent, line)
		} else {
			result = append(result, line)
		}
	}
	
	// 处理不完整的折叠块
	if inFoldBlock {
		// 如果折叠块未正确关闭，则添加回原始行
		result = append(result, "---折叠---")
		result = append(result, foldContent...)
	}
	
	return strings.Join(result, "\n")
}

// postProcessFolding 将折叠占位符转换为Confluence展开宏
// 参数:
//   - content: 包含折叠占位符的内容
// 返回:
//   - string: 处理后的内容，其中占位符被替换为Confluence宏
func (ch *ContentHandler) postProcessFolding(content string) string {
	re := regexp.MustCompile("FOLD_PLACEHOLDER_TITLE:([^:]*?):CONTENT:([\\s\\S]*?):FOLD_PLACEHOLDER")
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		title := submatches[1]
		foldContent := submatches[2]
		
		// 将内容转换为HTML（嵌套内容需要）
		var nestedHTML strings.Builder
		if err := ch.markdown.Convert([]byte(foldContent), &nestedHTML); err != nil {
			return match // 错误时返回原始内容
		}
		
		// 转义嵌套内容中的CDATA结束标记
		nestedContent := escapeCDATA(nestedHTML.String())
		
		return `<ac:structured-macro ac:name="expand">` +
			`<ac:parameter ac:name="title">` + title + `</ac:parameter>` +
			`<ac:rich-text-body>` + nestedContent + `</ac:rich-text-body>` +
			`</ac:structured-macro>`
	})
}

// preProcessTaskLists 处理任务列表/检查列表
// 参数:
//   - content: Markdown内容
// 返回:
//   - string: 预处理后的内容，其中任务列表被替换为占位符
func (ch *ContentHandler) preProcessTaskLists(content string) string {
	// 匹配任务列表: - [ ] 项目 或 - [x] 项目
	re := regexp.MustCompile(`(?m)^- \[([ x])\] (.+)$`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		checked := submatches[1] == "x"
		text := submatches[2]
		
		status := "incomplete"
		if checked {
			status = "complete"
		}
		
		return "TASK_PLACEHOLDER_STATUS:" + status + ":TEXT:" + text + ":TASK_PLACEHOLDER"
	})
}

// postProcessCodeBlocks 将代码块转换为Confluence代码宏
// 参数:
//   - content: HTML内容
// 返回:
//   - string: 处理后的内容，其中代码块被替换为Confluence宏
func (ch *ContentHandler) postProcessCodeBlocks(content string) string {
	// 查找带有语言规范的代码块
	re := regexp.MustCompile(`<pre><code class="language-([^"]+)">([\s\S]*?)</code></pre>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		language := submatches[1]
		code := submatches[2]
		
		// 在代码中取消转义HTML实体
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		
		// 转义CDATA结束标记
		code = escapeCDATA(code)
		
		return `<ac:structured-macro ac:name="code">` +
			`<ac:parameter ac:name="language">` + language + `</ac:parameter>` +
			`<ac:plain-text-body><![CDATA[` + code + `]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
	
	// 查找没有语言规范的代码块
	reNoLang := regexp.MustCompile(`<pre><code>([\s\S]*?)</code></pre>`)
	content = reNoLang.ReplaceAllStringFunc(content, func(match string) string {
		submatches := reNoLang.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		code := submatches[1]
		
		// 在代码中取消转义HTML实体
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		
		// 转义CDATA结束标记
		code = escapeCDATA(code)
		
		return `<ac:structured-macro ac:name="code">` +
			`<ac:plain-text-body><![CDATA[` + code + `]]></ac:plain-text-body>` +
			`</ac:structured-macro>`
	})
	
	return content
}

// postProcessLinks 处理链接中的特殊属性
// 参数:
//   - content: HTML内容
// 返回:
//   - string: 处理后的内容，其中链接被适当编码
func (ch *ContentHandler) postProcessLinks(content string) string {
	// 处理链接以正确编码特殊字符
	re := regexp.MustCompile(`<a href="([^"]+)"`)
	
	return re.ReplaceAllStringFunc(content, func(match string) string {
		url := re.FindStringSubmatch(match)[1]
		
		// 确保正确编码&符号
		encodedURL := strings.ReplaceAll(url, "&amp;", "&")
		encodedURL = strings.ReplaceAll(encodedURL, "&", "&amp;")
		
		return `<a href="` + encodedURL + `"`
	})
}

// postProcessTables 修复Confluence的表格HTML
// 参数:
//   - content: HTML内容
// 返回:
//   - string: 处理后的内容，其中表格HTML被修复
func (ch *ContentHandler) postProcessTables(content string) string {
	// 确保所有表格都有tbody
	content = regexp.MustCompile(`<table>\s*<tr>`).ReplaceAllString(content, "<table><tbody><tr>")
	content = regexp.MustCompile(`</tr>\s*</table>`).ReplaceAllString(content, "</tr></tbody></table>")
	
	// 修复表格中的BR标签以正确关闭
	content = regexp.MustCompile(`<br(?:\s*/)?>`).ReplaceAllString(content, "<br/>")
	
	return content
}

// addTOCMacro 添加目录宏（如果需要）
// 参数:
//   - content: HTML内容
// 返回:
//   - string: 处理后的内容，如果需要会添加目录宏
func (ch *ContentHandler) addTOCMacro(content string) string {
		tocMacro := `<ac:structured-macro ac:name="toc">` +
				`<ac:parameter ac:name="printable">true</ac:parameter>` +
				`<ac:parameter ac:name="style">disc</ac:parameter>` +
				`<ac:parameter ac:name="maxLevel">3</ac:parameter>` +
				`<ac:parameter ac:name="minLevel">1</ac:parameter>` +
				`</ac:structured-macro>`
	
    return tocMacro + "\n" + content
}

// postProcessMarkHighlights 将 HTML <mark style="background: ...">text</mark>
// 转为 Confluence 友好的 <span style="background-color: ...; color: ...">text</span>
// 颜色解析失败时默认使用红色（#FFEBE6 背景），并根据调色板选择合适文字色
func (ch *ContentHandler) postProcessMarkHighlights(content string) string {
    // 匹配 <mark ...>...</mark>（大小写不敏感）
    reMark := regexp.MustCompile(`(?i)<mark([^>]*)>([\s\S]*?)</mark>`)
    return reMark.ReplaceAllStringFunc(content, func(match string) string {
        sub := reMark.FindStringSubmatch(match)
        if len(sub) < 3 {
            return match
        }
        attrs := sub[1]
        inner := sub[2]

        // 提取 style 属性（分别处理单双引号，RE2 无反向引用）
        styleVal := ""
        if m := regexp.MustCompile(`(?i)style\s*=\s*"([^"]*)"`).FindStringSubmatch(attrs); len(m) >= 2 {
            styleVal = m[1]
        } else if m := regexp.MustCompile(`(?i)style\s*=\s*'([^']*)'`).FindStringSubmatch(attrs); len(m) >= 2 {
            styleVal = m[1]
        }

        // 从 style 中提取 background/background-color（大小写不敏感）
        reBg := regexp.MustCompile(`(?i)background(?:-color)?\s*:\s*([^;]+)`) // 捕获到分号为止
        bgRaw := ""
        if styleVal != "" {
            if m := reBg.FindStringSubmatch(styleVal); len(m) >= 2 {
                bgRaw = strings.TrimSpace(m[1])
            }
        }

        // 解析颜色
        r, g, b, ok := parseColorToRGB(bgRaw)
        if !ok {
            // 默认红色高亮（subtle red）
            r, g, b = 255, 235, 230 // #FFEBE6
        }

        // 映射到 Confluence 浅色高亮与文字色
        bgHex, textHex := mapToConfluenceHighlight(r, g, b)

        // 用 span 替换 mark
        return `<span style="background-color: ` + bgHex + `; color: ` + textHex + `;">` + inner + `</span>`
    })
}

// parseColorToRGB 支持 #RGB、#RRGGBB、#RRGGBBAA、rgb()/rgba()
func parseColorToRGB(s string) (int, int, int, bool) {
    if s == "" {
        return 0, 0, 0, false
    }
    s = strings.TrimSpace(strings.ToLower(s))

    // 去掉可能的 !important
    if i := strings.Index(s, "!important"); i >= 0 {
        s = strings.TrimSpace(s[:i])
    }

    // 十六进制
    if strings.HasPrefix(s, "#") {
        hex := strings.TrimPrefix(s, "#")
        if len(hex) == 8 { // RRGGBBAA -> 忽略 alpha
            hex = hex[:6]
        } else if len(hex) == 4 { // RGBA -> 展开前3位并忽略 alpha
            hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
        } else if len(hex) == 3 { // RGB -> 展开
            hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
        } else if len(hex) != 6 {
            return 0, 0, 0, false
        }
        r := mustHexToByte(hex[0:2])
        g := mustHexToByte(hex[2:4])
        b := mustHexToByte(hex[4:6])
        return int(r), int(g), int(b), true
    }

    // rgb()/rgba()
    if strings.HasPrefix(s, "rgb(") || strings.HasPrefix(s, "rgba(") {
        l := strings.IndexByte(s, '(')
        rpar := strings.LastIndexByte(s, ')')
        if l < 0 || rpar <= l {
            return 0, 0, 0, false
        }
        parts := strings.Split(s[l+1:rpar], ",")
        if len(parts) < 3 {
            return 0, 0, 0, false
        }
        r := parseIntClamp(parts[0])
        g := parseIntClamp(parts[1])
        b := parseIntClamp(parts[2])
        return r, g, b, true
    }

    return 0, 0, 0, false
}

func parseIntClamp(s string) int {
    s = strings.TrimSpace(s)
    // 百分比形式，如 100%
    if strings.HasSuffix(s, "%") {
        sv := strings.TrimSpace(strings.TrimSuffix(s, "%"))
        if f, err := strconv.ParseFloat(sv, 64); err == nil {
            if f < 0 {
                f = 0
            }
            if f > 100 {
                f = 100
            }
            v := int(f*255.0/100.0 + 0.5)
            if v < 0 {
                return 0
            }
            if v > 255 {
                return 255
            }
            return v
        }
        return 0
    }
    if v, err := strconv.Atoi(s); err == nil {
        if v < 0 {
            return 0
        }
        if v > 255 {
            return 255
        }
        return v
    }
    return 0
}

func mustHexToByte(hs string) byte {
    if v, err := strconv.ParseUint(hs, 16, 8); err == nil {
        return byte(v)
    }
    return 0
}

// mapToConfluenceHighlight 将 RGB 映射到 Confluence 常见浅色高亮，返回 (背景HEX, 文字HEX)
func mapToConfluenceHighlight(r, g, b int) (string, string) {
    type colorPair struct {
        bg   [3]int
        txt  [3]int
        bgHx string
        txHx string
    }

    palette := []colorPair{
        // Yellow
        {bg: [3]int{255, 250, 230}, txt: [3]int{23, 43, 77}, bgHx: "#FFFAE6", txHx: "#172B4D"},
        // Blue
        {bg: [3]int{222, 235, 255}, txt: [3]int{7, 71, 166}, bgHx: "#DEEBFF", txHx: "#0747A6"},
        // Green
        {bg: [3]int{227, 252, 239}, txt: [3]int{0, 102, 68}, bgHx: "#E3FCEF", txHx: "#006644"},
        // Red（默认失败回退色）
        {bg: [3]int{255, 235, 230}, txt: [3]int{191, 38, 0}, bgHx: "#FFEBE6", txHx: "#BF2600"},
        // Purple
        {bg: [3]int{234, 230, 255}, txt: [3]int{82, 67, 170}, bgHx: "#EAE6FF", txHx: "#5243AA"},
        // Teal
        {bg: [3]int{230, 252, 255}, txt: [3]int{7, 71, 166}, bgHx: "#E6FCFF", txHx: "#0747A6"},
        // Gray
        {bg: [3]int{244, 245, 247}, txt: [3]int{66, 82, 110}, bgHx: "#F4F5F7", txHx: "#42526E"},
        // Orange（接近示例 #FFB86C 的暖色）
        {bg: [3]int{255, 216, 181}, txt: [3]int{143, 63, 14}, bgHx: "#FFD8B5", txHx: "#8F3F0E"},
    }

    bestIdx := 0
    bestDist := 1<<31 - 1
    for i, p := range palette {
        dr := r - p.bg[0]
        dg := g - p.bg[1]
        db := b - p.bg[2]
        dist := dr*dr + dg*dg + db*db
        if dist < bestDist {
            bestDist = dist
            bestIdx = i
        }
    }
    return palette[bestIdx].bgHx, palette[bestIdx].txHx
}
