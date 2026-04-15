package common

import "context"

type contextKey string

const (
	ctxKeyUserID    contextKey = "user_id"
	ctxKeyRequestID contextKey = "request_id"
	ctxKeyTraceID   contextKey = "trace_id"
)

// WithUserID 将 UserID 写入 context。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// UserIDFromContext 从 context 中提取 UserID。
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// WithRequestID 将 RequestID 写入 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

// RequestIDFromContext 从 context 中提取 RequestID。
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// WithTraceID 将 TraceID 写入 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}

// TraceIDFromContext 从 context 中提取 TraceID。
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyTraceID).(string)
	return v
}
