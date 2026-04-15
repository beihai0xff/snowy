// Package middleware 定义 HTTP 中间件。
// 参考技术方案 §9.1 API Gateway / BFF。
package middleware

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// Auth JWT 鉴权中间件。
// 参考技术方案 §18A — 支持匿名试用。
func Auth(cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 匿名用户 — 允许通过但不设置 UserID
			c.Set("anonymous", true)
			c.Next()

			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			abortWithError(c, common.ErrUnauthorized)

			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}

			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			abortWithError(c, common.ErrUnauthorized)

			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			abortWithError(c, common.ErrUnauthorized)

			return
		}

		userID, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)

		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("anonymous", false)

		ctx := common.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)

		slog.DebugContext(c.Request.Context(), "authenticated", "user_id", userID, "role", role)
		c.Next()
	}
}

// RequireAuth 强制要求认证（拒绝匿名）。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		anonymous, _ := c.Get("anonymous")
		if anonymous == true {
			abortWithError(c, common.ErrUnauthorized)

			return
		}

		c.Next()
	}
}

func abortWithError(c *gin.Context, err *common.AppError) {
	requestID := common.RequestIDFromContext(c.Request.Context())
	c.AbortWithStatusJSON(err.HTTPStatus, common.Fail(err, requestID))
}
