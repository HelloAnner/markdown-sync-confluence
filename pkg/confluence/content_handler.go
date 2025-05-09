package confluence

import (
	"fmt"
	"regexp"
	"strings"
)

// ContentHandler handles the conversion of Confluence HTML content to Markdown
type ContentHandler struct {
	// 存储已处理的图片映射
	processedImages map[string]string
}

// NewContentHandler creates a new ContentHandler instance
func NewContentHandler() *ContentHandler {
	return &ContentHandler{
		processedImages: make(map[string]string),
	}
}

// ConvertToMarkdown converts Confluence HTML content to Markdown format
func (h *ContentHandler) ConvertToMarkdown(content string) (string, error) {
	// 预处理内容
	content = h.preProcessContent(content)

	// 转换各种元素
	content = h.convertHeadings(content)
	content = h.convertParagraphs(content)
	content = h.convertLists(content)
	content = h.convertTables(content)
	content = h.convertLinks(content)
	content = h.convertImages(content)
	content = h.convertCodeBlocks(content)
	content = h.convertMacros(content)
	content = h.convertTextFormatting(content)
	content = h.convertTaskLists(content)
	content = h.convertQuotes(content)
	content = h.convertAttachments(content)
	content = h.convertEmojis(content)
	content = h.convertMentions(content)
	content = h.convertStatus(content)

	// 后处理内容
	content = h.postProcessContent(content)

	return content, nil
}

// preProcessContent 预处理 HTML 内容
func (h *ContentHandler) preProcessContent(content string) string {
	// 移除 HTML 注释
	content = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(content, "")
	// 移除多余的空行
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	// 替换 <br/> 为换行符
	content = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(content, "\n")
	// 替换 &nbsp; 为空格
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	return content
}

// convertHeadings 转换标题
func (h *ContentHandler) convertHeadings(content string) string {
	// 转换 h1-h6 标签
	for i := 6; i >= 1; i-- {
		re := regexp.MustCompile(`(?s)<h` + string(rune('0'+i)) + `[^>]*>(.*?)</h` + string(rune('0'+i)) + `>`)
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			inner := re.FindStringSubmatch(match)[1]
			inner = h.cleanHTML(inner)
			return strings.Repeat("#", i) + " " + inner + "\n"
		})
	}
	return content
}

// convertParagraphs 转换段落
func (h *ContentHandler) convertParagraphs(content string) string {
	re := regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		inner := re.FindStringSubmatch(match)[1]
		inner = h.cleanHTML(inner)
		if strings.TrimSpace(inner) == "" {
			return "\n"
		}
		return inner + "\n\n"
	})
	return content
}

// convertLists 转换列表
func (h *ContentHandler) convertLists(content string) string {
	// 转换无序列表
	content = h.convertUnorderedLists(content)
	// 转换有序列表
	content = h.convertOrderedLists(content)
	return content
}

// convertUnorderedLists 转换无序列表
func (h *ContentHandler) convertUnorderedLists(content string) string {
	reList := regexp.MustCompile(`(?s)<ul[^>]*>(.*?)</ul>`)
	reItem := regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)

	return reList.ReplaceAllStringFunc(content, func(match string) string {
		listContent := reList.FindStringSubmatch(match)[1]
		return reItem.ReplaceAllStringFunc(listContent, func(item string) string {
			inner := reItem.FindStringSubmatch(item)[1]
			inner = h.cleanHTML(inner)
			return "- " + inner + "\n"
		}) + "\n"
	})
}

// convertOrderedLists 转换有序列表
func (h *ContentHandler) convertOrderedLists(content string) string {
	reList := regexp.MustCompile(`(?s)<ol[^>]*>(.*?)</ol>`)
	reItem := regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)
	
	return reList.ReplaceAllStringFunc(content, func(match string) string {
		listContent := reList.FindStringSubmatch(match)[1]
		items := reItem.FindAllStringSubmatch(listContent, -1)
		var result strings.Builder
		for i, item := range items {
			inner := h.cleanHTML(item[1])
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, inner))
		}
		result.WriteString("\n")
		return result.String()
	})
}

// convertTables 转换表格
func (h *ContentHandler) convertTables(content string) string {
	reTable := regexp.MustCompile(`(?s)<table[^>]*>(.*?)</table>`)
	reRow := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	reHeader := regexp.MustCompile(`(?s)<th[^>]*>(.*?)</th>`)
	reCell := regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)

	return reTable.ReplaceAllStringFunc(content, func(table string) string {
		rows := reRow.FindAllString(table, -1)
		if len(rows) == 0 {
			return table
		}

		var result strings.Builder
		
		// 处理表头
		headerRow := rows[0]
		headers := reHeader.FindAllStringSubmatch(headerRow, -1)
		if len(headers) == 0 {
			headers = reCell.FindAllStringSubmatch(headerRow, -1)
		}
		
		// 写入表头
		for i, header := range headers {
			if i > 0 {
				result.WriteString(" | ")
			}
			result.WriteString(h.cleanHTML(header[1]))
		}
		result.WriteString("\n")

		// 写入分隔行
		for i := range headers {
			if i > 0 {
				result.WriteString(" | ")
			}
			result.WriteString("---")
		}
		result.WriteString("\n")

		// 处理数据行
		for i := 1; i < len(rows); i++ {
			cells := reCell.FindAllStringSubmatch(rows[i], -1)
			for j, cell := range cells {
				if j > 0 {
					result.WriteString(" | ")
				}
				result.WriteString(h.cleanHTML(cell[1]))
			}
			result.WriteString("\n")
		}
		result.WriteString("\n")

		return result.String()
	})
}

// convertLinks 转换链接
func (h *ContentHandler) convertLinks(content string) string {
	re := regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		href := submatches[1]
		text := h.cleanHTML(submatches[2])
		return fmt.Sprintf("[%s](%s)", text, href)
	})
}

// convertImages 转换图片
func (h *ContentHandler) convertImages(content string) string {
	// 转换标准图片标签
	reImg := regexp.MustCompile(`<img[^>]*?src="([^"]+)"[^>]*?(?:alt="([^"]*)")?[^>]*?>`)
	content = reImg.ReplaceAllStringFunc(content, func(match string) string {
		submatches := reImg.FindStringSubmatch(match)
		src := submatches[1]
		alt := submatches[2]
		if alt == "" {
			alt = "image"
		}
		return fmt.Sprintf("![%s](%s)", alt, src)
	})

	// 转换 Confluence 图片宏
	reMacro := regexp.MustCompile(`<ac:image[^>]*?>(?:<ri:attachment[^>]*?ri:filename="([^"]+)"[^>]*?>)?(?:<ac:parameter[^>]*?ac:name="alt"[^>]*?>(.*?)</ac:parameter>)?</ac:image>`)
	return reMacro.ReplaceAllStringFunc(content, func(match string) string {
		submatches := reMacro.FindStringSubmatch(match)
		filename := submatches[1]
		alt := submatches[2]
		if alt == "" {
			alt = filename
		}
		if filename == "" {
			return match
		}
		return fmt.Sprintf("![%s](%s)", alt, filename)
	})
}

// convertCodeBlocks 转换代码块
func (h *ContentHandler) convertCodeBlocks(content string) string {
	// 转换 Confluence 代码宏
	reMacro := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="code"[^>]*?>(?s)(?:<ac:parameter[^>]*?ac:name="language"[^>]*?>(?:(.*?))</ac:parameter>)?(?:<ac:parameter[^>]*?ac:name="title"[^>]*?>(?:(.*?))</ac:parameter>)?.*?<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>.*?</ac:structured-macro>`)
	content = reMacro.ReplaceAllStringFunc(content, func(match string) string {
		submatches := reMacro.FindStringSubmatch(match)
		language := strings.TrimSpace(submatches[1])
		title := strings.TrimSpace(submatches[2])
		code := strings.TrimSpace(submatches[3])
		
		var result strings.Builder
		if title != "" {
			result.WriteString(fmt.Sprintf("**%s**\n", title))
		}
		result.WriteString("```")
		if language != "" {
			result.WriteString(language)
		}
		result.WriteString("\n")
		result.WriteString(code)
		result.WriteString("\n```\n")
		return result.String()
	})

	// 转换 <pre> 标签的代码块
	rePre := regexp.MustCompile(`<pre[^>]*?>(?s)(.*?)</pre>`)
	content = rePre.ReplaceAllStringFunc(content, func(match string) string {
		code := rePre.FindStringSubmatch(match)[1]
		code = h.cleanHTML(code)
		return "```\n" + code + "\n```\n"
	})

	return content
}

// convertMacros 转换 Confluence 宏
func (h *ContentHandler) convertMacros(content string) string {
	// 转换 Mermaid 图表宏
	content = h.convertMermaidMacros(content)
	// 转换展开宏
	content = h.convertExpandMacros(content)
	// 转换信息面板宏
	content = h.convertInfoPanelMacros(content)
	// 转换目录宏
	content = h.convertTOCMacros(content)
	// 转换其他宏
	content = h.convertOtherMacros(content)
	return content
}

// convertMermaidMacros 转换 Mermaid 图表宏
func (h *ContentHandler) convertMermaidMacros(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="markdown"[^>]*?>.*?<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>.*?</ac:structured-macro>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		mermaid := strings.TrimSpace(re.FindStringSubmatch(match)[1])
		return mermaid + "\n"
	})
}

// convertExpandMacros 转换展开宏
func (h *ContentHandler) convertExpandMacros(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="expand"[^>]*?>.*?<ac:parameter[^>]*?ac:name="title"[^>]*?>(.*?)</ac:parameter>.*?<ac:rich-text-body>(.*?)</ac:rich-text-body>.*?</ac:structured-macro>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		title := strings.TrimSpace(submatches[1])
		body := strings.TrimSpace(submatches[2])
		return fmt.Sprintf("---%s---\n%s\n---%s---\n", title, body, title)
	})
}

// convertInfoPanelMacros 转换信息面板宏
func (h *ContentHandler) convertInfoPanelMacros(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="(info|note|warning|tip)"[^>]*?>(?s).*?<ac:rich-text-body>(.*?)</ac:rich-text-body>.*?</ac:structured-macro>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		panelType := submatches[1]
		body := strings.TrimSpace(submatches[2])
		
		var prefix string
		switch panelType {
		case "info":
			prefix = "ℹ️ **Info:** "
		case "note":
			prefix = "📝 **Note:** "
		case "warning":
			prefix = "⚠️ **Warning:** "
		case "tip":
			prefix = "💡 **Tip:** "
		}
		
		return prefix + body + "\n\n"
	})
}

// convertTOCMacros 转换目录宏
func (h *ContentHandler) convertTOCMacros(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="toc"[^>]*?>.*?</ac:structured-macro>`)
	return re.ReplaceAllString(content, "[TOC]\n\n")
}

// convertOtherMacros 转换其他宏
func (h *ContentHandler) convertOtherMacros(content string) string {
	// 移除 TOC 宏
	content = regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="toc"[^>]*?>.*?</ac:structured-macro>`).ReplaceAllString(content, "")
	// 移除其他不需要的宏
	content = regexp.MustCompile(`<ac:structured-macro.*?</ac:structured-macro>`).ReplaceAllString(content, "")
	return content
}

// convertTextFormatting 转换文本格式化
func (h *ContentHandler) convertTextFormatting(content string) string {
	// 转换加粗
	content = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(content, "**$1**")
	// 转换斜体
	content = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(content, "*$1*")
	// 转换删除线
	content = regexp.MustCompile(`<del>(.*?)</del>`).ReplaceAllString(content, "~~$1~~")
	// 转换代码
	content = regexp.MustCompile(`<code>(.*?)</code>`).ReplaceAllString(content, "`$1`")
	return content
}

// postProcessContent 后处理内容
func (h *ContentHandler) postProcessContent(content string) string {
	// 清理多余的空行
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	// 清理行尾空格
	content = regexp.MustCompile(`[ \t]+\n`).ReplaceAllString(content, "\n")
	// 确保文件以单个换行符结束
	content = strings.TrimSpace(content) + "\n"
	return content
}

// cleanHTML 增强 HTML 清理
func (h *ContentHandler) cleanHTML(content string) string {
	// 移除 HTML 标签
	content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, "")
	
	// 解码所有 HTML 实体
	content = strings.NewReplacer(
		"&nbsp;", " ",
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", "\"",
		"&#39;", "'",
		"&ldquo;", "\"",
		"&rdquo;", "\"",
		"&lsquo;", "'",
		"&rsquo;", "'",
		"&hellip;", "...",
		"&mdash;", "—",
		"&ndash;", "–",
		"&trade;", "™",
		"&copy;", "©",
		"&reg;", "®",
	).Replace(content)
	
	return content
}

// convertTaskLists 转换任务列表
func (h *ContentHandler) convertTaskLists(content string) string {
	// 转换任务列表宏
	re := regexp.MustCompile(`<ac:task-list>(?s)(.*?)</ac:task-list>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		taskListContent := re.FindStringSubmatch(match)[1]
		
		// 转换单个任务项
		reTask := regexp.MustCompile(`<ac:task>(?s)(.*?)</ac:task>`)
		return reTask.ReplaceAllStringFunc(taskListContent, func(task string) string {
			// 提取任务状态和内容
			reStatus := regexp.MustCompile(`<ac:task-status>(.*?)</ac:task-status>`)
			reBody := regexp.MustCompile(`<ac:task-body>(.*?)</ac:task-body>`)
			
			status := reStatus.FindStringSubmatch(task)[1]
			body := reBody.FindStringSubmatch(task)[1]
			
			// 清理任务内容中的HTML
			body = h.cleanHTML(body)
			
			// 根据状态设置复选框
			checkbox := "[ ]"
			if status == "complete" {
				checkbox = "[x]"
			}
			
			return fmt.Sprintf("- %s %s\n", checkbox, body)
		})
	})
	return content
}

// convertQuotes 转换引用块
func (h *ContentHandler) convertQuotes(content string) string {
	// 转换 blockquote 标签
	re := regexp.MustCompile(`<blockquote[^>]*>(?s)(.*?)</blockquote>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		inner := re.FindStringSubmatch(match)[1]
		inner = h.cleanHTML(inner)
		// 为每一行添加引用标记
		lines := strings.Split(inner, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				lines[i] = "> " + line
			}
		}
		return strings.Join(lines, "\n") + "\n\n"
	})
	return content
}

// convertAttachments 转换附件
func (h *ContentHandler) convertAttachments(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="attachments"[^>]*?>(?s)(.*?)</ac:structured-macro>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取附件列表
		reFile := regexp.MustCompile(`<ri:attachment[^>]*?ri:filename="([^"]+)"[^>]*?>`)
		files := reFile.FindAllStringSubmatch(match, -1)
		
		var result strings.Builder
		result.WriteString("\n**Attachments:**\n")
		for _, file := range files {
			filename := file[1]
			result.WriteString(fmt.Sprintf("- [%s](%s)\n", filename, filename))
		}
		result.WriteString("\n")
		return result.String()
	})
}

// convertEmojis 转换表情符号
func (h *ContentHandler) convertEmojis(content string) string {
	// 转换 Confluence 表情宏
	re := regexp.MustCompile(`<ac:emoticon[^>]*?ac:name="([^"]+)"[^>]*?/>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		emoji := re.FindStringSubmatch(match)[1]
		// 映射常见的表情符号
		emojiMap := map[string]string{
			"smile":        ":smile:",
			"sad":         ":sad:",
			"wink":        ":wink:",
			"laugh":       ":laughing:",
			"thumbs-up":   ":+1:",
			"thumbs-down": ":-1:",
			"information": ":information_source:",
			"tick":        ":white_check_mark:",
			"cross":       ":x:",
			"warning":     ":warning:",
		}
		if replacement, ok := emojiMap[emoji]; ok {
			return replacement
		}
		return ":" + emoji + ":"
	})
}

// convertMentions 转换@提及
func (h *ContentHandler) convertMentions(content string) string {
	// 转换用户提及
	re := regexp.MustCompile(`<ac:link[^>]*?><ri:user[^>]*?ri:username="([^"]+)"[^>]*?/></ac:link>`)
	content = re.ReplaceAllString(content, "@$1")
	
	// 转换组提及
	reGroup := regexp.MustCompile(`<ac:link[^>]*?><ri:group[^>]*?ri:name="([^"]+)"[^>]*?/></ac:link>`)
	content = reGroup.ReplaceAllString(content, "@$1")
	
	return content
}

// convertStatus 转换状态宏
func (h *ContentHandler) convertStatus(content string) string {
	re := regexp.MustCompile(`<ac:structured-macro[^>]*?ac:name="status"[^>]*?>.*?<ac:parameter[^>]*?ac:name="title"[^>]*?>(.*?)</ac:parameter>.*?<ac:parameter[^>]*?ac:name="colour"[^>]*?>(.*?)</ac:parameter>.*?</ac:structured-macro>`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		status := submatches[1]
		color := submatches[2]
		
		// 根据颜色添加不同的标记
		switch color {
		case "Green":
			return fmt.Sprintf("✅ %s", status)
		case "Red":
			return fmt.Sprintf("❌ %s", status)
		case "Yellow":
			return fmt.Sprintf("⚠️ %s", status)
		case "Blue":
			return fmt.Sprintf("ℹ️ %s", status)
		default:
			return fmt.Sprintf("【%s】", status)
		}
	})
}
