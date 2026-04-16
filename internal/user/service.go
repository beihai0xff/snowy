package user

import (
	"context"

	"github.com/google/uuid"
)

// GoogleUserInfo Google OAuth 回调中获取的用户信息。
type GoogleUserInfo struct {
	GoogleID  string
	Email     string
	Name      string
	AvatarURL string
}

// Service 用户域应用服务接口。
type Service interface {
	// GoogleLogin 通过 Google OAuth 登录（查找已有用户或自动注册），返回 access / refresh token。
	GoogleLogin(ctx context.Context, info *GoogleUserInfo) (accessToken, refreshToken string, err error)
	// GetProfile 获取用户资料。
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	// GetHistory 获取历史记录。
	GetHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error)
	// AddFavorite 添加收藏。
	AddFavorite(ctx context.Context, fav *Favorite) error
	// ListFavorites 列出收藏。
	ListFavorites(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error)
}
