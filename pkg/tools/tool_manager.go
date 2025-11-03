package tools

import (
	"context"
	"errors"
	"sync"
)

// Tool 是工具的接口
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolManager 管理可用的工具
type ToolManager struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewToolManager 创建一个新的工具管理器
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册一个工具
func (tm *ToolManager) RegisterTool(name string, tool Tool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tools[name]; exists {
		return errors.New("tool already registered")
	}

	tm.tools[name] = tool
	return nil
}

// GetTool 获取一个工具
func (tm *ToolManager) GetTool(name string) (Tool, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tool, exists := tm.tools[name]
	return tool, exists
}

// ListTools 列出所有工具
func (tm *ToolManager) ListTools() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tools := make([]string, 0, len(tm.tools))
	for name := range tm.tools {
		tools = append(tools, name)
	}
	return tools
}

// ExecuteTool 执行指定的工具
func (tm *ToolManager) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, exists := tm.GetTool(name)
	if !exists {
		return nil, errors.New("tool not found")
	}

	return tool.Execute(ctx, params)
}
