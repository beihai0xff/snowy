// Package http is the transport-layer package name for this adapter set.
//
//revive:disable:var-naming
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	chemistrysvc "github.com/beihai0xff/snowy/internal/modeling/chemistry/service"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/user"
)

// ChemistryHandler 化学建模 HTTP Handler。
// 参考 v7 §5。
type ChemistryHandler struct {
	chemSvc chemistrysvc.Service
	userSvc user.Service
}

// NewChemistryHandler 创建 ChemistryHandler。
func NewChemistryHandler(chemSvc chemistrysvc.Service, userSvc ...user.Service) *ChemistryHandler {
	var svc user.Service
	if len(userSvc) > 0 {
		svc = userSvc[0]
	}

	return &ChemistryHandler{chemSvc: chemSvc, userSvc: svc}
}

// Analyze POST /api/v1/modeling/chemistry/analyze — 化学反应解析。
func (h *ChemistryHandler) Analyze(c *gin.Context) {
	var req dto.ChemistryAnalyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	result, err := h.chemSvc.AnalyzeReaction(c.Request.Context(), req.Equation)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	recordHistory(c, h.userSvc, "chemistry", req.Equation)
	c.JSON(http.StatusOK, common.Success(result))
}

// Balance POST /api/v1/modeling/chemistry/balance — 化学方程式配平。
func (h *ChemistryHandler) Balance(c *gin.Context) {
	var req dto.ChemistryBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	result, err := h.chemSvc.Balance(c.Request.Context(), req.Equation)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(result))
}
