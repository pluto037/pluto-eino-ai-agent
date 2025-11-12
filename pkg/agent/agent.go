package agent

import (
	"agentEino/pkg/logger"
	"agentEino/pkg/memory"
	"agentEino/pkg/tools"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LLMClient 定义了LLM客户端的接口
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateStream(ctx context.Context, prompt string, responseChan chan<- string) error
}

// Agent 定义了AI Agent的基本接口
type Agent interface {
	// Initialize 初始化Agent
	Initialize(ctx context.Context, llmClient LLMClient, toolManager *tools.ToolManager) error

	// Process 处理用户输入并返回响应
	Process(ctx context.Context, input string) (string, error)

	// ProcessStream 处理用户输入并返回流式响应
	ProcessStream(ctx context.Context, input string, responseChan chan<- string) error

	// ExecuteTool 执行指定的工具
	ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error)

	// Learn 从反馈中学习
	Learn(ctx context.Context, feedback string) error

	// GetConversationID 获取当前Agent的会话ID
	GetConversationID() string
	// SetConversationID 切换当前Agent会话ID（如果记忆存在则同步历史）
	SetConversationID(id string) error
}

// Config 包含Agent的配置信息
type Config struct {
	Name         string
	Description  string
	ModelConfig  ModelConfig
	MemoryConfig MemoryConfig
	ToolsConfig  ToolsConfig
}

// ModelConfig 包含LLM模型的配置
type ModelConfig struct {
	Provider  string // "openai" 或 "ollama"
	ModelName string
	APIKey    string // 对于OpenAI需要，Ollama可选
	BaseURL   string // Ollama服务器URL，例如 "http://localhost:11434"
	MaxTokens int
	Prompt    string // Agent的系统提示词
}

// MemoryConfig 包含记忆系统的配置
type MemoryConfig struct {
	MemoryType string
	DBPath     string
}

// ToolsConfig 包含工具的配置
type ToolsConfig struct {
	EnabledTools []string
}

// EinoAgent 实现了Agent接口
type EinoAgent struct {
	config                Config
	llmClient             LLMClient
	memory                Memory
	tools                 *tools.ToolManager
	currentConversationID string    // 当前对话ID
	messageHistory        []Message // 消息历史
}

// Message 表示对话中的一条消息
type Message struct {
	Role    string `json:"role"` // "user" 或 "assistant"
	Content string `json:"content"`
}

// 注意：LLMClient 接口已在文件顶部定义

// Memory 是记忆系统的接口
type Memory interface {
	Store(ctx context.Context, key string, value interface{}) error
	Retrieve(ctx context.Context, key string) (interface{}, error)
	Search(ctx context.Context, query string, limit int) ([]interface{}, error)

	// 对话管理方法
	CreateConversation(ctx context.Context, title string) (string, error)
	AddMessageToConversation(ctx context.Context, conversationID string, role string, content string) error
	GetConversation(ctx context.Context, conversationID string) (interface{}, error)
	ListConversations(ctx context.Context, limit int) ([]interface{}, error)
}

// MemoryAdapter 适配器，将memory包中的实现适配到Memory接口
type MemoryAdapter struct {
	simpleMem *memory.SimpleMemory
	vectorMem *memory.VectorMemory
}

// Store 存储数据
func (m *MemoryAdapter) Store(ctx context.Context, key string, value interface{}) error {
	if m.vectorMem != nil {
		return m.vectorMem.Store(ctx, key, value)
	}
	if m.simpleMem != nil {
		return m.simpleMem.Store(ctx, key, value)
	}
	return fmt.Errorf("未初始化内存系统")
}

// Retrieve 检索数据
func (m *MemoryAdapter) Retrieve(ctx context.Context, key string) (interface{}, error) {
	if m.vectorMem != nil {
		return m.vectorMem.GetVector(ctx, key)
	}
	return nil, fmt.Errorf("不支持的操作")
}

// Search 搜索数据
func (m *MemoryAdapter) Search(ctx context.Context, query string, limit int) ([]interface{}, error) {
	if m.vectorMem != nil {
		return m.vectorMem.Search(ctx, query, limit)
	}
	return nil, fmt.Errorf("不支持的操作")
}

// CreateConversation 创建对话
func (m *MemoryAdapter) CreateConversation(ctx context.Context, title string) (string, error) {
	if m.simpleMem != nil {
		conv, err := m.simpleMem.CreateConversation(ctx, title)
		if err != nil {
			return "", err
		}
		return conv.ID, nil
	}
	if m.vectorMem != nil {
		conv, err := m.vectorMem.CreateConversation(ctx, title)
		if err != nil {
			return "", err
		}
		return conv.ID, nil
	}
	return "", fmt.Errorf("未初始化内存系统")
}

// AddMessageToConversation 添加消息到对话
func (m *MemoryAdapter) AddMessageToConversation(ctx context.Context, conversationID string, role string, content string) error {
	msg := memory.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	if m.simpleMem != nil {
		return m.simpleMem.AddMessage(ctx, conversationID, msg)
	}
	if m.vectorMem != nil {
		return m.vectorMem.AddMessage(ctx, conversationID, msg)
	}
	return fmt.Errorf("未初始化内存系统")
}

// GetConversation 获取对话
func (m *MemoryAdapter) GetConversation(ctx context.Context, conversationID string) (interface{}, error) {
	if m.simpleMem != nil {
		return m.simpleMem.GetConversation(ctx, conversationID)
	}
	if m.vectorMem != nil {
		return m.vectorMem.GetConversation(ctx, conversationID)
	}
	return nil, fmt.Errorf("未初始化内存系统")
}

// ListConversations 列出对话
func (m *MemoryAdapter) ListConversations(ctx context.Context, limit int) ([]interface{}, error) {
	if m.simpleMem != nil {
		convs, err := m.simpleMem.GetConversationHistory(ctx, limit)
		if err != nil {
			return nil, err
		}
		var items []interface{}
		for _, c := range convs {
			items = append(items, c)
		}
		return items, nil
	}
	if m.vectorMem != nil {
		convs, err := m.vectorMem.GetConversationHistory(ctx, limit)
		if err != nil {
			return nil, err
		}
		var items []interface{}
		for _, c := range convs {
			items = append(items, c)
		}
		return items, nil
	}
	return nil, fmt.Errorf("未初始化内存系统")
}

// NewEinoAgent 创建一个新的EinoAgent实例
func NewEinoAgent(config Config) *EinoAgent {
	return &EinoAgent{
		config:         config,
		messageHistory: make([]Message, 0),
	}
}

// Initialize 初始EinoAgent
func (a *EinoAgent) Initialize(ctx context.Context, llmClient LLMClient, toolManager *tools.ToolManager) error {
	// 设置LLM客户端
	a.llmClient = llmClient

	// 设置工具管理器
	a.tools = toolManager

	logger.Info("初始化Agent", map[string]interface{}{
		"name": a.config.Name,
		"provider": a.config.ModelConfig.Provider,
		"model": a.config.ModelConfig.ModelName,
	})

	// 初始化内存系统
	memory, err := initializeMemory(ctx, a.config.MemoryConfig)
	if err != nil {
		logger.Error("初始化内存系统失败", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("初始化内存系统失败: %w", err)
	}
	a.memory = memory

	// 创建新对话
	conversationID, err := a.memory.CreateConversation(ctx, "新对话")
	if err != nil {
		logger.Error("创建对话失败", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("创建对话失败: %w", err)
	}
	a.currentConversationID = conversationID
	logger.Debug("创建新对话", map[string]interface{}{"conversation_id": conversationID})

	return nil
}

// GetConversationID 获取当前会话ID
func (a *EinoAgent) GetConversationID() string {
	return a.currentConversationID
}

// SetConversationID 切换当前会话ID，并尝试同步历史
func (a *EinoAgent) SetConversationID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("会话ID不能为空")
	}
	a.currentConversationID = id
	// 尝试从记忆加载历史到 messageHistory
	if a.memory != nil {
		if convIface, err := a.memory.GetConversation(context.Background(), id); err == nil {
			if conv, ok := convIface.(*memory.Conversation); ok && conv != nil {
				a.messageHistory = make([]Message, 0, len(conv.Messages))
				for _, m := range conv.Messages {
					a.messageHistory = append(a.messageHistory, Message{Role: m.Role, Content: m.Content})
				}
			}
		}
	}
	return nil
}

// initializeMemory 根据配置初始化内存系统
func initializeMemory(ctx context.Context, config MemoryConfig) (Memory, error) {
	// 使用内存模块

	// 根据配置创建不同类型的内存系统
	switch config.MemoryType {
	case "vector":
		// 创建向量内存
		vectorMem := memory.NewVectorMemoryWithDataDir(config.DBPath, config.DBPath+"/vectors/vectors.json")

		// 创建内存适配器
		memAdapter := &MemoryAdapter{
			vectorMem: vectorMem,
		}

		return memAdapter, nil
	case "simple":
		fallthrough
	default:
		// 默认使用简单内存
		simpleMem := memory.NewSimpleMemoryWithDataDir(config.DBPath)

		// 创建内存适配器
		memAdapter := &MemoryAdapter{
			simpleMem: simpleMem,
		}

		// 暂时不加载历史对话，需要实现LoadConversations方法
		// TODO: 实现加载历史对话功能

		return memAdapter, nil
	}
}

// extractToolCall 从响应中提取工具调用
func (a *EinoAgent) extractToolCall(response string) (string, string) {
	// 方法1: 检查 JSON 格式的 Function Calling
	// 格式: {"tool":"tool_name","params":{...}}
	if strings.Contains(response, `"tool"`) && strings.Contains(response, `"params"`) {
		var toolCall struct {
			Tool   string                 `json:"tool"`
			Params map[string]interface{} `json:"params"`
		}
		if err := json.Unmarshal([]byte(response), &toolCall); err == nil {
			if toolCall.Tool != "" {
				paramsJSON, _ := json.Marshal(toolCall.Params)
				return toolCall.Tool, string(paramsJSON)
			}
		}
	}

	// 方法2: 检查 Markdown 代码块格式
	// 格式: ```tool:tool_name\n{params}\n```
	if strings.Contains(response, "```tool:") {
		start := strings.Index(response, "```tool:")
		if start != -1 {
			end := strings.Index(response[start+8:], "```")
			if end != -1 {
				block := response[start+8 : start+8+end]
				lines := strings.SplitN(strings.TrimSpace(block), "\n", 2)
				if len(lines) >= 1 {
					toolName := strings.TrimSpace(lines[0])
					params := ""
					if len(lines) > 1 {
						params = strings.TrimSpace(lines[1])
					}
					return toolName, params
				}
			}
		}
	}

	// 方法3: 简单实现：检查是否包含工具调用标记（兼容旧格式）
	if strings.Contains(response, "使用工具:") {
		parts := strings.Split(response, "使用工具:")
		if len(parts) > 1 {
			toolParts := strings.SplitN(strings.TrimSpace(parts[1]), " ", 2)
			if len(toolParts) > 1 {
				return toolParts[0], strings.TrimSpace(toolParts[1])
			}
			return toolParts[0], ""
		}
	}
	return "", ""
}

// ExecuteTool 执行工具调用
func (a *EinoAgent) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	if a.tools == nil {
		return nil, fmt.Errorf("工具管理器未初始化")
	}
	return a.tools.ExecuteTool(ctx, toolName, params)
}

// Process 处理用户输入
func (a *EinoAgent) Process(ctx context.Context, input string) (string, error) {
	// 如果上层上下文提供了会话ID，则尝试绑定
	if cid, ok := ctx.Value("conversation_id").(string); ok && strings.TrimSpace(cid) != "" {
		_ = a.SetConversationID(cid)
	}
	// 如果是第一次对话，创建对话ID
	if a.currentConversationID == "" {
		a.currentConversationID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
		fmt.Printf("创建新对话ID: %s\n", a.currentConversationID)
	}

	// 将用户输入添加到消息历史
	a.messageHistory = append(a.messageHistory, Message{
		Role:    "user",
		Content: input,
	})

	// 将用户消息添加到当前对话
	if a.memory != nil && a.currentConversationID != "" {
		if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "user", input); err != nil {
			fmt.Printf("警告: 保存用户消息到对话失败: %v\n", err)
		}
	}

	// 构建完整提示词，使用更清晰的对话格式
	fullPrompt := a.buildPrompt()

	// 第一轮生成：用于解析是否需要工具
	preResp, err := a.llmClient.Generate(ctx, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("生成响应失败: %w", err)
	}

	// 提取工具调用（若存在）
	toolName, toolParamsText := a.extractToolCall(preResp)
	if toolName != "" {
		logger.Info("检测到工具调用", map[string]interface{}{
			"tool": toolName,
			"conversation_id": a.currentConversationID,
		})
		// 解析参数
		params := parseParams(toolParamsText)
		// 执行工具
		toolResult, err := a.ExecuteTool(ctx, toolName, params)
		if err != nil {
			logger.Error("工具执行失败", map[string]interface{}{
				"tool": toolName,
				"error": err.Error(),
			})
			toolResult = fmt.Sprintf("工具 %s 执行失败: %v", toolName, err)
		} else {
			logger.Debug("工具执行成功", map[string]interface{}{"tool": toolName})
		}
		// 将工具结果注入为系统消息，参与下一轮生成
		a.messageHistory = append(a.messageHistory, Message{Role: "system", Content: fmt.Sprintf("工具(%s)输出: %v", toolName, toolResult)})
		// 重新构建提示并进行最终生成
		fullPrompt = a.buildPrompt()
		finalResp, err := a.llmClient.Generate(ctx, fullPrompt)
		if err != nil {
			return "", fmt.Errorf("二次生成失败: %w", err)
		}
		if finalResp == "" {
			finalResp = "抱歉，我无法生成有效的响应。请重试。"
		}
		// 将助手响应添加到消息历史
		a.messageHistory = append(a.messageHistory, Message{Role: "assistant", Content: finalResp})
		// 将助手响应添加到当前对话
		if a.memory != nil && a.currentConversationID != "" {
			if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "assistant", finalResp); err != nil {
				fmt.Printf("警告: 保存助手响应到对话失败: %v\n", err)
			}
		}
		return finalResp, nil
	}

	// 无工具调用时，直接采用预响应
	response := preResp
	if response == "" {
		response = "抱歉，我无法生成有效的响应。请重新尝试您的问题。"
		fmt.Println("警告: LLM返回空响应，使用默认消息")
	}

	// 将助手响应添加到消息历史
	a.messageHistory = append(a.messageHistory, Message{
		Role:    "assistant",
		Content: response,
	})

	// 将助手响应添加到当前对话
	if a.memory != nil && a.currentConversationID != "" {
		if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "assistant", response); err != nil {
			fmt.Printf("警告: 保存助手响应到对话失败: %v\n", err)
		}
	}

	return response, nil
}

// ProcessStream 处理用户输入并返回流式响应
func (a *EinoAgent) ProcessStream(ctx context.Context, input string, responseChan chan<- string) error {
	// 如果上层上下文提供了会话ID，则尝试绑定
	if cid, ok := ctx.Value("conversation_id").(string); ok && strings.TrimSpace(cid) != "" {
		_ = a.SetConversationID(cid)
	}
	// 如果是第一次对话，创建对话ID
	if a.currentConversationID == "" {
		a.currentConversationID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
		fmt.Printf("创建新对话ID: %s\n", a.currentConversationID)
	}

	// 将用户输入添加到消息历史
	a.messageHistory = append(a.messageHistory, Message{
		Role:    "user",
		Content: input,
	})

	// 将用户消息添加到当前对话
	if a.memory != nil && a.currentConversationID != "" {
		if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "user", input); err != nil {
			fmt.Printf("警告: 保存用户消息到对话失败: %v\n", err)
		}
	}

	// 构建完整提示词
	fullPrompt := a.buildPrompt()

	// 创建内部通道来收集完整响应
	internalChan := make(chan string, 100)
	var fullResponse strings.Builder

	// 启动goroutine来处理最终流式响应
	go func() {
		defer close(responseChan)

		for chunk := range internalChan {
			fullResponse.WriteString(chunk)
			responseChan <- chunk
		}

		// 流式响应完成后，保存完整响应到历史和对话
		response := fullResponse.String()
		if response != "" {
			// 将助手响应添加到消息历史
			a.messageHistory = append(a.messageHistory, Message{
				Role:    "assistant",
				Content: response,
			})

			// 将助手响应添加到当前对话
			if a.memory != nil && a.currentConversationID != "" {
				if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "assistant", response); err != nil {
					fmt.Printf("警告: 保存助手响应到对话失败: %v\n", err)
				}
			}
		}
	}()

	// 发送思考事件
	a.sendThinkingEvent(responseChan, "analyzing", "正在分析您的问题...")

	// 第一轮非流式生成，仅用于解析工具调用
	preResp, err := a.llmClient.Generate(ctx, fullPrompt)
	if err != nil {
		return fmt.Errorf("生成响应失败: %w", err)
	}
	
	toolName, toolParamsText := a.extractToolCall(preResp)
	if toolName != "" {
		// 发送工具调用事件
		a.sendThinkingEvent(responseChan, "tool_call", fmt.Sprintf("准备调用工具: %s", toolName))
		
		params := parseParams(toolParamsText)
		toolResult, err := a.ExecuteTool(ctx, toolName, params)
		if err != nil {
			toolResult = fmt.Sprintf("工具 %s 执行失败: %v", toolName, err)
			a.sendThinkingEvent(responseChan, "tool_error", fmt.Sprintf("工具执行失败: %v", err))
		} else {
			// 发送工具结果事件
			a.sendThinkingEvent(responseChan, "tool_result", fmt.Sprintf("工具返回结果，正在生成最终回复..."))
		}
		
		// 注入工具输出
		a.messageHistory = append(a.messageHistory, Message{Role: "system", Content: fmt.Sprintf("工具(%s)输出: %v", toolName, toolResult)})
		// 重新构建提示后进行流式最终生成
		finalPrompt := a.buildPrompt()
		a.sendThinkingEvent(responseChan, "generating", "正在生成回复...")
		return a.llmClient.GenerateStream(ctx, finalPrompt, internalChan)
	}
	
	// 无工具调用时直接流式生成
	a.sendThinkingEvent(responseChan, "generating", "正在生成回复...")
	return a.llmClient.GenerateStream(ctx, fullPrompt, internalChan)
}

// buildPrompt 构建完整的提示词
func (a *EinoAgent) buildPrompt() string {
	var fullPrompt string

	// 添加系统消息
	if a.config.ModelConfig.Prompt != "" {
		fullPrompt += "system: " + a.config.ModelConfig.Prompt + "\n\n"
	}

	// 添加历史消息上下文（最多保留最近10条消息）
	maxHistoryMessages := 10
	startIdx := 0
	if len(a.messageHistory) > maxHistoryMessages {
		startIdx = len(a.messageHistory) - maxHistoryMessages
	}

	// 添加对话历史
	for i := startIdx; i < len(a.messageHistory); i++ {
		msg := a.messageHistory[i]
		fullPrompt += fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content)
	}

	// 添加明确的助手提示
	fullPrompt += "assistant: "

	return fullPrompt
}

// Learn 从反馈中学习
func (a *EinoAgent) Learn(ctx context.Context, feedback string) error {
	// 如果内存系统未初始化，则跳过
	if a.memory == nil {
		return nil
	}

	// 创建反馈消息
	feedbackMsg := fmt.Sprintf("反馈 (%s): %s", time.Now().Format("2006-01-02 15:04:05"), feedback)

	// 将反馈添加到当前对话
	if a.currentConversationID != "" {
		if err := a.memory.AddMessageToConversation(ctx, a.currentConversationID, "system", feedbackMsg); err != nil {
			return fmt.Errorf("添加反馈到对话失败: %w", err)
		}
	}

	// 存储反馈到向量存储（如果支持）
	if err := a.memory.Store(ctx, fmt.Sprintf("feedback_%d", time.Now().Unix()), feedbackMsg); err != nil {
		return fmt.Errorf("存储反馈失败: %w", err)
	}

	return nil
}

// parseParams 支持JSON或k=v,k2=v2格式的简单解析
func parseParams(text string) map[string]interface{} {
	params := make(map[string]interface{})
	t := strings.TrimSpace(text)
	if t == "" {
		return params
	}
	// 优先尝试JSON
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(t), &m); err == nil {
		return m
	}
	// 回退到 key=value, key2=value2
	parts := strings.Split(t, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

// sendThinkingEvent 发送思维链事件（仅在流式模式下）
func (a *EinoAgent) sendThinkingEvent(responseChan chan<- string, eventType, message string) {
	// 发送特殊格式的事件标记
	eventData := fmt.Sprintf("[THINKING:%s:%s]", eventType, message)
	responseChan <- eventData
}
