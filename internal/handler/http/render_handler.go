//revive:disable:var-naming
package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicssvc "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// RenderHandler 浏览器渲染代码生成 HTTP Handler。
type RenderHandler struct {
	physicsSvc physicssvc.PhysicsService
}

// NewRenderHandler 创建 RenderHandler。
func NewRenderHandler(physicsSvc physicssvc.PhysicsService) *RenderHandler {
	return &RenderHandler{physicsSvc: physicsSvc}
}

// Generate POST /api/v1/modeling/render/generate — 生成浏览器可渲染的前端代码。
func (h *RenderHandler) Generate(c *gin.Context) {
	var req dto.RenderGenerateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	artifact, err := h.physicsSvc.GenerateRender(c.Request.Context(), &domain.SceneSpec{
		SceneType:    req.SceneSpec.SceneType,
		Title:        req.SceneSpec.Title,
		Summary:      req.SceneSpec.Summary,
		RenderMode:   domain.RenderMode(strings.TrimSpace(req.SceneSpec.RenderMode)),
		DefaultProps: req.SceneSpec.DefaultProps,
	}, req.RenderMode)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadGateway, common.Fail(common.ErrRenderValidationFailed.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(artifact))
}
