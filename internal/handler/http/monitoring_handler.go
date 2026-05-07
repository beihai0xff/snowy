package http

import (
	"net/http"

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

	c.JSON(http.StatusOK, common.Success(h.llmRecorder.Dashboard()))
}
