// Package callback 定义 Eino 回调钩子实现。
// 参考技术方案 §10.7 — Eino Callbacks 全链路。
package callback

import (
	"context"
	"log/slog"
)

// AuditLogger 审计日志回调。
type AuditLogger struct{}

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

// MetricsCollector Prometheus 指标收集回调。
type MetricsCollector struct{}

// OnNodeStart 记录节点开始时间。
func (m *MetricsCollector) OnNodeStart(_ context.Context, _ string, _ any) {
	// TODO: 记录 Prometheus histogram 开始时间
}

// OnNodeEnd 记录节点耗时和状态。
func (m *MetricsCollector) OnNodeEnd(_ context.Context, _ string, _ any, _ error) {
	// TODO: 记录 Prometheus histogram 耗时、counter 成功/失败
}

// OTelTracer OpenTelemetry 追踪回调。
type OTelTracer struct{}

// OnNodeStart 创建追踪 Span。
func (o *OTelTracer) OnNodeStart(_ context.Context, _ string, _ any) {
	// TODO: 使用 otel.Tracer 创建 Span
}

// OnNodeEnd 结束追踪 Span。
func (o *OTelTracer) OnNodeEnd(_ context.Context, _ string, _ any, _ error) {
	// TODO: 结束 Span，记录状态
}
