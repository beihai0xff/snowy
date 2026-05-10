package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository 用户持久化端口（DDD Port）。
// 由基础设施层（internal/repo/mysql）实现。
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
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

// ReactionRepository stores and aggregates user quality feedback.
type ReactionRepository interface {
	Upsert(ctx context.Context, reaction *Reaction) error
	Delete(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Reaction, int64, error)
	Summary(ctx context.Context, userID uuid.UUID, targetType string, targetID string, includeUsers bool) (*ReactionSummary, error)
}
