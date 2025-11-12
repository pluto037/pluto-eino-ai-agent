package llm

import (
	"context"
	"errors"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient 实现了LLM客户端接口
type OpenAIClient struct {
	client    *openai.Client
	modelName string
	maxTokens int
}

// NewOpenAIClient 创建一个新的OpenAI客户端
func NewOpenAIClient(apiKey string, modelName string, maxTokens int) *OpenAIClient {
	client := openai.NewClient(apiKey)
	return &OpenAIClient{
		client:    client,
		modelName: modelName,
		maxTokens: maxTokens,
	}
}

// Generate 生成文本
func (c *OpenAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("prompt cannot be empty")
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: c.maxTokens,
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateStream 生成流式响应
func (c *OpenAIClient) GenerateStream(ctx context.Context, prompt string, responseChan chan<- string) error {
	defer close(responseChan)

	if prompt == "" {
		return errors.New("prompt cannot be empty")
	}

	// 创建流式请求
	stream, err := c.client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: c.maxTokens,
			Stream:    true,
		},
	)

	if err != nil {
		return err
	}
	defer stream.Close()

	// 读取流式响应
	for {
		response, err := stream.Recv()
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			// 流结束
			return nil
		}

		// 发送内容片段
		if len(response.Choices) > 0 {
			content := response.Choices[0].Delta.Content
			if content != "" {
				responseChan <- content
			}
		}
	}
}
