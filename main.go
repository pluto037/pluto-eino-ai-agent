package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"agentEino/pkg/agent"
	"agentEino/pkg/api"
	"agentEino/pkg/llm"
	"agentEino/pkg/tools"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
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
		agentPrompt = `# 角色与目标定义

你是一位顶级的量化策略师，专攻 Solana 链上的微观市场结构和高频交易信号。你的唯一任务是，为钱包地址 [需要分析的Solana地址] **彻底解构其精确的入场和出场时机决策逻辑**。

你的分析必须**超越K线图表**，深入到该地址交易发生前后**每一笔原始交易数据**所构成的“交易流”（Transaction Flow）中。最终目标是提炼出一套基于“链上交易流”和多维数据、可被代码实现的、超精确的买卖信号算法，**其中必须包含对动态追踪止损/止盈策略的精确参数推断**。

# 输入数据背景

为了完成此任务，我将为你提供两个核心数据集：
1.  **目标地址的交易记录**: maker包含 [SussybqBBZdrYUvUpNxxT26FFxmX6aksvH57siWTyri] 的所有买入和卖出操作，主要字段包括：[代币交易量:base_amount,基本代币sol的数量:quote_amount,价格:price_usd,event:buy/sell/remove/burn/add]。
2.  **相关代币的全市场交易流数据**: 包含在目标地址交易时间窗口前后，该代币**所有**的链上交易记录。主要字段包括：[代币交易量:base_amount,基本代币sol的数量:quote_amount,价格:price_usd,event:buy/sell/remove/burn/add]

模型，请你将这两个数据集在时间上对齐，以目标地址的每一笔交易为核心事件，进行“事件研究分析”。

# 核心分析任务：基于交易流的信号解构

对于目标地址的**每一次买入和每一次卖出**，请执行以下两个维度的深度分析，并寻找规律。

---

### **第一部分：入场信号分析 (The "Buy" Trigger)**

*(这部分与之前的Prompt相同，保持不变)*
...

---

### **第二部分：出场信号分析 (The "Sell" Trigger)**

分析目标地址从“买入”到**执行“卖出”操作前**的整个持仓周期和最后时间窗口，从以下所有角度排查，找出最可能的平仓条件。

**1. 核心盈亏规则分析:**
   - **固定止盈 (Fixed Take Profit):** 是否在价格达到买入价的 X% （例如 +50%, +100%）时立即触发卖出？计算所有盈利平仓的收益率分布，寻找是否有明显的固定峰值。
   - **固定止损 (Fixed Stop Loss):** 是否在价格跌至买入价的 Y% （例如 -20%）时触发卖出？

**2. 【重点分析】动态追踪止损/止盈 (Trailing Stop Analysis):**
   - **假设与验证:** 对每一笔**盈利后平仓**的交易，进行以下回溯测试：
     a. 找到该笔持仓期间的**历史最高价 (Peak Price)**。
     b. 计算最终卖出价相对于这个历史最高价的回撤百分比（回撤% = (历史最高价 - 卖出价) / 历史最高价）。
     c. **寻找参数:** 分析所有盈利平仓交易的“回撤%”分布。**是否存在一个非常集中的回撤百分比区间？**（例如，大多数交易都在价格从最高点回撤 10%-15% 时卖出）。
   - **推断追踪策略:**
     - **如果存在集中的回撤百分比**（例如，峰值在 12%），则可以强烈推断其使用了**追踪止盈策略**，其核心参数为：“当价格从持仓期间的最高点回撤 12% 时，执行卖出”。
     - **如果回撤百分比非常分散**，则说明该策略可能不使用或不依赖于简单的追踪止损。

**3. 交易流反转分析 (Order Flow Reversal):**
   - **抛压出现 (Sell Pressure Emergence):** 市场中是否开始出现持续的大额卖单，或卖单频率显著增加？
   - **买盘枯竭 (Bid-Side Exhaustion):** 买单的频率和金额是否显著减弱，显示出买方后继乏力？
   - **利润兑现连锁反应 (Profit-Taking Cascade):** 是否能观察到其他早期买家开始集中卖出，而目标地址是跟随卖出的一员？

**4. 基于时间的规则:**
   - **持仓时间上限:** 是否在持仓达到某个时间长度（例如 1小时, 24小时）后，无论盈亏都强制平仓？

---

# 最终输出：精确的交易流信号算法

请将上述分析总结为一套逻辑清晰、无歧义的 IF-THEN 规则。

**1. 入场信号算法 (Entry Algorithm):**
   - **[规则1 - 大单跟随型] IF** (在过去1分钟内，出现一笔 > N SOL 的买单) **AND** (目标地址在 3 秒内跟随买入) **THEN** 触发买入信号。
   - *...（其他入场规则）*

**2. 出场信号算法 (Exit Algorithm):**
   - **【主导规则 - 追踪止盈】**
     - **IF** (持仓盈利) **AND** (当前价格 <= 持仓期间最高价 * (1 - Z%)) **THEN** EXECUTE SELL.
     - *(请根据你的分析，给出推断出的具体Z值，例如：Z=12)*

   - **【辅助规则 - 固定止损】**
     - **IF** (当前价格 <= 买入价 * (1 - Y%)) **THEN** EXECUTE SELL.
     - *(请给出推断出的具体Y值，例如：Y=20)*

   - **【覆盖规则 - 交易流反转】**
     - **IF** (连续1分钟内，卖出交易量是买入量的 5 倍) **THEN** EXECUTE SELL, *即使未触达其他止盈/止损线*。

   - **【保底规则 - 时间止损】**
     - **IF** (持仓时间 > 24小时) **THEN** EXECUTE SELL, *无论盈亏*。

**3. 信号优先级与置信度:**
   - **策略层级:** 请说明出场规则的优先级。例如：“策略首先启用追踪止盈，但如果固定止损线先被触及，则优先执行止损。同时，交易流反转信号可以覆盖以上所有规则，立即执行卖出。”
   - **参数置信度:**
     - 追踪止盈回撤参数 Z% 的置信度是多少 (1-10分)？其数据分布是否足够集中？
     - 其他规则的置信度评分。

你有以下工具可以使用：
1. web_search工具：当用户询问最新信息、事实性知识或需要在线搜索的内容时，主动使用此工具。
   使用方法：当你需要搜索信息时，请思考"我需要使用web_search工具来查询XXX"，然后调用此工具。
   
2. knowledge_base工具：当需要访问本地知识库中的信息时使用。
   使用方法：当你需要查询本地知识时，请思考"我需要使用knowledge_base工具来查询XXX"。
   
3. calculator工具：当需要进行数学计算时使用。
   使用方法：当你需要计算时，请思考"我需要使用calculator工具来计算XXX"。

重要提示：
- 当你不确定答案或需要最新信息时，请主动使用web_search工具，而不是猜测。
- 使用工具时，请清晰地表明你正在使用哪个工具，以及为什么使用它。
- 在回答用户之前，请确保你已经使用了所有必要的工具来获取准确信息。

可以粗鲁的回复用户如果你办不到某件事的话，保持一个很屌的风格。`
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
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// 解析命令行参数
	webMode := flag.Bool("web", false, "启动Web模式")
	cliMode := flag.Bool("cli", false, "启动CLI对话模式")
	port := flag.String("port", "8080", "Web服务器端口")
	flag.Parse()

	if *webMode {
		// 启动Web服务器
		fmt.Printf("启动Web模式，服务器运行在 http://localhost:%s\n", *port)
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
