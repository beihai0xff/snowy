// Package storage 定义对象存储统一接口与适配实现（基础设施层）。
// 参考技术方案 §6.2.3。
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

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
	cfg     config.MinIOConfig
	once    sync.Once
	client  *minio.Client
	initErr error
}

// NewMinIOStorage 创建 MinIO 对象存储。
func NewMinIOStorage(cfg config.MinIOConfig) ObjectStorage {
	return &minioStorage{cfg: cfg}
}

func (s *minioStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	if err := s.ensureBucket(ctx, client); err != nil {
		return err
	}

	_, err = client.PutObject(ctx, s.bucketName(), key, reader, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio upload %s: %w", key, err)
	}

	return nil
}

func (s *minioStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	obj, err := client.GetObject(ctx, s.bucketName(), key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio get object %s: %w", key, err)
	}

	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()

		return nil, fmt.Errorf("minio stat object %s: %w", key, err)
	}

	return obj, nil
}

func (s *minioStorage) Delete(ctx context.Context, key string) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	if err := client.RemoveObject(ctx, s.bucketName(), key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio delete %s: %w", key, err)
	}

	return nil
}

func (s *minioStorage) GetURL(ctx context.Context, key string) (string, error) {
	client, err := s.getClient()
	if err != nil {
		return "", err
	}

	urlValue, err := client.PresignedGetObject(ctx, s.bucketName(), key, 15*time.Minute, url.Values{})
	if err != nil {
		return "", fmt.Errorf("minio presign %s: %w", key, err)
	}

	return urlValue.String(), nil
}

func (s *minioStorage) getClient() (*minio.Client, error) {
	s.once.Do(func() {
		s.client, s.initErr = minio.New(s.cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.cfg.AccessKey, s.cfg.SecretKey, ""),
			Secure: s.cfg.UseSSL,
		})
	})

	if s.initErr != nil {
		return nil, fmt.Errorf("init minio client: %w", s.initErr)
	}

	return s.client, nil
}

func (s *minioStorage) ensureBucket(ctx context.Context, client *minio.Client) error {
	bucket := s.bucketName()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", bucket, err)
	}

	if exists {
		return nil
	}

	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", bucket, err)
	}

	return nil
}

func (s *minioStorage) bucketName() string {
	bucket := strings.TrimSpace(s.cfg.Bucket)
	if bucket == "" {
		return "snowy"
	}

	return bucket
}
