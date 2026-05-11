// Package router 定义模型路由策略。
// 参考技术方案 §14。
package router

import "context"

// TaskType 任务类型，影响模型路由策略。
type TaskType string

const (
	TaskSearchAnswer      TaskType = "search_answer"
	TaskPhysicsDerivation TaskType = "physics_derivation"
	TaskBiologyModeling   TaskType = "biology_modeling"
	TaskIntentClassify    TaskType = "intent_classify"
)

// ModelInfo 模型信息。
type ModelInfo struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Order    int    `json:"order"`
}

// Router 模型路由接口。
// 路由规则：按照 llm.models 的声明顺序选择当前任务入口；链路内部继续按声明顺序重试可用模型。
type Router interface {
	// Route 根据任务类型路由到合适的模型。
	Route(ctx context.Context, taskType TaskType) (*ModelInfo, error)
}
