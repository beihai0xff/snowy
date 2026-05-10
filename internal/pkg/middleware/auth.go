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

// Auth 鉴权中间件 — v5 支持 Bearer JWT；未携带或无效 token 时回落默认匿名用户，
// 以保持旧匿名试用链路可用。
func Auth(authCfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := common.DefaultUserID
		role := "student"
		anonymous := true

		if tokenText := bearerToken(c.GetHeader("Authorization")); tokenText != "" {
			if claims, err := parseJWT(tokenText, authCfg.JWTSecret); err == nil {
				if claimUserID, ok := claims["user_id"].(string); ok && strings.TrimSpace(claimUserID) != "" {
					userID = claimUserID
					anonymous = false
				}
				if claimRole, ok := claims["role"].(string); ok && strings.TrimSpace(claimRole) != "" {
					role = claimRole
				}
			} else {
				slog.DebugContext(c.Request.Context(), "invalid bearer token; fallback anonymous", "error", err)
			}
		}

		c.Set("user_id", userID)
		c.Set("role", role)
		c.Set("anonymous", anonymous)

		ctx := common.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)

		slog.DebugContext(c.Request.Context(), "auth resolved", "user_id", userID, "anonymous", anonymous)
		c.Next()
	}
}

// RequireAuth 强制要求认证。当前仅用于未来需要禁止匿名写入的路由。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		anonymous, _ := c.Get("anonymous")
		if value, ok := anonymous.(bool); ok && value {
			reqID := common.RequestIDFromContext(c.Request.Context())
			c.AbortWithStatusJSON(common.ErrUnauthorized.HTTPStatus, common.Fail(common.ErrUnauthorized, reqID))

			return
		}

		c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func parseJWT(tokenText string, secret string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenText, claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
