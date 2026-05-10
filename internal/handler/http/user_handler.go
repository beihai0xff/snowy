package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/user"
)

// UserHandler 用户 HTTP Handler。
// 参考技术方案 §17.7 & §18A。
// v5 支持邮箱登录；未携带 token 时仍回落默认匿名用户。
type UserHandler struct {
	userSvc user.Service
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(userSvc user.Service) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// Register POST /api/v1/auth/register — 邮箱注册。
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.EmailRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	access, refresh, profile, err := h.userSvc.EmailRegister(c.Request.Context(), req.Email, req.Password, req.Nickname)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(dto.AuthResp{AccessToken: access, RefreshToken: refresh, User: profile}))
}

// Login POST /api/v1/auth/login — 邮箱登录。
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.EmailLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	access, refresh, profile, err := h.userSvc.EmailLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(dto.AuthResp{AccessToken: access, RefreshToken: refresh, User: profile}))
}

// GetProfile GET /api/v1/user/profile — 获取当前用户资料。
func (h *UserHandler) GetProfile(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
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

func (h *UserHandler) ensureDefaultUser(ctx *gin.Context, uid uuid.UUID) bool {
	if h.userSvc == nil {
		return true
	}

	_, err := h.userSvc.GetProfile(ctx.Request.Context(), uid)
	if err == nil {
		return true
	}

	if uid.String() != common.DefaultUserID {
		reqID := common.RequestIDFromContext(ctx.Request.Context())
		ctx.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return false
	}

	usr, ok := h.userSvc.(interface {
		EnsureAnonymousUser(context.Context) (*user.User, error)
	})
	if !ok {
		reqID := common.RequestIDFromContext(ctx.Request.Context())
		ctx.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return false
	}

	_, ensureErr := usr.EnsureAnonymousUser(ctx.Request.Context())
	if ensureErr != nil {
		reqID := common.RequestIDFromContext(ctx.Request.Context())
		ctx.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(ensureErr.Error()), reqID))

		return false
	}

	return true
}

// resolveUserID 从 context 中获取 userID 并解析为 uuid.UUID，
// 若 context 中无 userID 则使用默认匿名用户。
func (h *UserHandler) resolveUserID(c *gin.Context) (uuid.UUID, bool) {
	userID := common.UserIDFromContext(c.Request.Context())
	if userID == "" {
		userID = common.DefaultUserID
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput, reqID))

		return uuid.Nil, false
	}

	if !h.ensureDefaultUser(c, uid) {
		return uuid.Nil, false
	}

	return uid, true
}

// GetHistory GET /api/v1/history — 历史记录。
func (h *UserHandler) GetHistory(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

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

// ListFavorites GET /api/v1/favorites — 收藏列表。
func (h *UserHandler) ListFavorites(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	items, total, err := h.userSvc.ListFavorites(c.Request.Context(), uid, 0, 20)
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

// GetRecommendations GET /api/v1/recommendations — 首页推荐数据。
func (h *UserHandler) GetRecommendations(c *gin.Context) {
	resp := dto.RecommendationsResp{
		HotTopics: []dto.RecommendationItem{
			{ID: "ht-1", Title: "牛顿第二定律", Description: "力与加速度的关系 F=ma", Category: "physics", Icon: "🔬"},
			{ID: "ht-2", Title: "光合作用", Description: "植物如何将光能转化为化学能", Category: "biology", Icon: "🌱"},
			{ID: "ht-3", Title: "匀变速直线运动", Description: "速度随时间均匀变化的运动", Category: "physics", Icon: "📐"},
		},
		PhysicsModels: []dto.RecommendationItem{
			{ID: "pm-1", Title: "匀速直线运动", Description: "速度恒定的运动模型", Category: "physics"},
			{ID: "pm-2", Title: "匀变速直线运动", Description: "加速度恒定的运动模型", Category: "physics"},
			{ID: "pm-3", Title: "平抛运动", Description: "水平抛出的运动模型", Category: "physics"},
			{ID: "pm-4", Title: "牛顿第二定律", Description: "力与运动的关系", Category: "physics"},
			{ID: "pm-5", Title: "功和能", Description: "做功与能量转换", Category: "physics"},
		},
		BiologyTopics: []dto.RecommendationItem{
			{ID: "bt-1", Title: "光合作用与细胞呼吸", Description: "能量代谢的核心过程", Category: "biology"},
			{ID: "bt-2", Title: "遗传的基本规律", Description: "孟德尔遗传定律", Category: "biology"},
			{ID: "bt-3", Title: "生态系统能量流动", Description: "能量在生态系统中的传递", Category: "biology"},
			{ID: "bt-4", Title: "细胞结构与物质运输", Description: "细胞膜、细胞器与物质跨膜运输", Category: "biology"},
		},
	}

	c.JSON(http.StatusOK, common.Success(resp))
}

// AddFavorite POST /api/v1/favorites — 添加收藏。
func (h *UserHandler) AddFavorite(c *gin.Context) {
	var req dto.FavoriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	fav := &user.Favorite{
		UserID:     uid,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Title:      req.Title,
	}

	if err := h.userSvc.AddFavorite(c.Request.Context(), fav); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(fav))
}

// SetReaction PUT /api/v1/reactions — 设置 like/dislike。
func (h *UserHandler) SetReaction(c *gin.Context) {
	var req dto.ReactionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	reaction := &user.Reaction{
		UserID:       uid,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		ReactionType: req.ReactionType,
		Visibility:   req.Visibility,
	}
	if err := h.userSvc.SetReaction(c.Request.Context(), reaction); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	summary, _ := h.userSvc.ReactionSummary(c.Request.Context(), uid, req.TargetType, req.TargetID, false)
	c.JSON(http.StatusOK, common.Success(summary))
}

// DeleteReaction DELETE /api/v1/reactions — 撤销反馈。
func (h *UserHandler) DeleteReaction(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	targetType := c.Query("target_type")
	targetID := c.Query("target_id")
	if targetType == "" || targetID == "" {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("target_type and target_id are required"), reqID))

		return
	}

	if err := h.userSvc.DeleteReaction(c.Request.Context(), uid, targetType, targetID); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(gin.H{"deleted": true}))
}

// ListReactions GET /api/v1/reactions — 当前用户反馈列表。
func (h *UserHandler) ListReactions(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	items, total, err := h.userSvc.ListReactions(c.Request.Context(), uid, 0, 20)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(common.PageResponse{Total: total, Page: 1, PageSize: 20, Items: items}))
}

// ReactionSummary GET /api/v1/reactions/summary — 目标反馈聚合。
func (h *UserHandler) ReactionSummary(c *gin.Context) {
	uid, ok := h.resolveUserID(c)
	if !ok {
		return
	}

	targetType := c.Query("target_type")
	targetID := c.Query("target_id")
	if targetType == "" || targetID == "" {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("target_type and target_id are required"), reqID))

		return
	}

	includeUsers, _ := strconv.ParseBool(c.DefaultQuery("include_users", "false"))
	summary, err := h.userSvc.ReactionSummary(c.Request.Context(), uid, targetType, targetID, includeUsers)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(summary))
}
