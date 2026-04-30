package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

// SearchHandler 知识检索 HTTP Handler。
// 参考技术方案 §17.2。
type SearchHandler struct {
	searchSvc search.Service
	userSvc   user.Service
}

// NewSearchHandler 创建 SearchHandler。
func NewSearchHandler(searchSvc search.Service, userSvc ...user.Service) *SearchHandler {
	var svc user.Service
	if len(userSvc) > 0 {
		svc = userSvc[0]
	}

	return &SearchHandler{searchSvc: searchSvc, userSvc: svc}
}

// Query POST /api/v1/search/query — 执行知识检索。
func (h *SearchHandler) Query(c *gin.Context) {
	var req dto.SearchQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	query := &search.Query{
		Text: req.Query,
		Filters: search.Filters{
			Subject: req.Filters.Subject,
			Grade:   req.Filters.Grade,
			Chapter: req.Filters.Chapter,
			Source:  req.Filters.Source,
		},
	}

	resp, err := h.searchSvc.Query(c.Request.Context(), query)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	recordHistory(c, h.userSvc, "search", req.Query)
	c.JSON(http.StatusOK, common.Success(resp))
}
