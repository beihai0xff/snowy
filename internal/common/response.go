package common

// APIResponse 统一 API 响应结构，参考技术方案 §17.9。
type APIResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Success 返回成功响应。
func Success(data any) *APIResponse {
	return &APIResponse{
		Code:    ErrOK.Code,
		Message: ErrOK.Message,
		Data:    data,
	}
}

// SuccessWithRequestID 返回带 RequestID 的成功响应。
func SuccessWithRequestID(data any, requestID string) *APIResponse {
	return &APIResponse{
		Code:      ErrOK.Code,
		Message:   ErrOK.Message,
		Data:      data,
		RequestID: requestID,
	}
}

// Fail 返回错误响应。
func Fail(err *ErrorCode, requestID string) *APIResponse {
	return &APIResponse{
		Code:      err.Code,
		Message:   err.Message,
		RequestID: requestID,
	}
}

// PageRequest 分页请求参数。
type PageRequest struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"page_size" form:"page_size" binding:"min=1,max=100"`
}

// Offset 返回数据库偏移量。
func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PageResponse 分页响应。
type PageResponse struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Items    any   `json:"items"`
}
