package api

import (
	"agentEino/pkg/agent"
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Server 表示Web API服务器
type Server struct {
	agent         agent.Agent
	conversations map[string]*Conversation
	// 将 Web 层的 conversation_id 映射到 Agent 层的记忆会话ID
	agentConvMap map[string]string
	mu           sync.Mutex
}

// Conversation 表示一个对话会话
type Conversation struct {
	ID        string
	Messages  []Message
	Context   context.Context
	CreatedAt int64
}

// Message 表示对话中的一条消息
type Message struct {
	Role    string `json:"role"`    // "user" 或 "assistant"
	Content string `json:"content"` // 消息内容
}

// ChatRequest 表示聊天请求
type ChatRequest struct {
	ConversationID string `json:"conversation_id,omitempty"`
	Message        string `json:"message"`
}

// ChatResponse 表示聊天响应
type ChatResponse struct {
	ConversationID string  `json:"conversation_id"`
	Message        Message `json:"message"`
}

// NewServer 创建一个新的API服务器
func NewServer(agent agent.Agent) *Server {
	return &Server{
		agent:         agent,
		conversations: make(map[string]*Conversation),
		agentConvMap:  make(map[string]string),
	}
}

// Start 启动Web服务器
func (s *Server) Start(port string) {
	// 设置静态文件服务
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/", fs)

	// API路由
	http.HandleFunc("/api/chat", s.handleChat)
	http.HandleFunc("/api/chat/stream", s.handleChatStream)

	log.Printf("Starting web server on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleChat 处理聊天请求
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	var conv *Conversation
	var exists bool

	// 获取或创建对话
	if req.ConversationID != "" {
		conv, exists = s.conversations[req.ConversationID]
	}

	if !exists {
		// 创建新对话
		conv = &Conversation{
			ID:        generateID(),
			Messages:  []Message{},
			Context:   context.Background(),
			CreatedAt: currentTimestamp(),
		}
		s.conversations[conv.ID] = conv
		// 绑定到当前 Agent 的会话ID
		if s.agent != nil {
			s.agentConvMap[conv.ID] = s.agent.GetConversationID()
		}
	}
	// 确保 Agent 切换到该会话对应的记忆ID
	if s.agent != nil {
		if aid, ok := s.agentConvMap[conv.ID]; ok && aid != "" {
			_ = s.agent.SetConversationID(aid)
		}
	}
	s.mu.Unlock()

	// 添加用户消息
	userMsg := Message{
		Role:    "user",
		Content: req.Message,
	}
	conv.Messages = append(conv.Messages, userMsg)

	// 处理消息并获取响应
	response, err := s.agent.Process(conv.Context, req.Message)
	if err != nil {
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	// 添加助手响应
	assistantMsg := Message{
		Role:    "assistant",
		Content: response,
	}
	conv.Messages = append(conv.Messages, assistantMsg)

	// 返回响应
	resp := ChatResponse{
		ConversationID: conv.ID,
		Message:        assistantMsg,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(resp)
}

// handleChatStream 处理SSE流式聊天
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析查询参数
	conversationID := r.URL.Query().Get("conversation_id")
	message := r.URL.Query().Get("message")
	if strings.TrimSpace(message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// 获取或创建对话
	s.mu.Lock()
	var conv *Conversation
	var exists bool
	if conversationID != "" {
		conv, exists = s.conversations[conversationID]
	}
	if !exists {
		conv = &Conversation{
			ID:        generateID(),
			Messages:  []Message{},
			Context:   context.Background(),
			CreatedAt: currentTimestamp(),
		}
		s.conversations[conv.ID] = conv
		// 将新会话绑定到当前Agent会话ID
		if s.agent != nil {
			s.agentConvMap[conv.ID] = s.agent.GetConversationID()
		}
	}
	// 获取绑定的Agent会话ID
	var agentConvID string
	if s.agent != nil {
		if aid, ok := s.agentConvMap[conv.ID]; ok {
			agentConvID = aid
			_ = s.agent.SetConversationID(aid)
		} else {
			agentConvID = s.agent.GetConversationID()
			s.agentConvMap[conv.ID] = agentConvID
		}
	}
	// 添加用户消息到会话缓存
	userMsg := Message{Role: "user", Content: message}
	conv.Messages = append(conv.Messages, userMsg)
	s.mu.Unlock()

	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Nginx/代理禁用缓冲（可选）
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 先发送元数据事件，通知前端会话ID（新会话时）
	meta := struct {
		ConversationID      string `json:"conversation_id"`
		AgentConversationID string `json:"agent_conversation_id"`
	}{ConversationID: conv.ID, AgentConversationID: agentConvID}
	metaBytes, _ := json.Marshal(meta)
	_, _ = w.Write([]byte("event: meta\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(metaBytes)
	_, _ = w.Write([]byte("\n\n"))
	flusher.Flush()

	// 准备流式通道
	streamChan := make(chan string, 100)

	// 启动Agent流式处理（包含工具闭环）
	go func() {
		// 使用请求上下文以便断开时取消
		_ = s.agent.ProcessStream(r.Context(), message, streamChan)
	}()

	// 将流式内容转发为SSE data事件
	for {
		select {
		case <-r.Context().Done():
			close(streamChan)
			return
		case chunk, ok := <-streamChan:
			if !ok {
				// 结束事件
				_, _ = w.Write([]byte("event: done\n"))
				_, _ = w.Write([]byte("data: done\n\n"))
				flusher.Flush()
				return
			}
			// 正常数据块
			esc, _ := json.Marshal(chunk)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(esc)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

// 生成唯一ID
func generateID() string {
	// 简单实现，实际应用中应使用UUID库
	return "conv_" + randomString(10)
}

// 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[randomInt(0, len(charset))]
	}
	return string(result)
}

// 生成随机整数
func randomInt(min, max int) int {
	return min + randomInt64(int64(max-min))
}

// 生成随机int64
func randomInt64(max int64) int {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return int(time.Now().UnixNano() % max)
	}
	return int(n.Int64())
}

// 获取当前时间戳
func currentTimestamp() int64 {
	return time.Now().UnixNano()
}
