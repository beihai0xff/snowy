//revive:disable:var-naming
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/modeling/generative"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/share"
)

const (
	shareResponseTokenKey     = "token"
	shareResponsePackageIDKey = "package_id"
	shareResponseExpiresAtKey = "expires_at"
)

// ShareHandler v7 §6.2 静态分享。
type ShareHandler struct {
	shareRepo     share.Repository
	generativeSvc generative.Service
}

// NewShareHandler 创建分享 Handler。
func NewShareHandler(shareRepo share.Repository, generativeSvc generative.Service) *ShareHandler {
	return &ShareHandler{shareRepo: shareRepo, generativeSvc: generativeSvc}
}

// CreatePackageShare POST /api/v1/share/packages — 为指定 package 生成只读分享链接。
func (h *ShareHandler) CreatePackageShare(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	var req dto.PackageShareCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	if h.generativeSvc == nil || h.shareRepo == nil {
		c.JSON(http.StatusServiceUnavailable, common.Fail(common.ErrInternal.WithMessage("share unavailable"), reqID))

		return
	}

	pkg, err := h.generativeSvc.GetPackage(c.Request.Context(), req.PackageID)
	if err != nil {
		c.JSON(http.StatusNotFound, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	snapshot, err := json.Marshal(pkg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	token, err := newShareToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	userID := common.DefaultUserID
	if fromCtx := common.UserIDFromContext(c.Request.Context()); fromCtx != "" {
		userID = fromCtx
	}

	uid, _ := uuid.Parse(userID)

	var expires *time.Time

	if req.ExpiresInHours > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expires = &t
	}

	entry := &share.Package{
		Token:        token,
		PackageID:    pkg.PackageID,
		SnapshotJSON: snapshot,
		CreatedBy:    uid,
		ExpiresAt:    expires,
		Mode:         defaultMode(req.Mode),
		CreatedAt:    time.Now(),
	}
	if err := h.shareRepo.Create(c.Request.Context(), entry); err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(gin.H{
		shareResponseTokenKey:     token,
		shareResponsePackageIDKey: pkg.PackageID,
		"mode":                    entry.Mode,
		shareResponseExpiresAtKey: expires,
	}))
}

// GetPackageShare GET /api/v1/share/:token — 公开访问。
func (h *ShareHandler) GetPackageShare(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	token := c.Param(shareResponseTokenKey)
	if token == "" {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("missing token"), reqID))

		return
	}

	if h.shareRepo == nil {
		c.JSON(http.StatusServiceUnavailable, common.Fail(common.ErrInternal.WithMessage("share unavailable"), reqID))

		return
	}

	entry, err := h.shareRepo.GetByToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, share.ErrNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(common.ErrInvalidInput.WithMessage("share not found"), reqID))

			return
		}

		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	var snapshot any
	if err := json.Unmarshal(entry.SnapshotJSON, &snapshot); err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(gin.H{
		shareResponseTokenKey:     entry.Token,
		shareResponsePackageIDKey: entry.PackageID,
		"mode":                    entry.Mode,
		"created_at":              entry.CreatedAt,
		shareResponseExpiresAtKey: entry.ExpiresAt,
		"snapshot":                snapshot,
		"can_collab":              entry.Mode == "collab",
	}))
}

// JoinPackageShare POST /api/v1/share/:token/join — 申请加入协同房间，返回 ws_url + role。
// v7 §6.3：仅在 share.mode=collab 时返回可加入的 WS 地址。
func (h *ShareHandler) JoinPackageShare(c *gin.Context) {
	reqID := common.RequestIDFromContext(c.Request.Context())

	token := c.Param(shareResponseTokenKey)
	if token == "" {
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("missing token"), reqID))

		return
	}

	if h.shareRepo == nil {
		c.JSON(http.StatusServiceUnavailable, common.Fail(common.ErrInternal.WithMessage("share unavailable"), reqID))

		return
	}

	entry, err := h.shareRepo.GetByToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, share.ErrNotFound) {
			c.JSON(http.StatusNotFound, common.Fail(common.ErrInvalidInput.WithMessage("share not found"), reqID))

			return
		}

		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	if entry.Mode != "collab" {
		c.JSON(
			http.StatusForbidden,
			common.Fail(common.ErrInvalidInput.WithMessage("share is not collaborative"), reqID),
		)

		return
	}

	userID := common.UserIDFromContext(c.Request.Context())

	role := "guest"
	if userID != "" && userID == entry.CreatedBy.String() {
		role = "host"
	}

	wsURL := "/api/v1/ws/session/" + entry.PackageID.String() + "?token=" + entry.Token

	c.JSON(http.StatusOK, common.Success(gin.H{
		shareResponseTokenKey:     entry.Token,
		shareResponsePackageIDKey: entry.PackageID,
		"role":                    role,
		"ws_url":                  wsURL,
		shareResponseExpiresAtKey: entry.ExpiresAt,
	}))
}

func newShareToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func defaultMode(m string) string {
	if m == "" {
		return "view"
	}

	return m
}
