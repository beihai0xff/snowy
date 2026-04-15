// Package tool 定义 Agent 业务工具，通过 Eino Tool Registry 注册。
// 参考技术方案 §10.3。
package tool

import (
	"context"
	"fmt"
)

// Tool Agent 工具统一接口。
// 每个工具定义输入/输出 schema、超时配置、重试策略、审计策略。
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input any) (any, error)
}

// SearchTool 知识检索工具。
type SearchTool struct {
	// searchService search.Service — 注入检索域服务
}

func (t *SearchTool) Name() string        { return "SearchTool" }
func (t *SearchTool) Description() string { return "执行知识检索，返回多路召回结果" }
func (t *SearchTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 调用 search.Service.Query
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}

// PhysicsAnalyzeTool 物理分析工具。
type PhysicsAnalyzeTool struct{}

func (t *PhysicsAnalyzeTool) Name() string        { return "PhysicsAnalyzeTool" }
func (t *PhysicsAnalyzeTool) Description() string { return "抽取物理条件并识别物理模型" }
func (t *PhysicsAnalyzeTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 调用 physics.Service.Analyze
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}

// PhysicsSimulateTool 物理模拟工具。
type PhysicsSimulateTool struct{}

func (t *PhysicsSimulateTool) Name() string        { return "PhysicsSimulateTool" }
func (t *PhysicsSimulateTool) Description() string { return "执行物理数值计算并生成图表" }
func (t *PhysicsSimulateTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 调用 physics.Service.Simulate
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}

// BiologyAnalyzeTool 生物分析工具。
type BiologyAnalyzeTool struct{}

func (t *BiologyAnalyzeTool) Name() string { return "BiologyAnalyzeTool" }
func (t *BiologyAnalyzeTool) Description() string {
	return "识别生物主题、概念并抽取关系"
}

func (t *BiologyAnalyzeTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 调用 biology.Service.Analyze
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}

// CitationTool 引用拼装工具。
type CitationTool struct{}

func (t *CitationTool) Name() string        { return "CitationTool" }
func (t *CitationTool) Description() string { return "拼装引用片段和来源信息" }
func (t *CitationTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 组装引用
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}

// HistoryTool 历史查询工具。
type HistoryTool struct{}

func (t *HistoryTool) Name() string        { return "HistoryTool" }
func (t *HistoryTool) Description() string { return "查询用户历史记录" }
func (t *HistoryTool) Run(_ context.Context, _ any) (any, error) {
	// TODO: 调用 user.Service.GetHistory
	return nil, fmt.Errorf("%s: not implemented", t.Name())
}
