package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// RateLimiter 限流器接口，由基础设施层（repo/redis）实现。
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimit 限流中间件。
// 参考技术方案 §18A.4 — 已认证 60/min，匿名 10/min。
func RateLimit(limiter RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string
		var limit int

		anonymous, _ := c.Get("anonymous")
		if anonymous == true {
			key = fmt.Sprintf("rate:%s:1m", c.ClientIP())
			limit = cfg.AnonymousRPM
		} else {
			userID, _ := c.Get("user_id")
			key = fmt.Sprintf("rate:%s:1m", userID)
			limit = cfg.AuthenticatedRPM
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
