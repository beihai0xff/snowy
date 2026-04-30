package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamResponseAggregator_Consume(t *testing.T) {
	agg := NewStreamResponseAggregator(ModeAuto)

	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventContent, Data: "F="}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventContent, Data: map[string]any{"content": "ma"}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventCitation, Data: Citation{
		DocID:      "doc-1",
		SourceType: "textbook",
		Snippet:    "第二定律",
		Score:      0.9,
	}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventToolCall, Data: map[string]any{"tool": "SearchTool", "status": "running"}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventToolCall, Data: map[string]any{"tool": "SearchTool", "status": "success"}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventDone, Data: map[string]any{"mode": "search", "confidence": 0.93}}))

	resp := agg.Response()
	require.NotNil(t, resp)
	assert.True(t, agg.Done())
	assert.Equal(t, ModeSearch, resp.Mode)
	assert.Equal(t, "F=ma", resp.Answer)
	require.Len(t, resp.Citations, 1)
	assert.Equal(t, "doc-1", resp.Citations[0].DocID)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "success", resp.ToolCalls[0].Status)
	assert.Equal(t, 0.93, resp.Confidence)
}

func TestStreamResponseAggregator_DoneOverridesResponse(t *testing.T) {
	agg := NewStreamResponseAggregator(ModeAuto)

	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventContent, Data: "partial"}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventDone, Data: ChatResponse{
		Mode:       ModePhysics,
		Answer:     "final answer",
		Confidence: 0.88,
		ToolCalls:  []ToolCall{{Tool: "Calculator", Status: "success"}},
	}}))

	resp := agg.Response()
	require.NotNil(t, resp)
	assert.Equal(t, ModePhysics, resp.Mode)
	assert.Equal(t, "final answer", resp.Answer)
	assert.Equal(t, 0.88, resp.Confidence)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "Calculator", resp.ToolCalls[0].Tool)
}

func TestStreamResponseAggregator_IgnoresHeartbeat(t *testing.T) {
	agg := NewStreamResponseAggregator(ModePhysics)

	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventToolCall, Data: map[string]any{"tool": "RenderCodeTool", "status": "running"}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventHeartbeat, Data: map[string]any{"tool": "RenderCodeTool", "status": "running"}}))
	require.NoError(t, agg.Consume(SSEEvent{Event: SSEEventContent, Data: "生成中"}))

	resp := agg.Response()
	require.NotNil(t, resp)
	assert.Equal(t, "生成中", resp.Answer)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "RenderCodeTool", resp.ToolCalls[0].Tool)
	assert.Equal(t, "running", resp.ToolCalls[0].Status)
}
