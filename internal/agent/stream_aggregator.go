package agent

import (
	"encoding/json"

	"github.com/google/uuid"
)

// StreamResponseAggregator 将 SSE 事件流聚合为最终 ChatResponse。
type StreamResponseAggregator struct {
	response      ChatResponse
	events        []SSEEvent
	seenCitations map[string]struct{}
	toolCallIndex map[string]int
	previewStatus string
	packageID     *uuid.UUID
	done          bool
}

// NewStreamResponseAggregator 创建流式响应聚合器。
func NewStreamResponseAggregator(initialMode Mode) *StreamResponseAggregator {
	return &StreamResponseAggregator{
		response:      ChatResponse{Mode: initialMode},
		seenCitations: make(map[string]struct{}),
		toolCallIndex: make(map[string]int),
	}
}

// Consume 消费一个 SSE 事件并更新聚合结果。
func (a *StreamResponseAggregator) Consume(event SSEEvent) error {
	a.events = append(a.events, event)

	if event.Event == SSEEventDone {
		a.done = true
		a.consumeDone(event.Data)

		return nil
	}

	a.consumeIncrementalEvent(event)

	return nil
}

// Done 返回是否已收到终态事件。
func (a *StreamResponseAggregator) Done() bool {
	return a.done
}

// Events 返回已收到的全部事件副本（v7 §3：用于 agent_message_events 落库与 replay）。
func (a *StreamResponseAggregator) Events() []SSEEvent {
	if len(a.events) == 0 {
		return nil
	}

	out := make([]SSEEvent, len(a.events))
	copy(out, a.events)

	return out
}

// Response 返回聚合后的最终响应副本。
func (a *StreamResponseAggregator) Response() *ChatResponse {
	resp := a.response
	if len(a.response.Citations) > 0 {
		resp.Citations = append([]Citation(nil), a.response.Citations...)
	}

	if len(a.response.ToolCalls) > 0 {
		resp.ToolCalls = append([]ToolCall(nil), a.response.ToolCalls...)
	}

	if len(a.response.NextActions) > 0 {
		resp.NextActions = append([]string(nil), a.response.NextActions...)
	}

	return &resp
}

func (a *StreamResponseAggregator) consumeIncrementalEvent(event SSEEvent) {
	switch event.Event {
	case SSEEventThinking, SSEEventHeartbeat, SSEEventDone:
	case SSEEventContent:
		content, ok := stringValue(event.Data, "content", "text", "answer")
		if ok {
			a.response.Answer += content
		}
	case SSEEventCitation:
		citation, ok := decodeCitation(event.Data)
		if ok {
			a.appendCitation(citation)
		}
	case SSEEventToolCall:
		toolCall, ok := decodeToolCall(event.Data)
		if ok {
			a.upsertToolCall(toolCall)
		}
	case SSEEventDiagram, SSEEventRenderCode:
		if a.response.StructuredPayload == nil {
			a.response.StructuredPayload = event.Data
		}
	case SSEEventPreview:
		// v7 §3 锁定：preview 事件覆盖到最新 complete，保留最后一个有效快照。
		status, _ := stringValue(event.Data, "status")
		if a.previewStatus == "complete" && status != "complete" {
			// 已经有 complete 快照，忽略后续 partial。
			return
		}

		a.response.StructuredPayload = event.Data
		if status != "" {
			a.previewStatus = status
		}

		if pid, ok := stringValue(event.Data, "package_id"); ok && pid != "" {
			if parsed, err := uuid.Parse(pid); err == nil {
				a.packageID = &parsed
			}
		}
	}
}

// PackageID v7 §3：返回 preview 事件中携带的 package_id（若有）。
func (a *StreamResponseAggregator) PackageID() *uuid.UUID {
	if a.packageID == nil {
		return nil
	}

	id := *a.packageID

	return &id
}

func (a *StreamResponseAggregator) consumeDone(data any) {
	if data == nil {
		return
	}

	var done ChatResponse
	if err := decodeViaJSON(data, &done); err == nil {
		a.applyDoneResponse(done)

		return
	}

	a.applyDoneFields(data)
}

func (a *StreamResponseAggregator) appendCitation(citation Citation) {
	key := citation.DocID + "|" + citation.SourceType + "|" + citation.Snippet
	if _, ok := a.seenCitations[key]; ok {
		return
	}

	a.seenCitations[key] = struct{}{}
	a.response.Citations = append(a.response.Citations, citation)
}

func (a *StreamResponseAggregator) upsertToolCall(toolCall ToolCall) {
	if index, ok := a.toolCallIndex[toolCall.Tool]; ok {
		a.response.ToolCalls[index] = toolCall

		return
	}

	a.toolCallIndex[toolCall.Tool] = len(a.response.ToolCalls)
	a.response.ToolCalls = append(a.response.ToolCalls, toolCall)
}

func decodeCitation(data any) (Citation, bool) {
	var citation Citation
	if err := decodeViaJSON(data, &citation); err != nil {
		return Citation{}, false
	}

	return citation, citation.DocID != "" || citation.Snippet != ""
}

func decodeToolCall(data any) (ToolCall, bool) {
	var toolCall ToolCall
	if err := decodeViaJSON(data, &toolCall); err != nil {
		return ToolCall{}, false
	}

	return toolCall, toolCall.Tool != ""
}

func decodeViaJSON(data any, dest any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, dest)
}

func stringValue(data any, keys ...string) (string, bool) {
	if value, ok := data.(string); ok {
		return value, true
	}

	m, ok := data.(map[string]any)
	if !ok {
		return "", false
	}

	for _, key := range keys {
		if value, ok := m[key].(string); ok {
			return value, true
		}
	}

	return "", false
}

func floatValue(data any, key string) (float64, bool) {
	m, ok := data.(map[string]any)
	if !ok {
		return 0, false
	}

	switch value := m[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}

func modeValue(data any, key string) (Mode, bool) {
	value, ok := stringValue(data, key)
	if !ok || value == "" {
		return "", false
	}

	return Mode(value), true
}

func (a *StreamResponseAggregator) applyDoneResponse(done ChatResponse) {
	if done.Mode != "" {
		a.response.Mode = done.Mode
	}

	if done.Answer != "" {
		a.response.Answer = done.Answer
	}

	if done.Confidence != 0 {
		a.response.Confidence = done.Confidence
	}

	for _, citation := range done.Citations {
		a.appendCitation(citation)
	}

	for _, call := range done.ToolCalls {
		a.upsertToolCall(call)
	}

	if done.StructuredPayload != nil {
		a.response.StructuredPayload = done.StructuredPayload
	}

	if len(done.NextActions) > 0 {
		a.response.NextActions = append([]string(nil), done.NextActions...)
	}
}

func (a *StreamResponseAggregator) applyDoneFields(data any) {
	if mode, ok := modeValue(data, "mode"); ok {
		a.response.Mode = mode
	}

	if answer, ok := stringValue(data, "answer", "content"); ok {
		a.response.Answer = answer
	}

	if confidence, ok := floatValue(data, "confidence"); ok {
		a.response.Confidence = confidence
	}
}
