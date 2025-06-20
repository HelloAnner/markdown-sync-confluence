package ai

import (
	"context"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func Chat(question string) (string, error) {
	return ChatWithPrompt(question, `
		角色: 格式专家
		任务: 将输入的文字内容重新规整、修正错别字和格式化内容,禁止扩展含义.
		输出: 润色后的文本,直接输出内容,禁止任何废话
		注意: 不允许修改markdown的语法,如标题、代码块内容、换行、空格等视作文本的一部分,仅修改文字内容
		`)
}

// ChatWithPrompt 使用自定义提示词进行AI对话
func ChatWithPrompt(content, systemPrompt string) (string, error) {
	ctx := context.Background()
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("DEEPSEEK_BASE_URL")),
		openai.WithToken(os.Getenv("DEEPSEEK_API_KEY")),
		openai.WithModel(os.Getenv("DEEPSEEK_MODEL_NAME")),
	)

	if err != nil {
		return "", err
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, content),
	}

	completion, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		return "", err
	}

	return completion.Choices[0].Content, nil
}