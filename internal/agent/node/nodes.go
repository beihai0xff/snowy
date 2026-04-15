// Package node 定义 Eino Graph 自定义节点。
// 每个节点对应技术方案 §10.7 中的一个 Graph Node。
package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/agent/assembler"
	agentrouter "github.com/beihai0xff/snowy/internal/agent/router"
	biologymodel "github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	physicsmodel "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

// Node 通用节点接口，所有自定义节点实现此接口。
type Node interface {
	Name() string
	Run(ctx context.Context, input any) (any, error)
}

// MessageRepository 抽象消息读取能力，供 SessionNode 使用。
type MessageRepository interface {
	ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error)
}

// InputPayload 是图执行入口载荷。
type InputPayload struct {
	Request *agent.ChatRequest
	Stream  bool
	Events  chan<- agent.SSEEvent
}

// State 是 Agent Graph 在节点之间传递的内部状态。
type State struct {
	Request            *agent.ChatRequest
	Stream             bool
	Events             chan<- agent.SSEEvent
	History            []*agent.Message
	ResolvedMode       agent.Mode
	TaskType           agentrouter.TaskType
	PrimaryModel       *agentrouter.ModelInfo
	FallbackModel      *agentrouter.ModelInfo
	ToolOutputs        map[string]any
	ToolCalls          []agent.ToolCall
	Response           *agent.ChatResponse
	ValidationWarnings []string
	FallbackReason     string
}

// InputNode 请求解析节点。
type InputNode struct{}

func (n *InputNode) Name() string { return "InputNode" }
func (n *InputNode) Run(_ context.Context, input any) (any, error) {
	switch value := input.(type) {
	case *InputPayload:
		if value == nil || value.Request == nil {
			return nil, fmt.Errorf("input payload is nil")
		}
		request := *value.Request
		if request.Mode == "" {
			request.Mode = agent.ModeAuto
		}
		return &State{Request: &request, Stream: value.Stream, Events: value.Events, ToolOutputs: make(map[string]any)}, nil
	case *agent.ChatRequest:
		if value == nil {
			return nil, fmt.Errorf("chat request is nil")
		}
		request := *value
		if request.Mode == "" {
			request.Mode = agent.ModeAuto
		}
		return &State{Request: &request, ToolOutputs: make(map[string]any)}, nil
	default:
		return nil, fmt.Errorf("unsupported input type %T", input)
	}
}

// SessionNode 会话上下文加载节点（基于 eino/memory）。
type SessionNode struct {
	messageRepo MessageRepository
}

// NewSessionNode 创建会话节点。
func NewSessionNode(messageRepo MessageRepository) *SessionNode {
	return &SessionNode{messageRepo: messageRepo}
}

func (n *SessionNode) Name() string { return "SessionNode" }
func (n *SessionNode) Run(ctx context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("session node expects *State, got %T", input)
	}
	if n.messageRepo == nil || state.Request == nil || state.Request.SessionID == uuid.Nil {
		return state, nil
	}
	messages, _, err := n.messageRepo.ListBySession(ctx, state.Request.SessionID, 0, 20)
	if err != nil {
		return nil, fmt.Errorf("load session messages: %w", err)
	}
	state.History = messages
	return state, nil
}

// IntentNode 意图分类节点（基于 ChatModel + Structured Output）。
type IntentNode struct {
	router agentrouter.Router
}

// NewIntentNode 创建意图节点。
func NewIntentNode(router agentrouter.Router) *IntentNode {
	return &IntentNode{router: router}
}

func (n *IntentNode) Name() string { return "IntentNode" }
func (n *IntentNode) Run(ctx context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("intent node expects *State, got %T", input)
	}
	mode := state.Request.Mode
	if mode == agent.ModeAuto {
		mode = inferMode(state.Request.Message, state.Request.Filters.Subject)
	}
	state.ResolvedMode = mode
	state.TaskType = resolveTaskType(mode)
	if n.router != nil {
		primary, err := n.router.Route(ctx, state.TaskType)
		if err == nil {
			state.PrimaryModel = primary
		}
		fallback, err := n.router.Fallback(ctx, state.TaskType)
		if err == nil {
			state.FallbackModel = fallback
		}
	}
	return state, nil
}

// ValidateNode Schema 校验节点。
type ValidateNode struct{}

func (n *ValidateNode) Name() string { return "ValidateNode" }
func (n *ValidateNode) Run(_ context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("validate node expects *State, got %T", input)
	}
	if state.Response == nil {
		return nil, fmt.Errorf("response is nil")
	}
	if strings.TrimSpace(state.Response.Answer) == "" && state.Response.StructuredPayload == nil {
		return nil, fmt.Errorf("response is empty")
	}
	switch state.ResolvedMode {
	case agent.ModePhysics:
		model, ok := state.ToolOutputs["physics"].(*physicsmodel.PhysicsModel)
		if !ok || model == nil || len(model.Steps) == 0 {
			return nil, fmt.Errorf("physics response is invalid")
		}
	case agent.ModeBiology:
		model, ok := state.ToolOutputs["biology"].(*biologymodel.BiologyModel)
		if !ok || model == nil || len(model.Concepts) == 0 {
			return nil, fmt.Errorf("biology response is invalid")
		}
	case agent.ModeSearch:
		if len(state.Response.Citations) == 0 {
			state.ValidationWarnings = append(state.ValidationWarnings, "检索结果缺少引用，已降级为摘要回答")
		}
	}
	return state, nil
}

// FallbackNode 备选模型重试节点。
type FallbackNode struct{}

func (n *FallbackNode) Name() string { return "FallbackNode" }
func (n *FallbackNode) Run(_ context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("fallback node expects *State, got %T", input)
	}
	answer := "当前结果未通过完整校验，已返回降级回答。"
	if state.Request != nil && strings.TrimSpace(state.Request.Message) != "" {
		answer = fmt.Sprintf("关于“%s”，当前返回的是降级结果，建议补充题干或稍后重试。", state.Request.Message)
	}
	state.FallbackReason = "validation_failed"
	state.Response = &agent.ChatResponse{
		Mode:        state.ResolvedMode,
		Answer:      answer,
		Confidence:  0.35,
		NextActions: []string{"补充更具体的题干条件", "切换到对应学科模式后再试"},
	}
	if state.FallbackModel != nil {
		state.Response.NextActions = append(state.Response.NextActions, fmt.Sprintf("已切换备选模型 %s/%s", state.FallbackModel.Provider, state.FallbackModel.Model))
	}
	return state, nil
}

// AssembleNode 结果组装节点。
type AssembleNode struct {
	assembler assembler.Assembler
}

// NewAssembleNode 创建结果组装节点。
func NewAssembleNode(assembler assembler.Assembler) *AssembleNode {
	return &AssembleNode{assembler: assembler}
}

func (n *AssembleNode) Name() string { return "AssembleNode" }
func (n *AssembleNode) Run(ctx context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("assemble node expects *State, got %T", input)
	}
	if n.assembler == nil {
		return nil, fmt.Errorf("assembler is nil")
	}
	response, err := n.assembler.Assemble(ctx, state.ResolvedMode, state.ToolOutputs, state.Response)
	if err != nil {
		return nil, err
	}
	response.ToolCalls = append([]agent.ToolCall(nil), state.ToolCalls...)
	if len(state.ValidationWarnings) > 0 {
		response.NextActions = append(response.NextActions, state.ValidationWarnings...)
	}
	state.Response = response
	return state, nil
}

// OutputNode SSE/JSON 输出节点。
type OutputNode struct{}

func (n *OutputNode) Name() string { return "OutputNode" }
func (n *OutputNode) Run(_ context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("output node expects *State, got %T", input)
	}
	if state.Response == nil {
		return nil, fmt.Errorf("response is nil")
	}
	if state.Stream && state.Events != nil {
		sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventThinking, Data: map[string]any{"content": "正在整理回答..."}})
		for _, call := range state.ToolCalls {
			sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventToolCall, Data: call})
		}
		if state.Response.Answer != "" {
			sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventContent, Data: map[string]any{"content": state.Response.Answer}})
		}
		for _, citation := range state.Response.Citations {
			sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventCitation, Data: citation})
		}
		switch state.ResolvedMode {
		case agent.ModePhysics:
			sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventChart, Data: state.Response.StructuredPayload})
		case agent.ModeBiology:
			sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventDiagram, Data: state.Response.StructuredPayload})
		}
		sendEvent(state.Events, agent.SSEEvent{Event: agent.SSEEventDone, Data: state.Response})
	}
	return state.Response, nil
}

func inferMode(message, subject string) agent.Mode {
	hint := strings.ToLower(strings.TrimSpace(message + " " + subject))
	switch {
	case strings.Contains(hint, "physics") || strings.Contains(hint, "物理") || strings.Contains(hint, "受力") || strings.Contains(hint, "速度"):
		return agent.ModePhysics
	case strings.Contains(hint, "biology") || strings.Contains(hint, "生物") || strings.Contains(hint, "细胞") || strings.Contains(hint, "光合作用"):
		return agent.ModeBiology
	default:
		return agent.ModeSearch
	}
}

func resolveTaskType(mode agent.Mode) agentrouter.TaskType {
	switch mode {
	case agent.ModePhysics:
		return agentrouter.TaskPhysicsDerivation
	case agent.ModeBiology:
		return agentrouter.TaskBiologyModeling
	default:
		return agentrouter.TaskSearchAnswer
	}
}

func sendEvent(events chan<- agent.SSEEvent, event agent.SSEEvent) {
	select {
	case events <- event:
	default:
		events <- event
	}
}
