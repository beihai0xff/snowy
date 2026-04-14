// Package node 定义 Eino Graph 自定义节点。
// 每个节点对应技术方案 §10.7 中的一个 Graph Node。
package node

import "context"

// Node 通用节点接口，所有自定义节点实现此接口。
type Node interface {
	Name() string
	Run(ctx context.Context, input any) (any, error)
}

// InputNode 请求解析节点。
type InputNode struct{}

func (n *InputNode) Name() string { return "InputNode" }
func (n *InputNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 解析原始请求为内部结构
	return input, nil
}

// SessionNode 会话上下文加载节点（基于 eino/memory）。
type SessionNode struct{}

func (n *SessionNode) Name() string { return "SessionNode" }
func (n *SessionNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 从 Session Manager 加载上下文
	return input, nil
}

// IntentNode 意图分类节点（基于 ChatModel + Structured Output）。
type IntentNode struct{}

func (n *IntentNode) Name() string { return "IntentNode" }
func (n *IntentNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 调用 ChatModel 进行意图识别
	return input, nil
}

// ValidateNode Schema 校验节点。
type ValidateNode struct{}

func (n *ValidateNode) Name() string { return "ValidateNode" }
func (n *ValidateNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 校验模型输出的结构化数据
	return input, nil
}

// FallbackNode 备选模型重试节点。
type FallbackNode struct{}

func (n *FallbackNode) Name() string { return "FallbackNode" }
func (n *FallbackNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 切换备选模型重试
	return input, nil
}

// AssembleNode 结果组装节点。
type AssembleNode struct{}

func (n *AssembleNode) Name() string { return "AssembleNode" }
func (n *AssembleNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 组装最终响应
	return input, nil
}

// OutputNode SSE/JSON 输出节点。
type OutputNode struct{}

func (n *OutputNode) Name() string { return "OutputNode" }
func (n *OutputNode) Run(ctx context.Context, input any) (any, error) {
	// TODO: 桥接到 Gin SSE/JSON 输出
	return input, nil
}
