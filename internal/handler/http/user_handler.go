package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/user"
)

// UserHandler 用户 HTTP Handler。
// 参考技术方案 §17.7 & §18A。
type UserHandler struct {
	userSvc user.Service
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(userSvc user.Service) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// Login POST /api/v1/auth/login — 手机号+验证码登录。
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	access, refresh, err := h.userSvc.Login(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized.WithMessage("登录失败"), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(dto.LoginResp{
		AccessToken:  access,
		RefreshToken: refresh,
	}))
}

// Register POST /api/v1/auth/register — 注册。
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	u, err := h.userSvc.Register(c.Request.Context(), req.Phone, req.Nickname)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(u))
}

// GetProfile GET /api/v1/user/profile — 获取当前用户资料。
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := common.UserIDFromContext(c.Request.Context())
	if userID == "" {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized, reqID))

		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput, reqID))

		return
	}

	profile, err := h.userSvc.GetProfile(c.Request.Context(), uid)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(profile))
}

// GetHistory GET /api/v1/history — 历史记录。
func (h *UserHandler) GetHistory(c *gin.Context) {
	userID := common.UserIDFromContext(c.Request.Context())
	uid, _ := uuid.Parse(userID)

	items, total, err := h.userSvc.GetHistory(c.Request.Context(), uid, 0, 20)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(common.PageResponse{
		Total:    total,
		Page:     1,
		PageSize: 20,
		Items:    items,
	}))
}

// AddFavorite POST /api/v1/favorites — 添加收藏。
func (h *UserHandler) AddFavorite(c *gin.Context) {
	var req dto.FavoriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	userID := common.UserIDFromContext(c.Request.Context())
	uid, _ := uuid.Parse(userID)

	fav := &user.Favorite{
		UserID:     uid,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Title:      req.Title,
	}

	if err := h.userSvc.AddFavorite(c.Request.Context(), fav); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(fav))
}
