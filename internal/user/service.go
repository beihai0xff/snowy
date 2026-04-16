package user

import (
	"context"

	"github.com/google/uuid"
)

// Service 用户域应用服务接口。
type Service interface {
	// Register 注册新用户。
	Register(ctx context.Context, phone, nickname string) (*User, error)
	// Login 登录（手机号+验证码），返回 access / refresh token。
	Login(ctx context.Context, phone, code string) (accessToken, refreshToken string, err error)
	// GetProfile 获取用户资料。
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	// GetHistory 获取历史记录。
	GetHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error)
	// AddFavorite 添加收藏。
	AddFavorite(ctx context.Context, fav *Favorite) error
	// ListFavorites 列出收藏。
	ListFavorites(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error)
}
