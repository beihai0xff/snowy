package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/share"
)

type shareRepo struct {
	db *gorm.DB
}

// NewShareRepository 创建 share.Repository 实现。
func NewShareRepository(db *gorm.DB) share.Repository {
	return &shareRepo{db: db}
}

func (r *shareRepo) Create(ctx context.Context, p *share.Package) error {
	if p == nil {
		return errors.New("share package is nil")
	}

	var raw any
	if len(p.SnapshotJSON) > 0 {
		if err := json.Unmarshal(p.SnapshotJSON, &raw); err != nil {
			return fmt.Errorf("decode snapshot: %w", err)
		}
	}

	createdAt := p.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	row := &packageShareSchema{
		Token:        p.Token,
		PackageID:    p.PackageID,
		SnapshotJSON: newJSONValue(raw),
		CreatedBy:    p.CreatedBy,
		ExpiresAt:    p.ExpiresAt,
		Mode:         defaultIfBlank(p.Mode, "view"),
		CreatedAt:    createdAt,
	}
	if err := dbFromContext(ctx, r.db).Create(row).Error; err != nil {
		return fmt.Errorf("insert package share: %w", err)
	}

	return nil
}

func (r *shareRepo) GetByToken(ctx context.Context, token string) (*share.Package, error) {
	var row packageShareSchema
	err := dbFromContext(ctx, r.db).Where("token = ?", token).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, share.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query package share: %w", err)
	}
	if row.ExpiresAt != nil && row.ExpiresAt.Before(time.Now()) {
		return nil, share.ErrNotFound
	}

	raw, err := json.Marshal(row.SnapshotJSON.Data)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}

	return &share.Package{
		Token:        row.Token,
		PackageID:    row.PackageID,
		SnapshotJSON: raw,
		CreatedBy:    row.CreatedBy,
		ExpiresAt:    row.ExpiresAt,
		Mode:         row.Mode,
		CreatedAt:    row.CreatedAt,
	}, nil
}

func defaultIfBlank(s, def string) string {
	if s == "" {
		return def
	}

	return s
}
