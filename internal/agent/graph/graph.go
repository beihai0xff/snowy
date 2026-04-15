// Package graph 定义 Eino Graph 编排拓扑。
// 参考技术方案 §10.7。
package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/agent/assembler"
	"github.com/beihai0xff/snowy/internal/agent/callback"
	nodepkg "github.com/beihai0xff/snowy/internal/agent/node"
	"github.com/beihai0xff/snowy/internal/agent/policy"
	agentrouter "github.com/beihai0xff/snowy/internal/agent/router"
	"github.com/beihai0xff/snowy/internal/agent/tool"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
)

// Builder 构建 Agent 编排图。
// 基于 Eino Graph 编排：InputNode → SessionNode → PrePolicyNode → IntentNode
// → [SearchToolNode / PhysicsToolNode / BioToolNode] → ValidateNode
// → AssembleNode → PostPolicyNode → OutputNode。
type Builder struct {
	router             agentrouter.Router
	policy             policy.Engine
	assembler          assembler.Assembler
	messageRepo        nodepkg.MessageRepository
	searchTool         *tool.SearchTool
	physicsAnalyzeTool *tool.PhysicsAnalyzeTool
	biologyAnalyzeTool *tool.BiologyAnalyzeTool
	citationTool       *tool.CitationTool
	callbacks          []callback.NodeCallback
	primaryLLM         llm.Provider
	fallbackLLM        llm.Provider

	buildOnce sync.Once
	buildErr  error
}

// Option 配置选项。
type Option func(*Builder)

func WithRouter(router agentrouter.Router) Option  { return func(b *Builder) { b.router = router } }
func WithPolicyEngine(engine policy.Engine) Option { return func(b *Builder) { b.policy = engine } }
func WithAssembler(assembler assembler.Assembler) Option {
	return func(b *Builder) { b.assembler = assembler }
}
func WithMessageRepository(repo nodepkg.MessageRepository) Option {
	return func(b *Builder) { b.messageRepo = repo }
}
func WithSearchTool(searchTool *tool.SearchTool) Option {
	return func(b *Builder) { b.searchTool = searchTool }
}
func WithPhysicsAnalyzeTool(physicsTool *tool.PhysicsAnalyzeTool) Option {
	return func(b *Builder) { b.physicsAnalyzeTool = physicsTool }
}
func WithBiologyAnalyzeTool(biologyTool *tool.BiologyAnalyzeTool) Option {
	return func(b *Builder) { b.biologyAnalyzeTool = biologyTool }
}
func WithCitationTool(citationTool *tool.CitationTool) Option {
	return func(b *Builder) { b.citationTool = citationTool }
}
func WithCallbacks(callbacks ...callback.NodeCallback) Option {
	return func(b *Builder) { b.callbacks = append(b.callbacks, callbacks...) }
}
func WithLLMProviders(primary, fallback llm.Provider) Option {
	return func(b *Builder) {
		b.primaryLLM = primary
		b.fallbackLLM = fallback
	}
}

// NewBuilder 创建 Graph Builder。
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Build 构建并返回可运行的 Agent 编排图。
func (b *Builder) Build(_ context.Context) error {
	b.buildOnce.Do(func() {
		if b.assembler == nil {
			b.buildErr = fmt.Errorf("assembler is nil")
		}
	})
	return b.buildErr
}

// Chat 执行同步编排。
func (b *Builder) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	if err := b.Build(ctx); err != nil {
		return nil, err
	}
	ctx = ensureRequestID(ctx)
	return b.run(ctx, &nodepkg.InputPayload{Request: req})
}

// ChatStream 执行流式编排。
func (b *Builder) ChatStream(ctx context.Context, req *agent.ChatRequest, events chan<- agent.SSEEvent) error {
	if err := b.Build(ctx); err != nil {
		return err
	}
	ctx = ensureRequestID(ctx)
	_, err := b.run(ctx, &nodepkg.InputPayload{Request: req, Stream: true, Events: events})
	return err
}

// Execute 执行同步编排。
func (b *Builder) Execute(ctx context.Context, input *nodepkg.InputPayload) (*agent.ChatResponse, error) {
	if err := b.Build(ctx); err != nil {
		return nil, err
	}
	return b.run(ctx, input)
}

// ExecuteStream 执行流式编排。
func (b *Builder) ExecuteStream(ctx context.Context, input *nodepkg.InputPayload) error {
	if err := b.Build(ctx); err != nil {
		return err
	}
	_, err := b.run(ctx, input)
	return err
}

func (b *Builder) run(ctx context.Context, input *nodepkg.InputPayload) (*agent.ChatResponse, error) {
	var err error
	var current any = input
	for _, node := range []nodepkg.Node{
		&nodepkg.InputNode{},
		nodepkg.NewSessionNode(b.messageRepo),
		nodepkg.NewIntentNode(b.router),
	} {
		current, err = b.runNode(ctx, node, current)
		if err != nil {
			return nil, err
		}
	}

	state, ok := current.(*nodepkg.State)
	if !ok {
		return nil, fmt.Errorf("unexpected graph state %T", current)
	}
	if err := b.preCheck(ctx, state); err != nil {
		return nil, err
	}
	if err := b.executeTool(ctx, state); err != nil {
		state.FallbackReason = err.Error()
		state, err = b.runFallback(ctx, state)
		if err != nil {
			return nil, err
		}
	} else {
		current, err = b.runNode(ctx, nodepkg.NewAssembleNode(b.assembler), state)
		if err != nil {
			state.FallbackReason = err.Error()
			state, err = b.runFallback(ctx, state)
			if err != nil {
				return nil, err
			}
		} else {
			state = current.(*nodepkg.State)
			current, err = b.runNode(ctx, &nodepkg.ValidateNode{}, state)
			if err != nil {
				state.FallbackReason = err.Error()
				state, err = b.runFallback(ctx, state)
				if err != nil {
					return nil, err
				}
			} else {
				state = current.(*nodepkg.State)
			}
		}
	}
	if err := b.postCheck(ctx, state); err != nil {
		state.FallbackReason = err.Error()
		state, err = b.runFallback(ctx, state)
		if err != nil {
			return nil, err
		}
	}
	current, err = b.runNode(ctx, &nodepkg.OutputNode{}, state)
	if err != nil {
		return nil, err
	}
	response, ok := current.(*agent.ChatResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected graph output %T", current)
	}
	return response, nil
}

func (b *Builder) runNode(ctx context.Context, node nodepkg.Node, input any) (output any, err error) {
	for _, cb := range b.callbacks {
		cb.OnNodeStart(ctx, node.Name(), input)
	}
	defer func() {
		for _, cb := range b.callbacks {
			cb.OnNodeEnd(ctx, node.Name(), output, err)
		}
	}()
	return node.Run(ctx, input)
}

func (b *Builder) executeTool(ctx context.Context, state *nodepkg.State) error {
	switch state.ResolvedMode {
	case agent.ModePhysics:
		if b.physicsAnalyzeTool == nil {
			return fmt.Errorf("physics tool is nil")
		}
		state.ToolCalls = append(state.ToolCalls, agent.ToolCall{Tool: b.physicsAnalyzeTool.Name(), Status: "running"})
		output, err := b.physicsAnalyzeTool.Run(ctx, tool.PhysicsAnalyzeInput{Question: state.Request.Message, SessionContext: joinHistory(state.History)})
		if err != nil {
			state.ToolCalls[len(state.ToolCalls)-1].Status = "failed"
			return err
		}
		state.ToolCalls[len(state.ToolCalls)-1].Status = "success"
		state.ToolOutputs["physics"] = output
	case agent.ModeBiology:
		if b.biologyAnalyzeTool == nil {
			return fmt.Errorf("biology tool is nil")
		}
		state.ToolCalls = append(state.ToolCalls, agent.ToolCall{Tool: b.biologyAnalyzeTool.Name(), Status: "running"})
		output, err := b.biologyAnalyzeTool.Run(ctx, tool.BiologyAnalyzeInput{Question: state.Request.Message, SessionContext: joinHistory(state.History)})
		if err != nil {
			state.ToolCalls[len(state.ToolCalls)-1].Status = "failed"
			return err
		}
		state.ToolCalls[len(state.ToolCalls)-1].Status = "success"
		state.ToolOutputs["biology"] = output
	default:
		if b.searchTool == nil {
			return fmt.Errorf("search tool is nil")
		}
		state.ToolCalls = append(state.ToolCalls, agent.ToolCall{Tool: b.searchTool.Name(), Status: "running"})
		output, err := b.searchTool.Run(ctx, tool.SearchInput{Query: state.Request.Message, Filters: searchdomain.Filters{Subject: state.Request.Filters.Subject, Grade: state.Request.Filters.Grade}})
		if err != nil {
			state.ToolCalls[len(state.ToolCalls)-1].Status = "failed"
			return err
		}
		state.ToolCalls[len(state.ToolCalls)-1].Status = "success"
		state.ToolOutputs["search"] = output
		if b.citationTool != nil {
			citations, err := b.citationTool.Run(ctx, output)
			if err == nil {
				state.ToolOutputs["citations"] = citations
			}
		}
	}
	return nil
}

func (b *Builder) runFallback(ctx context.Context, state *nodepkg.State) (*nodepkg.State, error) {
	current, err := b.runNode(ctx, &nodepkg.FallbackNode{}, state)
	if err != nil {
		return nil, err
	}
	fallbackState, ok := current.(*nodepkg.State)
	if !ok {
		return nil, fmt.Errorf("unexpected fallback state %T", current)
	}
	current, err = b.runNode(ctx, nodepkg.NewAssembleNode(b.assembler), fallbackState)
	if err != nil {
		return nil, err
	}
	assembledState, ok := current.(*nodepkg.State)
	if !ok {
		return nil, fmt.Errorf("unexpected assembled state %T", current)
	}
	return assembledState, nil
}

func (b *Builder) preCheck(ctx context.Context, state *nodepkg.State) error {
	if b.policy == nil {
		return nil
	}
	result, err := b.policy.PreCheck(ctx, state.Request.Message)
	if err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("pre policy check failed: %s", result.Reason)
	}
	return nil
}

func (b *Builder) postCheck(ctx context.Context, state *nodepkg.State) error {
	if b.policy == nil {
		return nil
	}
	result, err := b.policy.PostCheck(ctx, state.Response)
	if err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("post policy check failed: %s", result.Reason)
	}
	return nil
}

func joinHistory(messages []*agent.Message) string {
	if len(messages) == 0 {
		return ""
	}
	result := ""
	for _, message := range messages {
		if message == nil || message.Content == "" {
			continue
		}
		if result != "" {
			result += "\n"
		}
		result += message.Content
	}
	return result
}

func ensureRequestID(ctx context.Context) context.Context {
	if common.RequestIDFromContext(ctx) != "" {
		return ctx
	}
	return common.WithRequestID(ctx, uuid.NewString())
}
