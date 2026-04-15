package http

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// AgentHandler Agent 会话 HTTP Handler。
// 参考技术方案 §17.1 & §17.6。
type AgentHandler struct {
	agentSvc agent.Service
}

// NewAgentHandler 创建 AgentHandler。
func NewAgentHandler(agentSvc agent.Service) *AgentHandler {
	return &AgentHandler{agentSvc: agentSvc}
}

// Chat POST /api/v1/agent/chat — 统一会话入口（支持 SSE）。
func (h *AgentHandler) Chat(c *gin.Context) {
	var req dto.ChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))
		return
	}

	// 判断是否请求 SSE 流式
	if c.GetHeader("Accept") == "text/event-stream" {
		h.chatStream(c, &req)
		return
	}

	chatReq := &agent.ChatRequest{
		Message: req.Message,
		Mode:    agent.Mode(req.Mode),
		Filters: agent.Filters{
			Subject: req.Filters.Subject,
			Grade:   req.Filters.Grade,
		},
	}

	resp, err := h.agentSvc.Chat(c.Request.Context(), chatReq)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))
		return
	}

	c.JSON(http.StatusOK, common.Success(resp))
}

// chatStream 流式输出 SSE。
func (h *AgentHandler) chatStream(c *gin.Context, req *dto.ChatReq) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	events := make(chan agent.SSEEvent, 32)

	chatReq := &agent.ChatRequest{
		Message: req.Message,
		Mode:    agent.Mode(req.Mode),
		Filters: agent.Filters{
			Subject: req.Filters.Subject,
			Grade:   req.Filters.Grade,
		},
	}

	go func() {
		defer close(events)
		_ = h.agentSvc.ChatStream(c.Request.Context(), chatReq, events)
	}()

	c.Stream(func(w io.Writer) bool {
		event, ok := <-events
		if !ok {
			return false
		}
		c.SSEvent(string(event.Event), event.Data)
		return true
	})
}

// CreateSession POST /api/v1/agent/sessions — 创建会话。
func (h *AgentHandler) CreateSession(c *gin.Context) {
	// TODO: 创建 Agent Session
	c.JSON(http.StatusOK, common.Success(nil))
}

// GetSession GET /api/v1/agent/sessions/:id — 获取会话详情。
func (h *AgentHandler) GetSession(c *gin.Context) {
	// TODO: 获取 Agent Session
	c.JSON(http.StatusOK, common.Success(nil))
}

// ListMessages GET /api/v1/agent/sessions/:id/messages — 获取会话消息列表。
func (h *AgentHandler) ListMessages(c *gin.Context) {
	// TODO: 列出 Agent Session 消息
	c.JSON(http.StatusOK, common.Success(nil))
}
