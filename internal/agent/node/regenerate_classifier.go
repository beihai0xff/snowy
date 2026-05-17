package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/repo/llm"
)

// RegenerateClassifierNode 追问意图分类节点。
type RegenerateClassifierNode struct {
	llm llm.Provider
}

// NewRegenerateClassifierNode 创建追问意图分类节点。llm 可为 nil（仅启用关键词规则）。
func NewRegenerateClassifierNode(provider llm.Provider) *RegenerateClassifierNode {
	return &RegenerateClassifierNode{llm: provider}
}

// Name 节点名。
func (n *RegenerateClassifierNode) Name() string { return "RegenerateClassifierNode" }

// classifierOutput LLM 结构化输出。
type classifierOutput struct {
	Action     string             `json:"action"`
	TargetVars map[string]float64 `json:"target_vars,omitempty"`
	Reason     string             `json:"reason,omitempty"`
}

// Run 执行分类。仅当 ParentPackageID 命中时才生效。
func (n *RegenerateClassifierNode) Run(ctx context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("regenerate classifier expects *State, got %T", input)
	}

	if state.Request == nil || state.Request.ParentPackageID == nil {
		state.RegenerateAction = RegenerateActionNew

		return state, nil
	}

	action, vars, reason := n.classifyWithLLM(ctx, state.Request.Message)
	if action == "" {
		action, reason = classifyByKeyword(state.Request.Message)
	}

	state.RegenerateAction = action
	state.TargetVars = vars

	if reason != "" {
		state.RegenerateReason = reason
	} else if state.Request.RegenerateReason != "" {
		state.RegenerateReason = state.Request.RegenerateReason
	}

	return state, nil
}

const classifierSystemPrompt = `你是教育对话改写分类器。用户已经看到一个交互式演示，正在追问。请判断意图：
- recompute：仅希望修改演示的变量值（数值、初速度、角度等），返回 target_vars 映射。
- regenerate：希望换一个例子、换一种解法、换场景等结构性变化。
- new：与当前演示无关的新主题。
只输出 JSON，形如 {"action":"recompute","target_vars":{"v0":30},"reason":"..."}。`

func (n *RegenerateClassifierNode) classifyWithLLM(
	ctx context.Context,
	message string,
) (RegenerateAction, map[string]float64, string) {
	if n.llm == nil || strings.TrimSpace(message) == "" {
		return "", nil, ""
	}

	req := &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: classifierSystemPrompt},
			{Role: "user", Content: message},
		},
		Temperature: 0,
	}

	resp, err := n.llm.Generate(ctx, req)
	if err != nil || resp == nil {
		return "", nil, ""
	}

	out, err := parseClassifierOutput(resp.Content)
	if err != nil {
		return "", nil, ""
	}

	switch RegenerateAction(out.Action) {
	case RegenerateActionRecompute, RegenerateActionRegenerate, RegenerateActionNew:
		return RegenerateAction(out.Action), out.TargetVars, out.Reason
	}

	return "", nil, ""
}

func parseClassifierOutput(content string) (*classifierOutput, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("empty classifier output")
	}

	start := strings.Index(content, "{")

	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return nil, errors.New("classifier output is not json")
	}

	var out classifierOutput
	if err := json.Unmarshal([]byte(content[start:end+1]), &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// classifyByKeyword 关键词兜底规则。命中"换/重新/再来一个/不同"等关键词 → regenerate；命中数字或参数词 → recompute；其余 new。
func classifyByKeyword(message string) (RegenerateAction, string) {
	lower := strings.ToLower(message)

	regenerateKeywords := []string{"换一个", "换个", "重新", "再来一个", "不同", "另一个", "其他例子", "再举", "换成"}
	for _, kw := range regenerateKeywords {
		if strings.Contains(message, kw) || strings.Contains(lower, kw) {
			return RegenerateActionRegenerate, kw
		}
	}

	recomputeKeywords := []string{"如果", "假设", "改成", "变成", "调成", "设为", "if "}
	for _, kw := range recomputeKeywords {
		if strings.Contains(message, kw) || strings.Contains(lower, kw) {
			return RegenerateActionRecompute, kw
		}
	}

	if containsDigit(message) {
		return RegenerateActionRecompute, "numeric_param"
	}

	return RegenerateActionNew, ""
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}

	return false
}
