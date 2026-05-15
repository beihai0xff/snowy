package assembler

import (
	"context"
	"fmt"

	"github.com/beihai0xff/snowy/internal/agent"
	biologymodel "github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	physicsmodel "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
)

type defaultAssembler struct{}

// NewDefaultAssembler 创建默认响应组装器。
func NewDefaultAssembler() Assembler {
	return &defaultAssembler{}
}

func (a *defaultAssembler) Assemble(
	_ context.Context,
	mode agent.Mode,
	toolOutputs map[string]any,
	modelOutput any,
) (*agent.ChatResponse, error) {
	if response, ok := modelOutput.(*agent.ChatResponse); ok && response != nil {
		return response, nil
	}

	switch mode {
	case agent.ModeSearch:
		return assembleSearchResponse(mode, toolOutputs)
	case agent.ModePhysics:
		return assemblePhysicsResponse(mode, toolOutputs)
	case agent.ModeBiology:
		return assembleBiologyResponse(mode, toolOutputs)
	case agent.ModeAuto:
		if response, err := assembleSearchResponse(agent.ModeSearch, toolOutputs); err == nil {
			return response, nil
		}

		if response, err := assemblePhysicsResponse(agent.ModePhysics, toolOutputs); err == nil {
			return response, nil
		}

		if response, err := assembleBiologyResponse(agent.ModeBiology, toolOutputs); err == nil {
			return response, nil
		}
	}

	return nil, fmt.Errorf("no tool output available for mode %s", mode)
}

func assembleSearchResponse(mode agent.Mode, toolOutputs map[string]any) (*agent.ChatResponse, error) {
	response, ok := toolOutputs["search"].(*searchdomain.Response)
	if !ok || response == nil {
		return nil, fmt.Errorf("no tool output available for mode %s", mode)
	}

	citations := make([]agent.Citation, 0, len(response.Citations))
	for _, citation := range response.Citations {
		citations = append(citations, agent.Citation{
			DocID:      citation.DocID,
			SourceType: citation.SourceType,
			Snippet:    citation.Snippet,
			Score:      citation.Score,
		})
	}

	return &agent.ChatResponse{
		Mode:              mode,
		Answer:            response.Answer,
		Citations:         citations,
		StructuredPayload: response,
		Confidence:        response.Confidence,
	}, nil
}

func assemblePhysicsResponse(mode agent.Mode, toolOutputs map[string]any) (*agent.ChatResponse, error) {
	model, ok := toolOutputs["physics"].(*physicsmodel.PhysicsModel)
	if !ok || model == nil {
		return nil, fmt.Errorf("no tool output available for mode %s", mode)
	}

	artifact, _ := toolOutputs["render"].(*physicsmodel.RenderArtifact)

	payload := map[string]any{
		"analysis": model,
	}
	if artifact != nil {
		payload["render_artifact"] = artifact
	}

	answer := model.ResultSummary
	if artifact != nil && artifact.ResultSummary != "" {
		answer = artifact.ResultSummary
	}

	return &agent.ChatResponse{
		Mode:              mode,
		Answer:            answer,
		StructuredPayload: payload,
		Confidence:        0.88,
		NextActions:       []string{"可在前端直接挂载 render_artifact，并通过参数面板驱动 Rapier 3D 原生物理引擎"},
	}, nil
}

func assembleBiologyResponse(mode agent.Mode, toolOutputs map[string]any) (*agent.ChatResponse, error) {
	response, ok := toolOutputs["biology"].(*biologymodel.BiologyModel)
	if !ok || response == nil {
		return nil, fmt.Errorf("no tool output available for mode %s", mode)
	}

	artifact, _ := toolOutputs["render"].(*physicsmodel.RenderArtifact)

	payload := map[string]any{
		"analysis": response,
	}
	if artifact != nil {
		payload["render_artifact"] = artifact
	}

	answer := response.ResultSummary
	if artifact != nil && artifact.ResultSummary != "" {
		answer = artifact.ResultSummary
	}

	return &agent.ChatResponse{
		Mode:              mode,
		Answer:            answer,
		StructuredPayload: payload,
		Confidence:        0.84,
		NextActions:       []string{"可查看生物动态演示页，也可继续查看概念图谱和实验变量分析"},
	}, nil
}
