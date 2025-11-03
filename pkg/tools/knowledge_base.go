package tools

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// KnowledgeBaseTool 实现了本地知识库查看功能
type KnowledgeBaseTool struct {
	basePath string
}

// NewKnowledgeBaseTool 创建一个新的知识库工具
func NewKnowledgeBaseTool(basePath string) *KnowledgeBaseTool {
	return &KnowledgeBaseTool{
		basePath: basePath,
	}
}

// Name 返回工具名称
func (t *KnowledgeBaseTool) Name() string {
	return "knowledge_base"
}

// Description 返回工具描述
func (t *KnowledgeBaseTool) Description() string {
	return "查看本地知识库中的文档"
}

// Execute 执行知识库查询
func (t *KnowledgeBaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 获取操作类型
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少操作类型参数")
	}

	switch operation {
	case "list":
		return t.listDocuments()
	case "read":
		docName, ok := params["document"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少文档名称参数")
		}
		return t.readDocument(docName)
	case "search":
		query, ok := params["query"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少搜索查询参数")
		}
		return t.searchDocuments(query)
	default:
		return nil, fmt.Errorf("不支持的操作类型: %s", operation)
	}
}

// listDocuments 列出所有文档
func (t *KnowledgeBaseTool) listDocuments() (interface{}, error) {
	// 确保知识库目录存在
	if err := t.ensureKnowledgeBaseExists(); err != nil {
		return nil, err
	}

	// 读取目录内容
	files, err := ioutil.ReadDir(t.basePath)
	if err != nil {
		return nil, fmt.Errorf("读取知识库目录失败: %w", err)
	}

	// 过滤出支持的文档类型（文本/Markdown/CSV/TSV）
	var documents []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(strings.ToLower(file.Name()), ".txt") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".md") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".csv") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".tsv")) {
			documents = append(documents, file.Name())
		}
	}

	if len(documents) == 0 {
		return "知识库中没有文档", nil
	}

	return documents, nil
}

// readDocument 读取指定文档
func (t *KnowledgeBaseTool) readDocument(docName string) (interface{}, error) {
	// 确保知识库目录存在
	if err := t.ensureKnowledgeBaseExists(); err != nil {
		return nil, err
	}

	// 构建文件路径
	filePath := filepath.Join(t.basePath, docName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文档不存在: %s", docName)
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文档失败: %w", err)
	}

	return string(content), nil
}

// searchDocuments 在文档中搜索内容
func (t *KnowledgeBaseTool) searchDocuments(query string) (interface{}, error) {
	// 确保知识库目录存在
	if err := t.ensureKnowledgeBaseExists(); err != nil {
		return nil, err
	}

	// 读取目录内容
	files, err := ioutil.ReadDir(t.basePath)
	if err != nil {
		return nil, fmt.Errorf("读取知识库目录失败: %w", err)
	}

	// 在每个文档中搜索
	results := make(map[string][]string)
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(strings.ToLower(file.Name()), ".txt") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".md") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".csv") ||
			strings.HasSuffix(strings.ToLower(file.Name()), ".tsv")) {
			filePath := filepath.Join(t.basePath, file.Name())
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue
			}

			// 简单的文本搜索；对CSV/TSV增加行号提示
			lines := strings.Split(string(content), "\n")
			var matches []string
			lowerQuery := strings.ToLower(query)
			isCSV := strings.HasSuffix(strings.ToLower(file.Name()), ".csv")
			isTSV := strings.HasSuffix(strings.ToLower(file.Name()), ".tsv")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), lowerQuery) {
					if isCSV || isTSV {
						// 为表格类文件标注行号，便于定位
						formatted := fmt.Sprintf("行 %d: %s", i+1, line)
						matches = append(matches, formatted)
					} else {
						matches = append(matches, line)
					}
				}
			}

			if len(matches) > 0 {
				results[file.Name()] = matches
			}
		}
	}

	if len(results) == 0 {
		return "没有找到匹配的内容", nil
	}

	return results, nil
}

// ensureKnowledgeBaseExists 确保知识库目录存在
func (t *KnowledgeBaseTool) ensureKnowledgeBaseExists() error {
	if _, err := os.Stat(t.basePath); os.IsNotExist(err) {
		// 创建知识库目录
		if err := os.MkdirAll(t.basePath, 0755); err != nil {
			return fmt.Errorf("创建知识库目录失败: %w", err)
		}

		// 创建一个示例文档
		examplePath := filepath.Join(t.basePath, "example.md")
		exampleContent := `# 示例知识库文档

这是一个示例文档，用于演示知识库功能。

## 使用方法

1. 将你的知识文档放在知识库目录中
2. 使用 knowledge_base 工具查询文档
3. 支持 .txt、.md、.csv、.tsv 格式的文档

## 示例查询

- 列出所有文档: {"operation": "list"}
- 读取文档: {"operation": "read", "document": "example.md"}
- 搜索内容: {"operation": "search", "query": "示例"}
`
		if err := ioutil.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
			return fmt.Errorf("创建示例文档失败: %w", err)
		}
	}

	return nil
}
