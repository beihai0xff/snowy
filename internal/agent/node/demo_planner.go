// Package node 中 demo_planner 节点。
// v7 §3：在 runPrimaryFlow 物理/生物/化学分支后执行，
// 根据 regenerate_classifier 结果（或新会话）调用 generative.Service
// 产出 GenerativeModelPackage，并发射 preview SSE 事件。
package node

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/modeling/generative"
)

// DemoPlannerNode 内嵌演示编排节点。
type DemoPlannerNode struct {
	svc generative.Service
}

// NewDemoPlannerNode 创建 demo_planner 节点。svc 为 nil 时节点变为 no-op。
func NewDemoPlannerNode(svc generative.Service) *DemoPlannerNode {
	return &DemoPlannerNode{svc: svc}
}

// Name 节点名。
func (n *DemoPlannerNode) Name() string { return "DemoPlannerNode" }

// Run 执行 demo 编排。
func (n *DemoPlannerNode) Run(ctx context.Context, input any) (any, error) {
	state, ok := input.(*State)
	if !ok {
		return nil, fmt.Errorf("demo planner expects *State, got %T", input)
	}

	if n.svc == nil || !demoApplicable(state) {
		return state, nil
	}

	stage, pkg, err := n.dispatch(ctx, state)
	if err != nil {
		emitPreviewFailed(state, stage, err)

		return state, nil
	}

	if pkg == nil {
		return state, nil
	}

	state.Package = pkg
	pid := pkg.PackageID
	state.PackageID = &pid

	emitPreviewComplete(state, stage, pkg)

	return state, nil
}

func demoApplicable(state *State) bool {
	switch state.ResolvedMode {
	case agent.ModePhysics, agent.ModeBiology, agent.ModeChemistry:
		return true
	}

	if state.Request != nil && state.Request.InteractiveDemo != nil {
		return true
	}

	return false
}

func (n *DemoPlannerNode) dispatch(
	ctx context.Context,
	state *State,
) (string, *generative.GenerativeModelPackage, error) {
	parentID := ""
	if state.Request != nil && state.Request.ParentPackageID != nil {
		parentID = state.Request.ParentPackageID.String()
	}

	switch state.RegenerateAction {
	case RegenerateActionRecompute:
		if parentID == "" {
			break
		}

		emitPreviewPartial(state, "recompute")

		pkg, err := n.svc.Recompute(ctx, parentID, state.TargetVars)

		return "recompute", pkg, err

	case RegenerateActionRegenerate:
		if parentID == "" {
			break
		}

		emitPreviewPartial(state, "regenerate")

		pkg, err := n.svc.Regenerate(ctx, parentID, state.RegenerateReason, buildRegenerateContext(state))

		return "regenerate", pkg, err
	}

	emitPreviewPartial(state, "compile")

	pkg, err := n.svc.Compile(ctx, buildCompileRequest(state))

	return "compile", pkg, err
}

func buildCompileRequest(state *State) *generative.CompileRequest {
	req := &generative.CompileRequest{
		Message:   state.Request.Message,
		Domain:    resolveDomain(state),
		GradeBand: state.Request.Filters.Grade,
	}

	if state.Request.InteractiveDemo != nil {
		if state.Request.InteractiveDemo.TargetMode != "" {
			req.TargetMode = state.Request.InteractiveDemo.TargetMode
		}

		if req.Domain == "" && state.Request.InteractiveDemo.Domain != "" {
			req.Domain = state.Request.InteractiveDemo.Domain
		}
	}

	req.Context = buildRegenerateContext(state)

	return req
}

func buildRegenerateContext(state *State) generative.CompileContext {
	hint := generative.CompileContext{}
	if state.Request != nil {
		if state.Request.RegenerateReason != "" {
			hint.UserNotes = state.Request.RegenerateReason
		} else if state.RegenerateReason != "" {
			hint.UserNotes = state.RegenerateReason
		}
	}

	for _, citation := range state.Response.Citations {
		hint.Citations = append(hint.Citations, generative.EvidenceRef{
			DocID:      citation.DocID,
			SourceType: citation.SourceType,
			Snippet:    citation.Snippet,
			Confidence: citation.Score,
		})
	}

	return hint
}

func resolveDomain(state *State) string {
	switch state.ResolvedMode {
	case agent.ModePhysics:
		return "physics"
	case agent.ModeBiology:
		return "biology"
	case agent.ModeChemistry:
		return "chemistry"
	}

	if state.Request != nil && state.Request.InteractiveDemo != nil && state.Request.InteractiveDemo.Domain != "" {
		return state.Request.InteractiveDemo.Domain
	}

	if state.Request != nil && state.Request.Filters.Subject != "" {
		return strings.ToLower(state.Request.Filters.Subject)
	}

	return "auto"
}

func emitPreviewPartial(state *State, stage string) {
	if !state.Stream || state.Events == nil {
		return
	}

	sendEvent(state.Events, agent.SSEEvent{
		Event: agent.SSEEventPreview,
		Data: agent.PreviewPayload{
			Status: "partial",
			Stage:  stage,
		},
	})
}

func emitPreviewComplete(state *State, stage string, pkg *generative.GenerativeModelPackage) {
	if state.Response != nil {
		state.Response.StructuredPayload = pkg
	}

	if !state.Stream || state.Events == nil {
		return
	}

	sendEvent(state.Events, agent.SSEEvent{
		Event: agent.SSEEventPreview,
		Data: agent.PreviewPayload{
			PackageID: pkg.PackageID,
			Status:    "complete",
			Stage:     stage,
			Package:   pkg,
		},
	})
}

func emitPreviewFailed(state *State, stage string, err error) {
	if !state.Stream || state.Events == nil {
		return
	}

	if err == nil {
		err = errors.New("unknown error")
	}

	sendEvent(state.Events, agent.SSEEvent{
		Event: agent.SSEEventPreview,
		Data: agent.PreviewPayload{
			Status: "failed",
			Stage:  stage,
			Error:  err.Error(),
		},
	})
}
