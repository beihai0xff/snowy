// Package storage 定义对象存储统一接口与适配实现（基础设施层）。
// 参考技术方案 §6.2.3。
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// ObjectStorage 对象存储统一接口。
type ObjectStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
}

// minioStorage MinIO / S3 兼容对象存储实现。
type minioStorage struct {
	cfg config.MinIOConfig
}

// NewMinIOStorage 创建 MinIO 对象存储。
func NewMinIOStorage(cfg config.MinIOConfig) ObjectStorage {
	return &minioStorage{cfg: cfg}
}

func (s *minioStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	// TODO: 使用 MinIO SDK 实现
	return fmt.Errorf("minio upload: not implemented")
}

func (s *minioStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("minio download: not implemented")
}

func (s *minioStorage) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("minio delete: not implemented")
}

func (s *minioStorage) GetURL(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("minio get url: not implemented")
}
