package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/user"
)

// UserHandler 用户 HTTP Handler。
// 参考技术方案 §17.7 & §18A。
type UserHandler struct {
	userSvc   user.Service
	googleCfg config.GoogleOAuthConfig
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(userSvc user.Service, googleCfg config.GoogleOAuthConfig) *UserHandler {
	return &UserHandler{userSvc: userSvc, googleCfg: googleCfg}
}

// GoogleLogin POST /api/v1/auth/google/callback — Google OAuth 登录。
// 前端传入 Google ID Token，后端验证后签发 JWT。
func (h *UserHandler) GoogleLogin(c *gin.Context) {
	var req dto.GoogleLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	info, err := h.verifyGoogleIDToken(req.IDToken)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized.WithMessage("Google 登录验证失败"), reqID))

		return
	}

	access, refresh, err := h.userSvc.GoogleLogin(c.Request.Context(), info)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(dto.LoginResp{
		AccessToken:  access,
		RefreshToken: refresh,
	}))
}

// verifyGoogleIDToken 验证 Google ID Token 并提取用户信息。
// 通过 Google tokeninfo 端点验证 token 的有效性和签发者。
func (h *UserHandler) verifyGoogleIDToken(idToken string) (*user.GoogleUserInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("verify google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token verification failed: status %d", resp.StatusCode)
	}

	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Aud           string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode google token payload: %w", err)
	}

	// 验证 audience 匹配
	if h.googleCfg.ClientID != "" && payload.Aud != h.googleCfg.ClientID {
		return nil, fmt.Errorf("google token audience mismatch: got %s", payload.Aud)
	}

	if payload.Sub == "" {
		return nil, fmt.Errorf("google token missing sub claim")
	}

	return &user.GoogleUserInfo{
		GoogleID:  payload.Sub,
		Email:     payload.Email,
		Name:      payload.Name,
		AvatarURL: payload.Picture,
	}, nil
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

	uid, err := uuid.Parse(userID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized, reqID))

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
	userID := common.UserIDFromContext(c.Request.Context())

	uid, err := uuid.Parse(userID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized, reqID))

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

	userID := common.UserIDFromContext(c.Request.Context())

	uid, err := uuid.Parse(userID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusUnauthorized, common.Fail(common.ErrUnauthorized, reqID))

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
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(fav))
}
