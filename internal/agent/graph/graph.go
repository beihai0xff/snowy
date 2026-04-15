// Package graph 定义 Eino Graph 编排拓扑。
// 参考技术方案 §10.7。
package graph

import "context"

// Builder 构建 Agent 编排图。
// 基于 Eino Graph 编排：InputNode → SessionNode → PrePolicyNode → IntentNode
// → [SearchToolNode / PhysicsToolNode / BioToolNode] → ValidateNode
// → AssembleNode → PostPolicyNode → OutputNode。
type Builder struct {
	// 依赖注入通过 Option 模式传入
}

// Option 配置选项。
type Option func(*Builder)

// NewBuilder 创建 Graph Builder。
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Build 构建并返回可运行的 Agent 编排图。
// TODO: 集成 Eino compose.Graph 实现完整编排拓扑。
func (b *Builder) Build(_ context.Context) error {
	// 1. 创建 Eino Graph
	// 2. 注册节点：Input, Session, PrePolicy, Intent
	// 3. 注册分支：search / physics / biology
	// 4. 注册节点：Validate, Fallback, Assemble, PostPolicy, Output
	// 5. 添加条件路由边
	// 6. 编译图
	return nil
}
