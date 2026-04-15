package common

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── ErrorCode Tests ──────────────────────────────────────

func TestErrorCode_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ErrorCode
		want string
	}{
		{"OK", ErrOK, "[OK] 成功"},
		{"InvalidInput", ErrInvalidInput, "[INVALID_INPUT] 请求参数不合法"},
		{"Internal", ErrInternal, "[INTERNAL_ERROR] 服务内部错误"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestErrorCode_WithMessage(t *testing.T) {
	original := ErrInvalidInput
	custom := original.WithMessage("手机号格式错误")

	assert.Equal(t, "手机号格式错误", custom.Message)
	assert.Equal(t, original.Code, custom.Code)
	assert.Equal(t, original.HTTPStatus, custom.HTTPStatus)
	// 原始实例不变
	assert.Equal(t, "请求参数不合法", original.Message)
}

// ── Response Tests ───────────────────────────────────────

func TestSuccess(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := Success(data)

	assert.Equal(t, "OK", resp.Code)
	assert.Equal(t, "成功", resp.Message)
	assert.Equal(t, data, resp.Data)
	assert.Empty(t, resp.RequestID)
}

func TestSuccessWithRequestID(t *testing.T) {
	resp := SuccessWithRequestID("payload", "req-123")

	assert.Equal(t, "OK", resp.Code)
	assert.Equal(t, "payload", resp.Data)
	assert.Equal(t, "req-123", resp.RequestID)
}

func TestFail(t *testing.T) {
	resp := Fail(ErrRateLimited, "req-456")

	assert.Equal(t, "RATE_LIMITED", resp.Code)
	assert.Equal(t, "请求频率超限", resp.Message)
	assert.Nil(t, resp.Data)
	assert.Equal(t, "req-456", resp.RequestID)
}

// ── PageRequest Tests ────────────────────────────────────

func TestPageRequest_Offset(t *testing.T) {
	tests := []struct {
		page, size, want int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 20, 40},
		{5, 15, 60},
	}
	for _, tt := range tests {
		pr := PageRequest{Page: tt.page, PageSize: tt.size}
		assert.Equal(t, tt.want, pr.Offset())
	}
}

// ── Context Helper Tests ─────────────────────────────────

func TestContextHelpers_UserID(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, UserIDFromContext(ctx))

	ctx = WithUserID(ctx, "user-abc")
	assert.Equal(t, "user-abc", UserIDFromContext(ctx))
}

func TestContextHelpers_RequestID(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, RequestIDFromContext(ctx))

	ctx = WithRequestID(ctx, "req-xyz")
	assert.Equal(t, "req-xyz", RequestIDFromContext(ctx))
}

func TestContextHelpers_TraceID(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, TraceIDFromContext(ctx))

	ctx = WithTraceID(ctx, "trace-001")
	assert.Equal(t, "trace-001", TraceIDFromContext(ctx))
}

// ── Predefined Error Codes ───────────────────────────────

func TestPredefinedErrorCodes(t *testing.T) {
	assert.Equal(t, http.StatusOK, ErrOK.HTTPStatus)
	assert.Equal(t, http.StatusBadRequest, ErrInvalidInput.HTTPStatus)
	assert.Equal(t, http.StatusUnauthorized, ErrUnauthorized.HTTPStatus)
	assert.Equal(t, http.StatusForbidden, ErrForbidden.HTTPStatus)
	assert.Equal(t, http.StatusTooManyRequests, ErrRateLimited.HTTPStatus)
	assert.Equal(t, http.StatusInternalServerError, ErrInternal.HTTPStatus)
	assert.Equal(t, http.StatusGatewayTimeout, ErrModelTimeout.HTTPStatus)
	assert.Equal(t, http.StatusServiceUnavailable, ErrModelUnavailable.HTTPStatus)
}
