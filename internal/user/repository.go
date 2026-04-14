package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository 用户持久化端口（DDD Port）。
// 由基础设施层（internal/store/mysql）实现。
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// FavoriteRepository 收藏持久化端口。
type FavoriteRepository interface {
	Add(ctx context.Context, fav *Favorite) error
	Remove(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error)
}

// HistoryRepository 历史记录持久化端口。
type HistoryRepository interface {
	Add(ctx context.Context, item *HistoryItem) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error)
}
