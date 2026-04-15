package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicssvc "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// PhysicsHandler 物理建模 HTTP Handler。
// 参考技术方案 §17.3 & §17.4。
type PhysicsHandler struct {
	physicsSvc physicssvc.PhysicsService
}

// NewPhysicsHandler 创建 PhysicsHandler。
func NewPhysicsHandler(physicsSvc physicssvc.PhysicsService) *PhysicsHandler {
	return &PhysicsHandler{physicsSvc: physicsSvc}
}

// Analyze POST /api/v1/modeling/physics/analyze — 物理解析。
func (h *PhysicsHandler) Analyze(c *gin.Context) {
	var req dto.PhysicsAnalyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))
		return
	}

	result, err := h.physicsSvc.Analyze(c.Request.Context(), req.Question, req.Context)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))
		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}

// Simulate POST /api/v1/modeling/physics/simulate — 物理调参。
func (h *PhysicsHandler) Simulate(c *gin.Context) {
	var req dto.PhysicsSimulateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))
		return
	}

	result, err := h.physicsSvc.Simulate(c.Request.Context(), domain.ModelType(req.ModelType), req.Parameters)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))
		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}
