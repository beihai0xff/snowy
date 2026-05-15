//revive:disable:var-naming
package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/monitoring"
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// MonitoringHandler exposes read-only observability data for the Snowy dashboard.
type MonitoringHandler struct {
	llmRecorder *monitoring.LLMRecorder
}

// NewMonitoringHandler creates a monitoring handler.
func NewMonitoringHandler(llmRecorder *monitoring.LLMRecorder) *MonitoringHandler {
	return &MonitoringHandler{llmRecorder: llmRecorder}
}

// LLMDashboard GET /api/v1/monitoring/llm — LLM PE and latency metrics.
func (h *MonitoringHandler) LLMDashboard(c *gin.Context) {
	if h == nil || h.llmRecorder == nil {
		c.JSON(http.StatusOK, common.Success(monitoring.LLMDashboard{}))

		return
	}

	filter, err := buildLLMRecordFilter(c)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(h.llmRecorder.Dashboard(filter)))
}

func buildLLMRecordFilter(c *gin.Context) (monitoring.LLMRecordFilter, error) {
	filter := monitoring.LLMRecordFilter{
		UserID:    strings.TrimSpace(c.Query("user_id")),
		Provider:  strings.TrimSpace(c.Query("provider")),
		Model:     strings.TrimSpace(c.Query("model")),
		Operation: strings.TrimSpace(c.Query("operation")),
	}
	if limitText := strings.TrimSpace(c.Query("limit")); limitText != "" {
		limit, err := strconv.Atoi(limitText)
		if err != nil {
			return filter, err
		}

		filter.Limit = limit
	}

	if sinceText := strings.TrimSpace(c.Query("since")); sinceText != "" {
		since, err := time.Parse(time.RFC3339, sinceText)
		if err != nil {
			return filter, err
		}

		filter.Since = since
	}

	if untilText := strings.TrimSpace(c.Query("until")); untilText != "" {
		until, err := time.Parse(time.RFC3339, untilText)
		if err != nil {
			return filter, err
		}

		filter.Until = until
	}

	return filter, nil
}
