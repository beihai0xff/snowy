package common

import (
	"fmt"
	"net/http"
)

// AppError 统一错误码，参考技术方案 §17.9。
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithMessage 返回一个携带自定义消息的新 AppError（不修改原始实例）。
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    msg,
		HTTPStatus: e.HTTPStatus,
	}
}

// ── 预定义错误码 ────────────────────────────────────────────

var (
	ErrOK             = &AppError{Code: "OK", Message: "成功", HTTPStatus: http.StatusOK}
	ErrInvalidInput   = &AppError{Code: "INVALID_INPUT", Message: "请求参数不合法", HTTPStatus: http.StatusBadRequest}
	ErrUnauthorized   = &AppError{Code: "UNAUTHORIZED", Message: "未认证", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden      = &AppError{Code: "FORBIDDEN", Message: "无权限", HTTPStatus: http.StatusForbidden}
	ErrSearchNoResult = &AppError{
		Code:       "SEARCH_NO_RESULT",
		Message:    "未找到相关结果，请尝试更换关键词",
		HTTPStatus: http.StatusOK,
	}
	ErrModelTimeout = &AppError{
		Code:       "MODEL_TIMEOUT",
		Message:    "大模型调用超时",
		HTTPStatus: http.StatusGatewayTimeout,
	}
	ErrModelUnavailable = &AppError{
		Code:       "MODEL_UNAVAILABLE",
		Message:    "大模型服务不可用",
		HTTPStatus: http.StatusServiceUnavailable,
	}
	ErrSchemaValidationFailed = &AppError{
		Code:       "SCHEMA_VALIDATION_FAILED",
		Message:    "模型输出结构校验失败",
		HTTPStatus: http.StatusBadGateway,
	}
	ErrLowConfidence         = &AppError{Code: "LOW_CONFIDENCE", Message: "结果可信度不足", HTTPStatus: http.StatusOK}
	ErrConditionInsufficient = &AppError{
		Code:       "CONDITION_INSUFFICIENT",
		Message:    "题干条件不足，需补充",
		HTTPStatus: http.StatusOK,
	}
	ErrRateLimited = &AppError{
		Code:       "RATE_LIMITED",
		Message:    "请求频率超限",
		HTTPStatus: http.StatusTooManyRequests,
	}
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "服务内部错误",
		HTTPStatus: http.StatusInternalServerError,
	}
)
