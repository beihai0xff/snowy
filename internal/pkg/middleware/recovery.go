package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// Recovery panic 恢复中间件，返回结构化错误。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					"error", r,
					"stack", string(debug.Stack()),
				)

				reqID := common.RequestIDFromContext(c.Request.Context())
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					common.Fail(common.ErrInternal, reqID),
				)
			}
		}()

		c.Next()
	}
}
