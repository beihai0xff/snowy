// Package share 包含 v7 §6.2 静态分享相关领域类型。
package share

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound 分享不存在或已过期。
var ErrNotFound = errors.New("share not found")

// Package 是分享落库的快照单元。
type Package struct {
	Token        string
	PackageID    uuid.UUID
	SnapshotJSON []byte
	CreatedBy    uuid.UUID
	ExpiresAt    *time.Time
	Mode         string
	CreatedAt    time.Time
}

// Repository 抽象分享存储。
type Repository interface {
	Create(ctx context.Context, p *Package) error
	GetByToken(ctx context.Context, token string) (*Package, error)
}
