# Eino AI Agent

一个轻量、可扩展的本地 AI Agent，内置会话记忆、SSE 流式输出、工具调用闭环、本地知识库，以及简洁的 Web 前端。

支持如下能力：
- 流式回复（SSE）与自动回退（POST），前端边生成边渲染。
- 会话持久化与绑定：Web `conversation_id` ↔ Agent 会话ID 映射，历史写入 `data/conversations/`。
- 工具调用闭环：智能识别指令并调用工具，将结果写入上下文后再生成最终回答。
- 本地知识库工具：列出/读取/搜索文档，支持 `.txt/.md/.csv/.tsv`。
- 联网搜索工具：DuckDuckGo 默认，支持可选 `SearchAPI`（如提供 API Key）。

---

## 快速开始

**环境要求**
- `Go` 1.21+（或兼容版本）
- 可选：`Ollama` 本地模型服务（如启用本地推理）

**拉取与依赖**
- `git clone <your-repo-url>`
- 在项目根目录执行：`go mod download`

**配置 `.env`（示例）**
```
MEMORY_DATA_DIR=./data/conversations
KNOWLEDGE_BASE_PATH=./data/knowledge_base

# 选择其一：Ollama 本地模型 或 OpenAI API
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.1

# 搜索（可选），留空则默认使用 DuckDuckGo
SEARCH_API_KEY=
```

> 注：首次运行会自动创建会话数据目录与示例知识库文档。

**启动 Web 前端**
- `go run main.go --web --port 8080`
- 打开 `http://localhost:8080/`

---

## 使用说明（Web）

- 直接在输入框对话，回复为 SSE 流式输出；网络或浏览器不支持 SSE 时自动回退至非流式。
- 会话在左侧持续存在；系统会绑定并持久化上下文（记忆）。
- 需要调用工具时，可以在对话中明确给出指令，例如：
  - 本地知识库搜索（JSON 参数）：
    - `使用工具: knowledge_base {"operation":"search","query":"向量检索"}`
  - 本地知识库读取（JSON 参数）：
    - `使用工具: knowledge_base {"operation":"read","document":"example.md"}`
  - 列出知识库文档（kv 参数）：
    - `使用工具: knowledge_base operation=list`
  - 联网搜索（kv 参数）：
    - `使用工具: web_search query=Go 并发 模式`

> 参数格式支持两种：JSON 或 `key=value`；Agent 会自动解析并执行工具，随后把结果写入上下文再生成最终回复。

---

## REST API

**非流式** `POST /api/chat`
- 请求体：`{"message":"你好","conversation_id":"<可选>"}`
- 响应体：
```
{
  "reply": "...",
  "conversation_id": "<web层ID>",
  "agent_conversation_id": "<agent层会话ID>"
}
```

**流式（SSE）** `GET /api/chat/stream?message=...&conversation_id=...`
- 响应类型：`text/event-stream`
- 事件：
  - `event: meta` → `{"conversation_id":"...","agent_conversation_id":"..."}`
  - `event: message` → 文本增量片段（多次）
  - `event: done` → 结束信号

SSE 示例（`curl`）：
```
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/api/chat/stream?message=%E4%BD%A0%E5%A5%BD"
```

---

## 工具一览

**knowledge_base**（本地知识库）
- `operation=list`：列出知识库中文档（支持 `.txt/.md/.csv/.tsv`）
- `operation=read`：读取指定文档，返回全文字符串
- `operation=search`：搜索关键词，返回 {文件名: 命中行列表}；对 `.csv/.tsv` 会标注行号

示例：
```
使用工具: knowledge_base {"operation":"search","query":"示例"}
使用工具: knowledge_base operation=read document=example.md
```

**web_search**（联网搜索）
- 参数：`query`（必填）
- 默认引擎：DuckDuckGo；如配置 `SEARCH_API_KEY` 则使用 `SearchAPI`。

示例：
```
使用工具: web_search {"query":"Go 泛型 教程"}
```

---

## 项目结构

```
├── pkg/
│   ├── agent/            # Agent 核心逻辑（工具闭环、记忆绑定、流式生成）
│   ├── api/              # HTTP 服务与 SSE 端点
│   ├── llm/              # LLM 客户端（Ollama / OpenAI）
│   ├── memory/           # 会话记忆实现（简单/向量，可扩展）
│   └── tools/            # 工具（knowledge_base、web_search、manager）
├── web/static/index.html # 简洁 Web 前端（SSE 渲染与回退）
├── data/conversations/   # 会话持久化存储
└── .env                  # 环境配置
```

---

## 常见问题

- SSE 无法连接或被代理缓存：尝试本地浏览器访问；或使用 `curl -N`；必要时走非流式 `POST /api/chat`。
- 知识库没有文档：首次运行会创建示例；将文档放入 `KNOWLEDGE_BASE_PATH` 指定目录。
- DuckDuckGo 结果偏少：配置 `SEARCH_API_KEY` 以启用 `SearchAPI`。
- 本地推理失败：确认 `OLLAMA_BASE_URL` 与 `OLLAMA_MODEL` 正确，且 Ollama 已启动并拉取模型。

---

## 开发与构建

- 构建：`go build ./...`
- 运行（Web）：`go run main.go --web --port 8080`
- 代码风格：保持简洁、可扩展；工具与记忆模块便于拓展新能力。

---

## 贡献

欢迎提交 issue / PR：
- 新工具（如 `http_fetch`、受限 `file_ops` 等）
- 流式体验优化（SSE 心跳、前端取消）
- 向量检索与引用（RAG）
- 测试与并发稳定性提升

---

如果你喜欢这个项目，欢迎 Star 支持！

基于 Eino 构建的 AI Agent 框架，提供了一个灵活、可扩展的智能代理系统。

## 项目结构

```
agentEino/
├── pkg/
│   ├── agent/     # Agent 核心实现
│   ├── llm/       # LLM 集成
│   ├── memory/    # 记忆系统
│   ├── tools/     # 工具管理
│   ├── config/    # 配置管理
│   └── api/       # API 接口
└── main.go        # 主程序入口
```

## 快速开始

1. 复制环境变量示例文件并填写必要的配置：

```bash
cp .env.example .env
# 编辑 .env 文件，填写 OPENAI_API_KEY
```

2. 运行示例程序：

```bash
go run main.go
```

## 核心组件

- **Agent 引擎**：协调各组件工作，管理 Agent 的状态和生命周期
- **LLM 集成**：连接到 Eino 或其他 LLM 服务
- **工具管理器**：注册和管理可用工具
- **记忆系统**：管理短期对话记忆和长期知识存储
- **任务规划器**：分解复杂任务并制定执行计划

## 自定义工具

你可以通过实现 `tools.Tool` 接口来创建自定义工具：

```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}
```

## 许可证

MIT