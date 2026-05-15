// Package tool 定义 Agent 业务工具，通过 Eino Tool Registry 注册。
// 参考技术方案 §10.3。
package tool

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	biologysvc "github.com/beihai0xff/snowy/internal/modeling/biology/service"
	physicsdomain "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicssvc "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

// Tool Agent 工具统一接口。
// 每个工具定义输入/输出 schema、超时配置、重试策略、审计策略。
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input any) (any, error)
}

type SearchInput struct {
	Query   string
	Filters searchdomain.Filters
}

type PhysicsAnalyzeInput struct {
	Question       string
	SessionContext string
}

type RenderCodeInput struct {
	SceneSpec  *physicsdomain.SceneSpec
	RenderMode string
}

type BiologyAnalyzeInput struct {
	Question       string
	SessionContext string
}

type HistoryInput struct {
	UserID uuid.UUID
	Offset int
	Limit  int
}

// SearchTool 知识检索工具。
type SearchTool struct {
	searchService searchdomain.Service
}

// NewSearchTool 创建知识检索工具。
func NewSearchTool(searchService searchdomain.Service) *SearchTool {
	return &SearchTool{searchService: searchService}
}

func (t *SearchTool) Name() string { return "SearchTool" }
func (t *SearchTool) Description() string {
	return "直接调用大模型回答知识点问题，返回兼容 SearchResponse 的结构化结果"
}

func (t *SearchTool) Run(ctx context.Context, input any) (any, error) {
	request, ok := input.(SearchInput)
	if !ok {
		return nil, fmt.Errorf("%s: invalid input type %T", t.Name(), input)
	}

	if t.searchService == nil {
		return nil, fmt.Errorf("%s: search service is nil", t.Name())
	}

	return t.searchService.Query(ctx, &searchdomain.Query{Text: request.Query, Filters: request.Filters})
}

// PhysicsAnalyzeTool 物理分析工具。
type PhysicsAnalyzeTool struct {
	physicsService physicssvc.PhysicsService
}

// NewPhysicsAnalyzeTool 创建物理分析工具。
func NewPhysicsAnalyzeTool(physicsService physicssvc.PhysicsService) *PhysicsAnalyzeTool {
	return &PhysicsAnalyzeTool{physicsService: physicsService}
}

func (t *PhysicsAnalyzeTool) Name() string { return "PhysicsAnalyzeTool" }
func (t *PhysicsAnalyzeTool) Description() string {
	return "抽取物理条件、识别场景并生成 scene_spec"
}

func (t *PhysicsAnalyzeTool) Run(ctx context.Context, input any) (any, error) {
	request, ok := input.(PhysicsAnalyzeInput)
	if !ok {
		return nil, fmt.Errorf("%s: invalid input type %T", t.Name(), input)
	}

	if t.physicsService == nil {
		return nil, fmt.Errorf("%s: physics service is nil", t.Name())
	}

	return t.physicsService.Analyze(ctx, request.Question, request.SessionContext)
}

// RenderCodeTool 渲染预览产物生成工具。
type RenderCodeTool struct {
	physicsService physicssvc.PhysicsService
}

// NewRenderCodeTool 创建渲染预览产物生成工具。
func NewRenderCodeTool(physicsService physicssvc.PhysicsService) *RenderCodeTool {
	return &RenderCodeTool{physicsService: physicsService}
}

func (t *RenderCodeTool) Name() string { return "RenderCodeTool" }
func (t *RenderCodeTool) Description() string {
	return "根据 scene_spec 生成浏览器可渲染的预览产物；物理场景返回 Rapier 原生引擎配置"
}

func (t *RenderCodeTool) Run(ctx context.Context, input any) (any, error) {
	request, ok := input.(RenderCodeInput)
	if !ok {
		return nil, fmt.Errorf("%s: invalid input type %T", t.Name(), input)
	}

	if t.physicsService == nil {
		return nil, fmt.Errorf("%s: physics service is nil", t.Name())
	}

	if request.SceneSpec == nil {
		return nil, fmt.Errorf("%s: scene spec is nil", t.Name())
	}

	return t.physicsService.GenerateRender(ctx, request.SceneSpec, request.RenderMode)
}

// BiologyAnalyzeTool 生物分析工具。
type BiologyAnalyzeTool struct {
	biologyService biologysvc.BiologyService
}

// NewBiologyAnalyzeTool 创建生物分析工具。
func NewBiologyAnalyzeTool(biologyService biologysvc.BiologyService) *BiologyAnalyzeTool {
	return &BiologyAnalyzeTool{biologyService: biologyService}
}

func (t *BiologyAnalyzeTool) Name() string { return "BiologyAnalyzeTool" }
func (t *BiologyAnalyzeTool) Description() string {
	return "识别生物主题、概念并抽取关系"
}

func (t *BiologyAnalyzeTool) Run(ctx context.Context, input any) (any, error) {
	request, ok := input.(BiologyAnalyzeInput)
	if !ok {
		return nil, fmt.Errorf("%s: invalid input type %T", t.Name(), input)
	}

	if t.biologyService == nil {
		return nil, fmt.Errorf("%s: biology service is nil", t.Name())
	}

	return t.biologyService.Analyze(ctx, request.Question, request.SessionContext)
}

// CitationTool 引用拼装工具。
type CitationTool struct{}

// NewCitationTool 创建引用工具。
func NewCitationTool() *CitationTool { return &CitationTool{} }

func (t *CitationTool) Name() string        { return "CitationTool" }
func (t *CitationTool) Description() string { return "拼装引用片段和来源信息" }
func (t *CitationTool) Run(_ context.Context, input any) (any, error) {
	switch value := input.(type) {
	case *searchdomain.Response:
		citations := make([]agent.Citation, 0, len(value.Citations))
		for _, citation := range value.Citations {
			citations = append(
				citations,
				agent.Citation{
					DocID:      citation.DocID,
					SourceType: citation.SourceType,
					Snippet:    citation.Snippet,
					Score:      citation.Score,
				},
			)
		}

		return citations, nil
	case []searchdomain.Citation:
		citations := make([]agent.Citation, 0, len(value))
		for _, citation := range value {
			citations = append(
				citations,
				agent.Citation{
					DocID:      citation.DocID,
					SourceType: citation.SourceType,
					Snippet:    citation.Snippet,
					Score:      citation.Score,
				},
			)
		}

		return citations, nil
	default:
		return nil, fmt.Errorf("%s: unsupported input type %T", t.Name(), input)
	}
}

// HistoryTool 历史查询工具。
type HistoryTool struct {
	userService user.Service
}

// NewHistoryTool 创建历史工具。
func NewHistoryTool(userService user.Service) *HistoryTool {
	return &HistoryTool{userService: userService}
}

func (t *HistoryTool) Name() string        { return "HistoryTool" }
func (t *HistoryTool) Description() string { return "查询用户历史记录" }
func (t *HistoryTool) Run(ctx context.Context, input any) (any, error) {
	request, ok := input.(HistoryInput)
	if !ok {
		return nil, fmt.Errorf("%s: invalid input type %T", t.Name(), input)
	}

	if t.userService == nil {
		return nil, fmt.Errorf("%s: user service is nil", t.Name())
	}

	items, total, err := t.userService.GetHistory(ctx, request.UserID, request.Offset, request.Limit)
	if err != nil {
		return nil, err
	}

	return map[string]any{"items": items, "total": total}, nil
}
