package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"agentEino/pkg/agent"
	"agentEino/pkg/api"
	"agentEino/pkg/llm"
	"agentEino/pkg/logger"
	"agentEino/pkg/tools"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		logger.Warn(".env 文件未找到，使用默认配置")
	}

	// 设置日志级别
	logLevel := os.Getenv("LOG_LEVEL")
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		logger.SetLevel(logger.DEBUG)
	case "INFO":
		logger.SetLevel(logger.INFO)
	case "WARN":
		logger.SetLevel(logger.WARN)
	case "ERROR":
		logger.SetLevel(logger.ERROR)
	default:
		logger.SetLevel(logger.INFO)
	}

	// 获取Ollama配置
	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://10.0.10.112:11434" // 默认Ollama URL
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "gpt-oss:20b" // 默认模型，使用更稳定的模型
	}

	logger.Info("加载配置", map[string]interface{}{
		"ollama_url": ollamaURL,
		"ollama_model": ollamaModel,
	})

	// 创建LLM客户端 (Ollama)
	llmClient := llm.NewOllamaClient(ollamaURL, ollamaModel, 1000)

	// 创建工具管理器
	toolManager := tools.NewToolManager()

	// 注册一个简单的计算器工具
	calculator := &CalculatorTool{}
	toolManager.RegisterTool(calculator.Name(), calculator)

	// 注册联网搜索工具
	searchAPIKey := os.Getenv("SEARCH_API_KEY")
	webSearch := tools.NewWebSearchTool(searchAPIKey)
	toolManager.RegisterTool(webSearch.Name(), webSearch)

	// 注册本地知识库工具
	knowledgeBasePath := os.Getenv("KNOWLEDGE_BASE_PATH")
	if knowledgeBasePath == "" {
		knowledgeBasePath = "./knowledge_base" // 默认知识库路径
	}
	knowledgeBase := tools.NewKnowledgeBaseTool(knowledgeBasePath)
	toolManager.RegisterTool(knowledgeBase.Name(), knowledgeBase)

	// 获取Agent Prompt
	agentPrompt := os.Getenv("AGENT_PROMPT")
	if agentPrompt == "" {
		agentPrompt = `你是一位智能AI助手。

当需要使用工具时，请使用以下格式之一：

方法1 - JSON格式（推荐）：
{"tool":"tool_name","params":{"param1":"value1"}}

方法2 - Markdown格式：
` + "```tool:tool_name\n{\"param1\":\"value1\"}\n```" + `

可用工具：
1. web_search: 联网搜索
2. knowledge_base: 本地知识库 (list/read/search)
3. calculator: 计算器`
	}

	// 创建Agent配置
	config := agent.Config{
		Name:        "EinoAgent",
		Description: "A simple AI agent built with Eino",
		ModelConfig: agent.ModelConfig{
			Provider:  "ollama",
			ModelName: ollamaModel,
			BaseURL:   ollamaURL,
			MaxTokens: 1000,
			Prompt:    agentPrompt,
		},
	}

	// 创建Agent
	myAgent := agent.NewEinoAgent(config)

	// 初始化Agent
	ctx := context.Background()
	err = myAgent.Initialize(ctx, llmClient, toolManager)
	if err != nil {
		logger.Fatalf("初始化Agent失败: %v", err)
	}

	// 解析命令行参数
	webMode := flag.Bool("web", false, "启动Web模式")
	cliMode := flag.Bool("cli", false, "启动CLI对话模式")
	port := flag.String("port", "8080", "Web服务器端口")
	flag.Parse()

	if *webMode {
		// 启动Web服务器
		logger.Infof("启动Web模式，服务器运行在 http://localhost:%s", *port)
		server := api.NewServer(myAgent)
		server.Start(*port)
	} else if *cliMode {
		// CLI对话模式 - 使用英文提示避免中文编码问题
		fmt.Println("Welcome to Eino AI Assistant (type 'exit' to quit)")
		fmt.Println("------------------------------")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("\n> ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "exit" {
				fmt.Println("Goodbye!")
				break
			}

			if input == "" {
				continue
			}

			fmt.Println("Thinking...")
			response, err := myAgent.Process(ctx, input)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Println("\n" + response)
		}
	} else {
		// 命令行模式 - 改为交互式对话，使用流式处理：
		fmt.Println("欢迎使用 Eino AI 助手 (输入 'exit' 退出)")
		fmt.Println("------------------------------")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("\n> ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "exit" {
				fmt.Println("再见!")
				break
			}

			if input == "" {
				continue
			}

			fmt.Println("思考中...")

			// 使用流式处理
			responseChan := make(chan string, 100)

			// 启动goroutine来处理流式响应
			go func() {
				err := myAgent.ProcessStream(ctx, input, responseChan)
				if err != nil {
					fmt.Printf("\n错误: %v\n", err)
				}
			}()

			// 实时显示响应
			fmt.Print("\n")
			for chunk := range responseChan {
				fmt.Print(chunk)
			}
			fmt.Println() // 换行
		}
	}
}

// CalculatorTool 是一个简单的计算器工具
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "A simple calculator that can perform basic arithmetic operations"
}

func (t *CalculatorTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 这里简化实现，实际应用中需要更完善的逻辑
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	a, ok := params["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("a parameter is required")
	}

	b, ok := params["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("b parameter is required")
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return result, nil
}
