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
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	IsPrimary bool   `json:"is_primary"`
}

// Router 模型路由接口。
// 路由规则：默认走主模型(gpt5)，失败/超时/校验失败/预算超限 时切换备选(gemini3)。
type Router interface {
	// Route 根据任务类型路由到合适的模型。
	Route(ctx context.Context, taskType TaskType) (*ModelInfo, error)
	// Fallback 获取备选模型。
	Fallback(ctx context.Context, taskType TaskType) (*ModelInfo, error)
}
