// Package callback 定义 Eino 回调钩子实现。
// 参考技术方案 §10.7 — Eino Callbacks 全链路。
package callback

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// NodeCallback 是图节点回调统一接口。
type NodeCallback interface {
	OnNodeStart(ctx context.Context, nodeName string, input any)
	OnNodeEnd(ctx context.Context, nodeName string, output any, err error)
}

// AuditLogger 审计日志回调。
type AuditLogger struct{}

// NewAuditLogger 创建审计日志回调。
func NewAuditLogger() *AuditLogger { return &AuditLogger{} }

// OnNodeStart 节点执行前记录审计日志。
func (a *AuditLogger) OnNodeStart(ctx context.Context, nodeName string, _ any) {
	slog.InfoContext(ctx, "agent node started", "node", nodeName)
}

// OnNodeEnd 节点执行后记录审计日志。
func (a *AuditLogger) OnNodeEnd(ctx context.Context, nodeName string, _ any, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "agent node failed", "node", nodeName, "error", err)

		return
	}

	slog.InfoContext(ctx, "agent node completed", "node", nodeName)
}

var (
	nodeDurationHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "snowy",
		Subsystem: "agent",
		Name:      "node_duration_seconds",
		Help:      "Latency of agent node execution.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"node", "status"})
	nodeExecutionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "snowy",
		Subsystem: "agent",
		Name:      "node_executions_total",
		Help:      "Total number of agent node executions.",
	}, []string{"node", "status"})
)

// MetricsCollector Prometheus 指标收集回调。
type MetricsCollector struct {
	startTimes sync.Map
}

// NewMetricsCollector 创建指标采集回调。
func NewMetricsCollector() *MetricsCollector { return &MetricsCollector{} }

// OnNodeStart 记录节点开始时间。
func (m *MetricsCollector) OnNodeStart(ctx context.Context, nodeName string, _ any) {
	m.startTimes.Store(metricKey(ctx, nodeName), time.Now())
}

// OnNodeEnd 记录节点耗时和状态。
func (m *MetricsCollector) OnNodeEnd(ctx context.Context, nodeName string, _ any, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	if start, ok := m.startTimes.LoadAndDelete(metricKey(ctx, nodeName)); ok {
		if startedAt, ok := start.(time.Time); ok {
			nodeDurationHistogram.WithLabelValues(nodeName, status).Observe(time.Since(startedAt).Seconds())
		}
	}
	nodeExecutionCounter.WithLabelValues(nodeName, status).Inc()
}

// OTelTracer OpenTelemetry 追踪回调。
type OTelTracer struct {
	tracer trace.Tracer
	spans  sync.Map
}

// NewOTelTracer 创建 OTel 回调。
func NewOTelTracer() *OTelTracer {
	return &OTelTracer{tracer: otel.Tracer("github.com/beihai0xff/snowy/internal/agent")}
}

// OnNodeStart 创建追踪 Span。
func (o *OTelTracer) OnNodeStart(ctx context.Context, nodeName string, _ any) {
	_, span := o.tracer.Start(ctx, nodeName)
	o.spans.Store(metricKey(ctx, nodeName), span)
}

// OnNodeEnd 结束追踪 Span。
func (o *OTelTracer) OnNodeEnd(ctx context.Context, nodeName string, _ any, err error) {
	value, ok := o.spans.LoadAndDelete(metricKey(ctx, nodeName))
	if !ok {
		return
	}
	span, ok := value.(trace.Span)
	if !ok {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "completed")
	}
	span.End()
}

func metricKey(ctx context.Context, nodeName string) string {
	requestID := common.RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = common.TraceIDFromContext(ctx)
	}
	if requestID == "" {
		requestID = fmt.Sprintf("fallback-%s", nodeName)
	}
	return requestID + ":" + nodeName
}
