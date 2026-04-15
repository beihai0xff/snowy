package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件，基于 slog 结构化输出。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if reqID, exists := c.Get("request_id"); exists {
			attrs = append(attrs, "request_id", reqID)
		}

		if userID, exists := c.Get("user_id"); exists {
			attrs = append(attrs, "user_id", userID)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			slog.ErrorContext(c.Request.Context(), "request completed", attrs...)
		case status >= 400:
			slog.WarnContext(c.Request.Context(), "request completed", attrs...)
		default:
			slog.InfoContext(c.Request.Context(), "request completed", attrs...)
		}
	}
}
