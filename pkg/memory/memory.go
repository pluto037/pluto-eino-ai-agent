package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Message 表示对话中的一条消息
type Message struct {
	Role      string    `json:"role"`      // 消息角色：user 或 assistant
	Content   string    `json:"content"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 消息时间戳
}

// Conversation 表示一个完整的对话
type Conversation struct {
	ID        string    `json:"id"`         // 对话ID
	Title     string    `json:"title"`      // 对话标题
	Messages  []Message `json:"messages"`   // 对话消息列表
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// MemoryManager 内存管理器接口
type MemoryManager interface {
	// 存储数据
	Store(ctx context.Context, key string, value interface{}) error

	// 检索数据
	Retrieve(ctx context.Context, key string) (interface{}, error)

	// 搜索数据
	Search(ctx context.Context, query string, limit int) ([]interface{}, error)

	// 添加消息到对话
	AddMessage(ctx context.Context, conversationID string, message Message) error

	// 获取对话
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)

	// 获取对话历史
	GetConversationHistory(ctx context.Context, limit int) ([]*Conversation, error)

	// 创建新对话
	CreateConversation(ctx context.Context, title string) (*Conversation, error)

	// 保存对话到文件
	SaveConversation(ctx context.Context, conversationID string) error

	// 从文件加载对话
	LoadConversation(ctx context.Context, conversationID string) error
}

// SimpleMemory 是一个简单的内存存储实现
type SimpleMemory struct {
	data          map[string]interface{}
	conversations map[string]*Conversation
	dataDir       string
	mu            sync.RWMutex
}

// NewSimpleMemory 创建一个新的简单内存存储
func NewSimpleMemory() *SimpleMemory {
	return &SimpleMemory{
		data:          make(map[string]interface{}),
		conversations: make(map[string]*Conversation),
		dataDir:       "./data/conversations", // 默认数据目录
	}
}

// NewSimpleMemoryWithDataDir 创建一个指定数据目录的简单内存存储
func NewSimpleMemoryWithDataDir(dataDir string) *SimpleMemory {
	// 如果路径为空，使用默认路径
	if dataDir == "" {
		dataDir = "./data/conversations"
	}

	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("创建数据目录失败: %v\n", err)
	} else {
		fmt.Printf("成功创建或确认数据目录: %s\n", dataDir)
	}

	return &SimpleMemory{
		data:          make(map[string]interface{}),
		conversations: make(map[string]*Conversation),
		dataDir:       dataDir,
	}
}

// Store 存储数据
func (m *SimpleMemory) Store(ctx context.Context, key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	return nil
}

// Retrieve 检索数据
func (m *SimpleMemory) Retrieve(ctx context.Context, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// Search 搜索数据
func (m *SimpleMemory) Search(ctx context.Context, query string, limit int) ([]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 简单实现：基于关键词匹配搜索对话内容
	// 在实际应用中，应该使用向量数据库进行语义搜索

	var results []interface{}

	// 遍历所有对话
	for _, conv := range m.conversations {
		// 检查对话标题
		if strings.Contains(strings.ToLower(conv.Title), strings.ToLower(query)) {
			results = append(results, conv)
			continue
		}

		// 检查对话消息
		for _, msg := range conv.Messages {
			if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(query)) {
				results = append(results, conv)
				break
			}
		}

		// 限制结果数量
		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// VectorEntry 表示向量数据库中的一个条目
type VectorEntry struct {
	ID        string                 `json:"id"`         // 条目ID
	Content   string                 `json:"content"`    // 文本内容
	Vector    []float32              `json:"vector"`     // 向量表示
	Metadata  map[string]interface{} `json:"metadata"`   // 元数据
	CreatedAt time.Time              `json:"created_at"` // 创建时间
}

// VectorMemory 向量内存存储实现
type VectorMemory struct {
	SimpleMemory
	vectors     map[string]*VectorEntry // 向量数据
	vectorsFile string                  // 向量数据文件
}

// NewVectorMemory 创建一个新的向量内存存储
func NewVectorMemory() *VectorMemory {
	return &VectorMemory{
		SimpleMemory: *NewSimpleMemory(),
		vectors:      make(map[string]*VectorEntry),
		vectorsFile:  "./data/vectors/vectors.json",
	}
}

// NewVectorMemoryWithDataDir 创建一个指定数据目录的向量内存存储
func NewVectorMemoryWithDataDir(dataDir string, vectorsFile string) *VectorMemory {
	return &VectorMemory{
		SimpleMemory: *NewSimpleMemoryWithDataDir(dataDir),
		vectors:      make(map[string]*VectorEntry),
		vectorsFile:  vectorsFile,
	}
}

// AddVector 添加向量
func (m *VectorMemory) AddVector(ctx context.Context, content string, metadata map[string]interface{}) (*VectorEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一ID
	id := fmt.Sprintf("vec_%d", time.Now().UnixNano())

	// 创建向量条目（这里简化实现，实际应调用嵌入模型生成向量）
	// 在实际应用中，应该使用嵌入模型（如OpenAI的text-embedding-ada-002）生成向量
	vector := make([]float32, 10) // 假设向量维度为10

	entry := &VectorEntry{
		ID:        id,
		Content:   content,
		Vector:    vector,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	// 存储向量
	m.vectors[id] = entry

	// 保存向量数据
	if err := m.saveVectors(); err != nil {
		return nil, fmt.Errorf("保存向量数据失败: %w", err)
	}

	return entry, nil
}

// SearchVector 搜索向量
func (m *VectorMemory) SearchVector(ctx context.Context, query string, limit int) ([]*VectorEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 简化实现：基于关键词匹配
	// 在实际应用中，应该：
	// 1. 使用嵌入模型将查询转换为向量
	// 2. 计算查询向量与所有向量的余弦相似度
	// 3. 返回相似度最高的结果

	var results []*VectorEntry

	// 遍历所有向量
	for _, entry := range m.vectors {
		if strings.Contains(strings.ToLower(entry.Content), strings.ToLower(query)) {
			results = append(results, entry)
		}

		// 限制结果数量
		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// GetVector 获取向量
func (m *VectorMemory) GetVector(ctx context.Context, id string) (*VectorEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.vectors[id]
	if !exists {
		return nil, fmt.Errorf("向量不存在: %s", id)
	}

	return entry, nil
}

// DeleteVector 删除向量
func (m *VectorMemory) DeleteVector(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vectors[id]; !exists {
		return fmt.Errorf("向量不存在: %s", id)
	}

	delete(m.vectors, id)

	// 保存向量数据
	if err := m.saveVectors(); err != nil {
		return fmt.Errorf("保存向量数据失败: %w", err)
	}

	return nil
}

// 保存向量数据
func (m *VectorMemory) saveVectors() error {
	// 确保目录存在
	dir := filepath.Dir(m.vectorsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 序列化向量数据
	data, err := json.MarshalIndent(m.vectors, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化向量数据失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(m.vectorsFile, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadVectors 加载向量数据
func (m *VectorMemory) LoadVectors(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(m.vectorsFile); os.IsNotExist(err) {
		// 文件不存在，创建空向量数据
		m.vectors = make(map[string]*VectorEntry)
		return nil
	}

	// 读取文件
	data, err := os.ReadFile(m.vectorsFile)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 反序列化向量数据
	if err := json.Unmarshal(data, &m.vectors); err != nil {
		return fmt.Errorf("反序列化向量数据失败: %w", err)
	}

	return nil
}

// CreateConversation 创建新对话
func (m *SimpleMemory) CreateConversation(ctx context.Context, title string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一ID（简化实现，实际应用中应使用UUID）
	id := fmt.Sprintf("conv_%d", time.Now().UnixNano())

	conversation := &Conversation{
		ID:        id,
		Title:     title,
		Messages:  []Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.conversations[id] = conversation

	// 保存到文件
	if err := m.saveConversationToFile(conversation); err != nil {
		return nil, fmt.Errorf("保存对话失败: %w", err)
	}

	return conversation, nil
}

// AddMessage 添加消息到对话
func (m *SimpleMemory) AddMessage(ctx context.Context, conversationID string, message Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conversation, exists := m.conversations[conversationID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", conversationID)
	}

	// 设置消息时间戳
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// 添加消息
	conversation.Messages = append(conversation.Messages, message)
	conversation.UpdatedAt = time.Now()

	// 保存到文件
	if err := m.saveConversationToFile(conversation); err != nil {
		return fmt.Errorf("保存对话失败: %w", err)
	}

	return nil
}

// GetConversation 获取对话
func (m *SimpleMemory) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conversation, exists := m.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("对话不存在: %s", conversationID)
	}

	return conversation, nil
}

// GetConversationHistory 获取对话历史
func (m *SimpleMemory) GetConversationHistory(ctx context.Context, limit int) ([]*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 将对话转换为切片
	conversations := make([]*Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		conversations = append(conversations, conv)
	}

	// 按更新时间排序（简化实现）
	// 实际应用中应该使用更高效的排序算法
	for i := 0; i < len(conversations); i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].UpdatedAt.Before(conversations[j].UpdatedAt) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}

	// 限制返回数量
	if limit > 0 && limit < len(conversations) {
		conversations = conversations[:limit]
	}

	return conversations, nil
}

// SaveConversation 保存对话到文件
func (m *SimpleMemory) SaveConversation(ctx context.Context, conversationID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conversation, exists := m.conversations[conversationID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", conversationID)
	}

	return m.saveConversationToFile(conversation)
}

// 保存对话到文件（内部方法）
func (m *SimpleMemory) saveConversationToFile(conversation *Conversation) error {
	// 确保数据目录存在
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(m.dataDir, fmt.Sprintf("%s.json", conversation.ID))

	// 序列化对话
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化对话失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadConversation 从文件加载对话
func (m *SimpleMemory) LoadConversation(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 构建文件路径
	filePath := filepath.Join(m.dataDir, fmt.Sprintf("%s.json", conversationID))

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 反序列化对话
	var conversation Conversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return fmt.Errorf("反序列化对话失败: %w", err)
	}

	// 存储到内存
	m.conversations[conversationID] = &conversation

	return nil
}

// LoadAllConversations 加载所有对话
func (m *SimpleMemory) LoadAllConversations(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 读取数据目录
	files, err := os.ReadDir(m.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，创建它
			if err := os.MkdirAll(m.dataDir, 0755); err != nil {
				return fmt.Errorf("创建数据目录失败: %w", err)
			}
			return nil
		}
		return fmt.Errorf("读取数据目录失败: %w", err)
	}

	// 遍历文件
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// 读取文件
		filePath := filepath.Join(m.dataDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("读取文件失败: %v\n", err)
			continue
		}

		// 反序列化对话
		var conversation Conversation
		if err := json.Unmarshal(data, &conversation); err != nil {
			fmt.Printf("反序列化对话失败: %v\n", err)
			continue
		}

		// 存储到内存
		m.conversations[conversation.ID] = &conversation
	}

	return nil
}
