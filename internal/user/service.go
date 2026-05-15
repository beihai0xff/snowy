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
//
//nolint:interfacebloat // The handler layer depends on this existing user service facade.
type Service interface {
	// GoogleLogin 通过 Google OAuth 登录（查找已有用户或自动注册），返回 access / refresh token。
	GoogleLogin(ctx context.Context, info *GoogleUserInfo) (accessToken, refreshToken string, err error)
	// GetProfile 获取用户资料。
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	// EnsureAnonymousUser 确保默认匿名用户存在。
	EnsureAnonymousUser(ctx context.Context) (*User, error)
	// GetHistory 获取历史记录。
	GetHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error)
	// AddHistory 添加历史记录。
	AddHistory(ctx context.Context, item *HistoryItem) error
	// AddFavorite 添加收藏。
	AddFavorite(ctx context.Context, fav *Favorite) error
	// ListFavorites 列出收藏。
	ListFavorites(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error)
	// EmailRegister 通过邮箱注册。
	EmailRegister(
		ctx context.Context,
		email, password, nickname string,
	) (accessToken, refreshToken string, profile *User, err error)
	// EmailLogin 通过邮箱登录。
	EmailLogin(ctx context.Context, email, password string) (accessToken, refreshToken string, profile *User, err error)
	// SetReaction 设置 like/dislike 反馈。
	SetReaction(ctx context.Context, reaction *Reaction) error
	// DeleteReaction 撤销反馈。
	DeleteReaction(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error
	// ListReactions 列出用户反馈。
	ListReactions(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Reaction, int64, error)
	// ReactionSummary 获取目标反馈聚合。
	ReactionSummary(
		ctx context.Context,
		userID uuid.UUID,
		targetType string,
		targetID string,
		includeUsers bool,
	) (*ReactionSummary, error)
}
