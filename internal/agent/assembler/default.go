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
		if response, ok := toolOutputs["search"].(*searchdomain.Response); ok && response != nil {
			citations := make([]agent.Citation, 0, len(response.Citations))
			for _, citation := range response.Citations {
				citations = append(citations, agent.Citation{DocID: citation.DocID, SourceType: citation.SourceType, Snippet: citation.Snippet, Score: citation.Score})
			}
			return &agent.ChatResponse{Mode: mode, Answer: response.Answer, Citations: citations, StructuredPayload: response, Confidence: response.Confidence}, nil
		}
	case agent.ModePhysics:
		if response, ok := toolOutputs["physics"].(*physicsmodel.PhysicsModel); ok && response != nil {
			return &agent.ChatResponse{Mode: mode, Answer: response.ResultSummary, StructuredPayload: response, Confidence: 0.82}, nil
		}
	case agent.ModeBiology:
		if response, ok := toolOutputs["biology"].(*biologymodel.BiologyModel); ok && response != nil {
			return &agent.ChatResponse{Mode: mode, Answer: response.ResultSummary, StructuredPayload: response, Confidence: 0.8}, nil
		}
	}

	return nil, fmt.Errorf("no tool output available for mode %s", mode)
}
