// Package ingest 定义内容入库服务。
package ingest

import "context"

// Service 内容入库域应用服务接口。
// 参考技术方案 §12.5。
type Service interface {
	// Ingest 执行内容入库：导入 → 清洗 → 切片 → embedding → 建索引 → 元数据写入。
	Ingest(ctx context.Context, req *Request) error
}

// Request 入库请求。
type Request struct {
	SourceType string `json:"source_type"`
	Subject    string `json:"subject"`
	FilePath   string `json:"file_path,omitempty"`
	RawContent []byte `json:"raw_content,omitempty"`
}
