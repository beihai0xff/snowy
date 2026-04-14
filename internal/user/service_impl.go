package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/config"
)

// serviceImpl 用户域应用服务实现。
type serviceImpl struct {
	repo     Repository
	favRepo  FavoriteRepository
	histRepo HistoryRepository
	authCfg  config.AuthConfig
}

// NewService 创建用户域应用服务。
func NewService(
	repo Repository,
	favRepo FavoriteRepository,
	histRepo HistoryRepository,
	authCfg config.AuthConfig,
) Service {
	return &serviceImpl{
		repo:     repo,
		favRepo:  favRepo,
		histRepo: histRepo,
		authCfg:  authCfg,
	}
}

func (s *serviceImpl) Register(ctx context.Context, phone, nickname string) (*User, error) {
	u := &User{
		ID:        uuid.New(),
		Phone:     phone,
		Nickname:  nickname,
		Role:      RoleStudent,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	slog.InfoContext(ctx, "user registered", "user_id", u.ID, "phone", phone)
	return u, nil
}

func (s *serviceImpl) Login(ctx context.Context, phone, code string) (string, string, error) {
	// TODO: 校验验证码

	u, err := s.repo.GetByPhone(ctx, phone)
	if err != nil {
		return "", "", fmt.Errorf("get user by phone: %w", err)
	}

	if err := s.repo.UpdateLastLogin(ctx, u.ID); err != nil {
		slog.WarnContext(ctx, "update last login failed", "error", err)
	}

	accessToken, err := s.generateToken(u, s.authCfg.AccessTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(u, s.authCfg.RefreshTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *serviceImpl) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *serviceImpl) GetHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error) {
	return s.histRepo.ListByUser(ctx, userID, offset, limit)
}

func (s *serviceImpl) AddFavorite(ctx context.Context, fav *Favorite) error {
	fav.ID = uuid.New()
	fav.CreatedAt = time.Now()
	return s.favRepo.Add(ctx, fav)
}

func (s *serviceImpl) ListFavorites(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error) {
	return s.favRepo.ListByUser(ctx, userID, offset, limit)
}

func (s *serviceImpl) generateToken(u *User, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": u.ID.String(),
		"role":    string(u.Role),
		"exp":     time.Now().Add(ttl).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.authCfg.JWTSecret))
}
