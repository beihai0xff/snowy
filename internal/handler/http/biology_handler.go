package http

import (
	"net/http"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	biologysvc "github.com/beihai0xff/snowy/internal/modeling/biology/service"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/gin-gonic/gin"
)

// BiologyHandler 生物建模 HTTP Handler。
// 参考技术方案 §17.5。
type BiologyHandler struct {
	biologySvc biologysvc.BiologyService
}

// NewBiologyHandler 创建 BiologyHandler。
func NewBiologyHandler(biologySvc biologysvc.BiologyService) *BiologyHandler {
	return &BiologyHandler{biologySvc: biologySvc}
}

// Analyze POST /api/v1/modeling/biology/analyze — 生物建模解析。
func (h *BiologyHandler) Analyze(c *gin.Context) {
	var req dto.BiologyAnalyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	result, err := h.biologySvc.Analyze(c.Request.Context(), req.Question, req.Context)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}
