package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaClient 实现了LLM客户端接口
type OllamaClient struct {
	baseURL   string
	modelName string
	maxTokens int
}

// OllamaRequest 表示发送到Ollama API的请求
type OllamaRequest struct {
	Model    string    `json:"model"`
	Prompt   string    `json:"prompt,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Stream   bool      `json:"stream,omitempty"`
	Options  Options   `json:"options,omitempty"`
}

// Message 表示对话中的一条消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options 表示Ollama请求的选项
type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	MaxTokens   int     `json:"num_predict,omitempty"`
}

// OllamaResponse 表示从Ollama API返回的响应
type OllamaResponse struct {
	Model      string `json:"model"`
	Response   string `json:"response"`
	CreatedAt  string `json:"created_at"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

// ChatStreamResponse 兼容 /api/chat 的返回结构（流式与非流式通用）
type ChatStreamResponse struct {
	Model      string      `json:"model"`
	Message    ChatMessage `json:"message"`
	CreatedAt  string      `json:"created_at"`
	Done       bool        `json:"done"`
	DoneReason string      `json:"done_reason"`
}

// ChatMessage 表示 chat 端点的消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewOllamaClient 创建一个新的Ollama客户端
func NewOllamaClient(baseURL, modelName string, maxTokens int) *OllamaClient {
	// 确保baseURL以"/"结尾
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &OllamaClient{
		baseURL:   baseURL,
		modelName: modelName,
		maxTokens: maxTokens,
	}
}

// parsePromptToMessages 将文本提示转换为消息数组
func parsePromptToMessages(prompt string) []Message {
	// 分割提示词为行
	lines := strings.Split(prompt, "\n")
	var currentRole, currentContent string
	var messages []Message

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			// 如果当前有内容，添加空行
			if currentContent != "" {
				currentContent += "\n"
			}
			continue
		}

		// 检查是否是新角色的开始
		if strings.HasPrefix(line, "system:") || strings.HasPrefix(line, "user:") || strings.HasPrefix(line, "assistant:") {
			// 保存之前的消息
			if currentRole != "" && currentContent != "" {
				messages = append(messages, Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent),
				})
			}

			// 提取新角色和内容开始
			parts := strings.SplitN(line, ":", 2)
			currentRole = parts[0]
			if len(parts) > 1 {
				currentContent = strings.TrimSpace(parts[1])
			} else {
				currentContent = ""
			}
		} else {
			// 继续添加到当前内容
			if currentContent != "" {
				currentContent += "\n" + line
			} else {
				currentContent = line
			}
		}
	}

	// 添加最后一条消息
	if currentRole != "" && currentContent != "" {
		messages = append(messages, Message{
			Role:    currentRole,
			Content: strings.TrimSpace(currentContent),
		})
	}

	return messages
}

// Generate 使用提示词生成响应，支持流式处理
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.generateWithRetry(ctx, prompt, 0)
}

// GenerateStream 生成流式响应，返回一个通道用于接收实时响应
func (c *OllamaClient) GenerateStream(ctx context.Context, prompt string, responseChan chan<- string) error {
	defer close(responseChan)
	return c.generateStreamWithRetry(ctx, prompt, responseChan, 0)
}

// generateStreamWithRetry 带重试的流式生成方法
func (c *OllamaClient) generateStreamWithRetry(ctx context.Context, prompt string, responseChan chan<- string, retryCount int) error {

	const maxLoadRetries = 3
	if retryCount > maxLoadRetries {
		return fmt.Errorf("模型加载重试次数超限，已尝试 %d 次", retryCount)
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// 构建请求
	req := OllamaRequest{
		Model:  c.modelName,
		Stream: true, // 启用流式响应
		Options: Options{
			Temperature: 0.7,
			MaxTokens:   c.maxTokens,
		},
	}

	// 检查是否是结构化消息格式，并标记是否走 chat 端点
	isChat := false
	if strings.Contains(prompt, "user:") && strings.Contains(prompt, "assistant:") {
		messages := parsePromptToMessages(prompt)
		if len(messages) > 0 {
			req.Messages = messages
			isChat = true
		} else {
			req.Prompt = prompt
		}
	} else {
		req.Prompt = prompt
	}

	// 发送请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求（依据 isChat 切换端点）
	endpoint := "api/generate"
	if isChat {
		endpoint = "api/chat"
	}
	httpReq, err := http.NewRequestWithContext(timeoutCtx, "POST", c.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 处理流式响应
	scanner := bufio.NewScanner(resp.Body)
	var fullResponse strings.Builder
	var isModelLoading bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 先尝试按 /api/generate 解析；失败则尝试 /api/chat
		var genResp OllamaResponse
		if err := json.Unmarshal([]byte(line), &genResp); err == nil && (genResp.Response != "" || genResp.Done || genResp.DoneReason != "") {
			if genResp.DoneReason == "load" {
				isModelLoading = true
				fmt.Printf("模型正在加载中，等待5秒后重试... (重试次数: %d/%d)\n", retryCount, maxLoadRetries)
				time.Sleep(5 * time.Second)
				return c.generateStreamWithRetry(ctx, prompt, responseChan, retryCount+1)
			}
			if genResp.Response != "" {
				responseChan <- genResp.Response
				fullResponse.WriteString(genResp.Response)
			}
			if genResp.Done {
				break
			}
			continue
		}

		var chatResp ChatStreamResponse
		if err := json.Unmarshal([]byte(line), &chatResp); err == nil {
			if chatResp.DoneReason == "load" {
				isModelLoading = true
				fmt.Printf("模型正在加载中，等待5秒后重试... (重试次数: %d/%d)\n", retryCount, maxLoadRetries)
				time.Sleep(5 * time.Second)
				return c.generateStreamWithRetry(ctx, prompt, responseChan, retryCount+1)
			}
			if chatResp.Message.Content != "" {
				responseChan <- chatResp.Message.Content
				fullResponse.WriteString(chatResp.Message.Content)
			}
			if chatResp.Done {
				break
			}
			continue
		}

		// 都解析失败，记录原始内容
		fmt.Printf("解析流式响应失败，原始内容: %s\n", line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %w", err)
	}

	if isModelLoading {
		return fmt.Errorf("模型仍在加载中")
	}

	if fullResponse.Len() == 0 {
		return fmt.Errorf("模型返回了空响应")
	}

	return nil
}

// generateWithRetry 带重试计数的生成方法，防止无限递归
func (c *OllamaClient) generateWithRetry(ctx context.Context, prompt string, retryCount int) (string, error) {
	// 防止无限递归，最多重试3次模型加载
	const maxLoadRetries = 3
	if retryCount > maxLoadRetries {
		return "", fmt.Errorf("模型加载重试次数超限，已尝试 %d 次", retryCount)
	}

	// 创建带超时的上下文
	fmt.Println("开始处理请求...")
	timeoutCtx, cancel := context.WithTimeout(ctx, 180*time.Second) // 增加超时时间到3分钟
	defer cancel()

	// 构建请求
	req := OllamaRequest{
		Model:  c.modelName,
		Stream: false, // 非流式响应
		Options: Options{
			Temperature: 0.7,
			MaxTokens:   c.maxTokens,
		},
	}

	// 检查是否是结构化消息格式，并标记是否走 chat 端点
	isChat := false
	if strings.Contains(prompt, "user:") && strings.Contains(prompt, "assistant:") {
		// 解析为消息数组
		messages := parsePromptToMessages(prompt)
		if len(messages) > 0 {
			req.Messages = messages
			isChat = true
		} else {
			req.Prompt = prompt
		}
	} else {
		req.Prompt = prompt
	}

	fmt.Printf("准备发送请求到Ollama...\n")
	// 发送请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 最大重试次数
	maxRetries := 3
	var resp *http.Response
	var lastErr error

	// 重试循环
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 创建HTTP请求（依据 isChat 切换端点）
		endpoint := "api/generate"
		if isChat {
			endpoint = "api/chat"
		}
		httpReq, err := http.NewRequestWithContext(timeoutCtx, "POST", c.baseURL+endpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			return "", fmt.Errorf("创建HTTP请求失败: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")

		fmt.Printf("尝试请求 #%d...\n", attempt)

		// 发送请求
		client := &http.Client{}
		resp, err = client.Do(httpReq)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				fmt.Printf("请求失败，等待 %d 秒后重试: %v\n", attempt*2, err)
				time.Sleep(time.Duration(attempt*2) * time.Second) // 指数退避
				continue
			}
			return "", fmt.Errorf("HTTP请求失败，已重试 %d 次: %w", maxRetries, lastErr)
		}
		break // 成功，退出重试循环
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("成功收到响应，正在处理...")

	// 解析响应
	responseStr := string(body)
	fmt.Printf("原始响应内容: %s\n", responseStr)

	// 检查是否包含错误信息
	if strings.Contains(responseStr, "error") {
		return "", fmt.Errorf("API返回错误: %s", responseStr)
	}

	// 优先尝试按 /api/generate 解析
	var genResp OllamaResponse
	if err := json.Unmarshal(body, &genResp); err == nil && (genResp.Response != "" || genResp.Done || genResp.DoneReason != "") {
		if genResp.DoneReason == "load" {
			fmt.Printf("模型正在加载中，等待5秒后重试... (重试次数: %d/%d)\n", retryCount, maxLoadRetries)
			time.Sleep(5 * time.Second)
			return c.generateWithRetry(ctx, prompt, retryCount+1)
		}
		if strings.TrimSpace(genResp.Response) != "" {
			fmt.Printf("成功生成响应，长度: %d 字符\n", len(genResp.Response))
			return genResp.Response, nil
		}
	}

	// 再尝试按 /api/chat 解析
	var chatResp ChatStreamResponse
	if err := json.Unmarshal(body, &chatResp); err == nil {
		if chatResp.DoneReason == "load" {
			fmt.Printf("模型正在加载中，等待5秒后重试... (重试次数: %d/%d)\n", retryCount, maxLoadRetries)
			time.Sleep(5 * time.Second)
			return c.generateWithRetry(ctx, prompt, retryCount+1)
		}
		if strings.TrimSpace(chatResp.Message.Content) != "" {
			fmt.Printf("成功生成响应（chat），长度: %d 字符\n", len(chatResp.Message.Content))
			return chatResp.Message.Content, nil
		}
	}

	// JSON解析都不符合或为空，尝试将响应作为纯文本处理
	if strings.TrimSpace(responseStr) != "" {
		fmt.Println("将响应作为纯文本处理")
		return strings.TrimSpace(responseStr), nil
	}

	// 最终失败
	fmt.Println("警告: 收到空响应")
	return "", fmt.Errorf("模型返回了空响应")
}
