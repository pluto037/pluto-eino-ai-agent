<div align="center">

# 🤖 Eino AI Agent

一个轻量、可扩展的本地 AI Agent 系统

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[特性](#-特性) • [快速开始](#-快速开始) • [API 文档](#-api-文档) • [开发指南](#-开发指南)

</div>

---

## ✨ 特性

### 🚀 智能对话
- **流式响应（SSE）** - 实时打字效果，边生成边显示
- **Markdown 渲染** - 完美支持 Markdown 格式，代码语法高亮
- **思维链可视化** - 实时展示 Agent 思考过程（分析→工具调用→生成）
- **多格式工具调用** - 支持 JSON、Markdown 代码块等多种格式

### 💾 会话管理
- **持久化存储** - 会话自动保存到本地文件系统
- **会话列表** - ChatGPT 风格的侧边栏，快速切换历史对话
- **完整 API** - 创建、查询、删除、更新会话的 RESTful API

### 🛠️ 工具生态
- **工具调用闭环** - 自动识别、执行工具并将结果融入回复
- **本地知识库** - list/read/search 文档（`.txt/.md/.csv/.tsv`）
- **联网搜索** - DuckDuckGo（默认）或 SearchAPI（可选）
- **计算器** - 基础数学运算

### 📊 开发友好
- **结构化日志** - 彩色输出、调用位置追踪、多级别控制
- **健康检查** - `/health` 端点监控服务状态
- **双 LLM 支持** - Ollama 本地模型 + OpenAI API（均支持流式）

---

## 🚀 快速开始

### 环境要求

- Go 1.21+
- （可选）Ollama 本地模型服务

### 安装步骤

**1. 克隆项目**

```bash
git clone <your-repo-url>
cd pluto-eino-ai-agent
```

**2. 安装依赖**

```bash
go mod download
```

**3. 配置环境变量**

创建 `.env` 文件：

```bash
# 日志级别（可选）
LOG_LEVEL=INFO  # DEBUG/INFO/WARN/ERROR

# 数据存储路径
MEMORY_DATA_DIR=./data/conversations
KNOWLEDGE_BASE_PATH=./data/knowledge_base

# LLM 配置（选择其一）
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.1
# 或使用 OpenAI
# OPENAI_API_KEY=your-api-key

# 联网搜索（可选）
SEARCH_API_KEY=  # 留空使用 DuckDuckGo
```

**4. 启动服务**

```bash
# Web 模式（推荐）
go run main.go --web --port 8080

# CLI 模式
go run main.go --cli

# 默认交互模式（流式输出）
go run main.go
```

**5. 访问前端**

打开浏览器访问：`http://localhost:8080`

---

## 💡 使用指南

### Web 界面功能

- **实时对话** - 输入消息后实时流式响应，支持 Markdown 渲染
- **代码高亮** - 自动识别代码块并语法高亮
- **会话管理**
  - 左侧边栏查看所有历史会话
  - 点击会话标题切换对话
  - 点击 ✕ 删除会话
  - 点击「+ 新对话」创建新会话
- **思维过程** - 黄色提示框实时显示 Agent 思考步骤

### 工具调用格式

Agent 支持三种工具调用格式：

**方式 1：JSON 格式（推荐）**

```json
{"tool":"web_search","params":{"query":"Go并发编程"}}
{"tool":"knowledge_base","params":{"operation":"search","query":"向量检索"}}
{"tool":"calculator","params":{"operation":"add","a":10,"b":5}}
```

**方式 2：Markdown 代码块**

````markdown
```tool:web_search
{"query":"Go并发编程"}
```
````

**方式 3：简单文本（兼容旧版）**

```
使用工具: knowledge_base {"operation":"list"}
使用工具: web_search query=Go并发模式
```

---

## 🔌 API 文档

### 对话 API

**非流式对话** `POST /api/chat`

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"你好","conversation_id":"<可选>"}'
```

响应：
```json
{
  "reply": "你好！有什么我可以帮助你的吗？",
  "conversation_id": "web-layer-id",
  "agent_conversation_id": "agent-session-id"
}
```

**流式对话（SSE）** `GET /api/chat/stream`

```bash
curl -N "http://localhost:8080/api/chat/stream?message=你好&conversation_id=<可选>"
```

SSE 事件类型：
- `meta` - 会话元数据
- `data` - 消息内容片段
- `done` - 响应结束

### 会话管理 API

**列出所有会话** `GET /api/conversations`

```bash
curl http://localhost:8080/api/conversations
```

**获取会话详情** `GET /api/conversations/:id`

```bash
curl http://localhost:8080/api/conversations/conv_123
```

**删除会话** `DELETE /api/conversations/:id`

```bash
curl -X DELETE http://localhost:8080/api/conversations/conv_123
```

**更新会话标题** `PUT /api/conversations/:id/title`

```bash
curl -X PUT http://localhost:8080/api/conversations/conv_123 \
  -H "Content-Type: application/json" \
  -d '{"title":"新标题"}'
```

### 健康检查 API

**服务健康状态** `GET /health`

```bash
curl http://localhost:8080/health
# 响应: {"status":"healthy","timestamp":1234567890}
```

---

## 🛠️ 内置工具

### knowledge_base（本地知识库）

**功能**：管理和检索本地文档

**支持格式**：`.txt` `.md` `.csv` `.tsv`

**操作类型**：
- `list` - 列出所有文档
- `read` - 读取指定文档内容
- `search` - 关键词搜索（CSV/TSV 会标注行号）

**使用示例**：

```json
{"tool":"knowledge_base","params":{"operation":"list"}}
{"tool":"knowledge_base","params":{"operation":"read","document":"example.md"}}
{"tool":"knowledge_base","params":{"operation":"search","query":"向量检索"}}
```

### web_search（联网搜索）

**功能**：实时搜索互联网信息

**搜索引擎**：
- 默认：DuckDuckGo（无需配置）
- 可选：SearchAPI（需配置 `SEARCH_API_KEY`）

**使用示例**：

```json
{"tool":"web_search","params":{"query":"Go 泛型教程"}}
{"tool":"web_search","params":{"query":"最新 AI 资讯"}}
```

### calculator（计算器）

**功能**：基础数学运算

**支持操作**：`add` `subtract` `multiply` `divide`

**使用示例**：

```json
{"tool":"calculator","params":{"operation":"add","a":10,"b":5}}
{"tool":"calculator","params":{"operation":"multiply","a":7,"b":8}}
```

---

## 📁 项目结构

```
├── pkg/
│   ├── agent/            # Agent 核心逻辑
│   │   └── agent.go      # 工具调用闭环、思维链、流式处理
│   ├── api/              # HTTP 服务层
│   │   └── server.go     # RESTful API、SSE 流式、会话管理
│   ├── llm/              # LLM 客户端
│   │   ├── ollama.go     # Ollama 本地模型（流式支持）
│   │   └── openai.go     # OpenAI API（流式支持）
│   ├── memory/           # 记忆系统
│   │   └── memory.go     # 会话持久化、向量存储接口
│   ├── logger/           # 日志系统
│   │   └── logger.go     # 结构化彩色日志
│   └── tools/            # 工具生态
│       ├── tool_manager.go    # 工具管理器
│       ├── knowledge_base.go  # 知识库工具
│       └── web_search.go      # 搜索工具
├── web/static/
│   └── index.html        # Web 前端（Markdown、代码高亮、会话管理）
├── data/
│   ├── conversations/    # 会话数据存储
│   └── knowledge_base/   # 知识库文档
├── main.go               # 程序入口
├── go.mod                # Go 模块依赖
└── .env                  # 环境配置
```

---

## ❓ 常见问题

### Q: 如何查看详细日志？
A: 设置环境变量 `LOG_LEVEL=DEBUG`，重启服务即可看到详细日志输出。

### Q: SSE 流式响应不工作？
A: 
- 确认浏览器支持 EventSource
- 检查是否有代理缓存（Nginx 需设置 `X-Accel-Buffering: no`）
- 系统会自动回退到非流式 POST 请求

### Q: 知识库没有文档？
A: 首次运行会自动创建示例文档，将你的文档放入 `KNOWLEDGE_BASE_PATH` 指定目录。

### Q: 如何切换 LLM 提供商？
A: 修改 `.env` 文件中的配置，支持 Ollama 和 OpenAI。

### Q: 工具调用不生效？
A: 
- 优先使用 JSON 格式：`{"tool":"...","params":{...}}`
- 检查参数是否正确
- 查看日志中的工具执行状态

### Q: 如何清理历史会话？
A: 使用 DELETE API 或直接删除 `data/conversations/` 目录下的文件。

---

## 🔧 开发指南

### 构建项目

```bash
# 编译
go build -o eino-agent main.go

# 运行
./eino-agent --web --port 8080
```

### 添加自定义工具

1. 在 `pkg/tools/` 创建新工具文件
2. 实现 `Tool` 接口：

```go
type CustomTool struct{}

func (t *CustomTool) Name() string {
    return "custom_tool"
}

func (t *CustomTool) Description() string {
    return "工具描述"
}

func (t *CustomTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // 工具逻辑
    return result, nil
}
```

3. 在 `main.go` 注册工具：

```go
customTool := &CustomTool{}
toolManager.RegisterTool(customTool.Name(), customTool)
```

### 日志级别

```go
import "agentEino/pkg/logger"

logger.Debug("调试信息", map[string]interface{}{"key": "value"})
logger.Info("普通信息")
logger.Warn("警告信息")
logger.Error("错误信息")
logger.Fatal("致命错误")  // 会退出程序
```

### 代码规范

- 保持模块化和可扩展性
- 使用结构化日志记录关键操作
- 工具和记忆模块独立，便于扩展
- 遵循 Go 标准代码风格

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

**可以贡献的方向**：
- 🛠️ 新工具开发（文件操作、HTTP 请求、数据库查询等）
- 🎨 前端界面优化（主题切换、移动端适配）
- 🔍 向量检索增强（RAG、语义搜索）
- 📊 监控与统计（Prometheus 指标、使用分析）
- 🧪 测试覆盖（单元测试、集成测试）
- 📝 文档改进（API 文档、使用教程）

**开发流程**：
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

---

## ⭐ Star History

如果这个项目对你有帮助，欢迎 Star 支持！

---

## 📄 许可证

MIT License

Copyright (c) 2025 Pluto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
