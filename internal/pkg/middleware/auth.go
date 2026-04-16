// Package middleware 定义 HTTP 中间件。
// 参考技术方案 §9.1 API Gateway / BFF。
package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// Auth 鉴权中间件 — 当前已禁用登录，所有请求自动使用默认匿名用户。
// 保留函数签名以保持兼容。
func Auth(_ config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", common.DefaultUserID)
		c.Set("role", "student")
		c.Set("anonymous", false)

		ctx := common.WithUserID(c.Request.Context(), common.DefaultUserID)
		c.Request = c.Request.WithContext(ctx)

		slog.DebugContext(c.Request.Context(), "open-access mode", "user_id", common.DefaultUserID)
		c.Next()
	}
}

// RequireAuth 强制要求认证（当前已禁用 — 总是放行）。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func abortWithError(c *gin.Context, err *common.AppError) {
	requestID := common.RequestIDFromContext(c.Request.Context())
	c.AbortWithStatusJSON(err.HTTPStatus, common.Fail(err, requestID))
}
