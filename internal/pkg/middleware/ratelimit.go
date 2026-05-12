package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// RateLimiter 限流器接口，由基础设施层（repo/redis）实现。
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

const (
	defaultAuthenticatedRPM = 300
	defaultAnonymousRPM     = 120
)

// RateLimit 限流中间件。
// v5 前端会并发拉取 profile/history/favorites/reactions/model packages/monitoring 等多条数据；
// 默认限额必须覆盖正常学习链路的突发请求，同时保留防滥用保护。
func RateLimit(limiter RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			key   string
			limit int
		)

		if isRateLimitExempt(c) {
			c.Next()

			return
		}

		anonymous, _ := c.Get("anonymous")
		if anonymous == true {
			key = fmt.Sprintf("rate:%s:1m", c.ClientIP())
			limit = effectiveLimit(cfg.AnonymousRPM, defaultAnonymousRPM)
		} else {
			userID, _ := c.Get("user_id")
			key = fmt.Sprintf("rate:%s:1m", userID)
			limit = effectiveLimit(cfg.AuthenticatedRPM, defaultAuthenticatedRPM)
		}

		allowed, err := limiter.Allow(c.Request.Context(), key, limit, time.Minute)
		if err != nil {
			c.Next() // 限流器故障时放行

			return
		}

		if !allowed {
			requestID, _ := c.Get("request_id")
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				common.Fail(common.ErrRateLimited, fmt.Sprintf("%v", requestID)),
			)

			return
		}

		c.Next()
	}
}

func effectiveLimit(configured int, fallback int) int {
	if configured > 0 {
		return configured
	}

	return fallback
}

func isRateLimitExempt(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}

	path := strings.TrimSpace(c.FullPath())
	if path == "" {
		path = c.Request.URL.Path
	}

	switch path {
	case "/api/v1/user/profile", "/api/v1/history", "/api/v1/favorites", "/api/v1/reactions", "/api/v1/reactions/summary", "/api/v1/modeling/packages", "/api/v1/answers", "/api/v1/monitoring/llm", "/api/v1/recommendations":
		return c.Request.Method == http.MethodGet
	default:
		return false
	}
}
