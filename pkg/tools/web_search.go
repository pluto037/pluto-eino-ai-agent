package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SearchEngineType 表示搜索引擎类型
type SearchEngineType string

const (
	// SearchAPI 使用SearchAPI.com搜索
	SearchAPI SearchEngineType = "searchapi"
	// DuckDuckGo 使用DuckDuckGo搜索
	DuckDuckGo SearchEngineType = "duckduckgo"
	// Mock 使用模拟数据
	Mock SearchEngineType = "mock"
)

// WebSearchTool 实现了联网搜索功能
type WebSearchTool struct {
	engineType   SearchEngineType
	searchAPIURL string
	apiKey       string
}

// SearchResult 表示搜索结果
type SearchResult struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
}

// SearchResponse 表示搜索API的响应
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// NewWebSearchTool 创建一个新的网络搜索工具
func NewWebSearchTool(apiKey string) *WebSearchTool {
	// 如果没有提供API密钥，默认使用DuckDuckGo
	if apiKey == "" {
		return &WebSearchTool{
			engineType:   DuckDuckGo,
			searchAPIURL: "https://api.duckduckgo.com/",
			apiKey:       "",
		}
	}

	// 有API密钥则使用SearchAPI
	return &WebSearchTool{
		engineType:   SearchAPI,
		searchAPIURL: "https://api.searchapi.com/v1/search",
		apiKey:       apiKey,
	}
}

// NewWebSearchToolWithEngine 创建指定搜索引擎的网络搜索工具
func NewWebSearchToolWithEngine(engineType SearchEngineType, apiKey string) *WebSearchTool {
	switch engineType {
	case SearchAPI:
		return &WebSearchTool{
			engineType:   SearchAPI,
			searchAPIURL: "https://api.searchapi.com/v1/search",
			apiKey:       apiKey,
		}
	case DuckDuckGo:
		return &WebSearchTool{
			engineType:   DuckDuckGo,
			searchAPIURL: "https://api.duckduckgo.com/",
			apiKey:       "",
		}
	case Mock:
		return &WebSearchTool{
			engineType: Mock,
			apiKey:     "",
		}
	default:
		// 默认使用DuckDuckGo
		return &WebSearchTool{
			engineType:   DuckDuckGo,
			searchAPIURL: "https://api.duckduckgo.com/",
			apiKey:       "",
		}
	}
}

// Name 返回工具名称
func (t *WebSearchTool) Name() string {
	return "web_search"
}

// Description 返回工具描述
func (t *WebSearchTool) Description() string {
	return "搜索互联网获取信息"
}

// Execute 执行搜索
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("搜索查询不能为空")
	}

	switch t.engineType {
	case SearchAPI:
		return t.searchWithSearchAPI(ctx, query)
	case DuckDuckGo:
		return t.searchWithDuckDuckGo(ctx, query)
	case Mock:
		results := t.mockSearch(query)
		return t.formatResults(results), nil
	default:
		// 默认使用DuckDuckGo
		return t.searchWithDuckDuckGo(ctx, query)
	}
}

// searchWithSearchAPI 使用SearchAPI进行搜索
func (t *WebSearchTool) searchWithSearchAPI(ctx context.Context, query string) (interface{}, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s?q=%s&api_key=%s",
		t.searchAPIURL,
		url.QueryEscape(query),
		t.apiKey)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 如果没有结果，返回提示信息
	if len(searchResp.Results) == 0 {
		return "没有找到相关结果", nil
	}

	return t.formatResults(searchResp.Results), nil
}

// searchWithDuckDuckGo 使用DuckDuckGo进行搜索
func (t *WebSearchTool) searchWithDuckDuckGo(ctx context.Context, query string) (interface{}, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s?q=%s&format=json&no_html=1&no_redirect=1",
		t.searchAPIURL,
		url.QueryEscape(query))

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析DuckDuckGo响应
	var ddgResp struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 转换为统一的SearchResult格式
	var results []SearchResult

	// 添加摘要结果
	if ddgResp.AbstractText != "" && ddgResp.AbstractURL != "" {
		results = append(results, SearchResult{
			Title:       "摘要",
			Link:        ddgResp.AbstractURL,
			Description: ddgResp.AbstractText,
		})
	}

	// 添加相关主题
	for _, topic := range ddgResp.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			results = append(results, SearchResult{
				Title:       strings.Split(topic.Text, " - ")[0],
				Link:        topic.FirstURL,
				Description: topic.Text,
			})
		}
	}

	// 添加结果
	for _, result := range ddgResp.Results {
		if result.Text != "" && result.FirstURL != "" {
			results = append(results, SearchResult{
				Title:       strings.Split(result.Text, " - ")[0],
				Link:        result.FirstURL,
				Description: result.Text,
			})
		}
	}

	// 如果没有结果，返回提示信息
	if len(results) == 0 {
		return "没有找到相关结果", nil
	}

	return t.formatResults(results), nil
}

// formatResults 格式化搜索结果
func (t *WebSearchTool) formatResults(results []SearchResult) []map[string]string {
	formattedResults := make([]map[string]string, 0, len(results))
	for _, result := range results {
		formattedResults = append(formattedResults, map[string]string{
			"title":       result.Title,
			"link":        result.Link,
			"description": result.Description,
		})
	}
	return formattedResults
}

// 模拟搜索功能（当没有真实API密钥时使用）
func (t *WebSearchTool) mockSearch(query string) []SearchResult {
	// 这里只是一个模拟实现，实际应用中应该使用真实的搜索API
	return []SearchResult{
		{
			Title:       "搜索结果 1 - " + query,
			Link:        "https://example.com/result1",
			Description: "这是关于 " + query + " 的第一个搜索结果。",
		},
		{
			Title:       "搜索结果 2 - " + query,
			Link:        "https://example.com/result2",
			Description: "这是关于 " + query + " 的第二个搜索结果。",
		},
	}
}
