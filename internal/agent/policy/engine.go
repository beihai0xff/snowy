// Package policy 定义安全/学段/内容边界策略引擎。
// 参考技术方案 §10.5。
package policy

import "context"

// Result 策略检查结果。
type Result struct {
	Passed   bool     `json:"passed"`
	Warnings []string `json:"warnings,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

// Engine 策略引擎接口。
// 职责：控制学段边界、未成年人场景内容安全、限制超纲实验输出、
// 对低可信内容附加风险提示。
type Engine interface {
	// PreCheck 前置校验 — 在模型调用之前拦截不合规请求。
	PreCheck(ctx context.Context, input any) (*Result, error)
	// PostCheck 后置校验 — 在模型输出之后检查内容安全。
	PostCheck(ctx context.Context, output any) (*Result, error)
}
