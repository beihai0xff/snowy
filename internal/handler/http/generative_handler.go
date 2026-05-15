//revive:disable:var-naming
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/modeling/generative"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/user"
)

// GenerativeHandler exposes Snowy v4 generative modeling APIs.
type GenerativeHandler struct {
	generativeSvc generative.Service
	userSvc       user.Service
}

func NewGenerativeHandler(generativeSvc generative.Service, userSvc ...user.Service) *GenerativeHandler {
	var svc user.Service
	if len(userSvc) > 0 {
		svc = userSvc[0]
	}

	return &GenerativeHandler{generativeSvc: generativeSvc, userSvc: svc}
}

// Compile POST /api/v1/modeling/compile.
func (h *GenerativeHandler) Compile(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	var req dto.ModelingCompileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	userID := common.DefaultUserID
	if fromCtx := common.UserIDFromContext(c.Request.Context()); fromCtx != "" {
		userID = fromCtx
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("invalid user id"), reqID))

		return
	}

	compileReq := &generative.CompileRequest{
		UserID:     uid,
		Message:    req.Message,
		Domain:     req.Domain,
		GradeBand:  req.GradeBand,
		TargetMode: req.TargetMode,
		Context: generative.CompileContext{
			Citations:     convertEvidenceRefs(req.Context.Citations),
			KnowledgeTags: req.Context.KnowledgeTags,
			SourcePage:    req.Context.SourcePage,
			UserNotes:     req.Context.UserNotes,
		},
	}
	if req.SessionID != "" {
		if sid, parseErr := uuid.Parse(req.SessionID); parseErr == nil {
			compileReq.SessionID = sid
		} else {
			c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("invalid session_id"), reqID))

			return
		}
	}

	pkg, err := h.generativeSvc.Compile(c.Request.Context(), compileReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	recordHistory(c, h.userSvc, "modeling", req.Message)
	c.JSON(http.StatusOK, common.Success(pkg))
}

// ListPackages GET /api/v1/modeling/packages.
func (h *GenerativeHandler) ListPackages(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	userID := common.DefaultUserID
	if fromCtx := common.UserIDFromContext(c.Request.Context()); fromCtx != "" {
		userID = fromCtx
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("invalid user id"), reqID))

		return
	}

	items, total, err := h.generativeSvc.ListPackages(c.Request.Context(), uid, 0, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(common.PageResponse{
		Total:    total,
		Page:     1,
		PageSize: 20,
		Items:    items,
	}))
}

// GetPackage GET /api/v1/modeling/packages/:id.
func (h *GenerativeHandler) GetPackage(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	pkg, err := h.generativeSvc.GetPackage(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, common.Fail(common.ErrInvalidInput.WithMessage("model package not found"), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(pkg))
}

func convertEvidenceRefs(items []dto.EvidenceRefDTO) []generative.EvidenceRef {
	out := make([]generative.EvidenceRef, 0, len(items))
	for _, item := range items {
		out = append(out, generative.EvidenceRef{
			DocID:         item.DocID,
			SourceType:    item.SourceType,
			Title:         item.Title,
			Chapter:       item.Chapter,
			Snippet:       item.Snippet,
			KnowledgeTags: item.KnowledgeTags,
			Confidence:    item.Confidence,
		})
	}

	return out
}
