package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	irepo "github.com/beihai0xff/snowy/internal/repo"
)

// serviceImpl 用户域应用服务实现。
type serviceImpl struct {
	repo       Repository
	favRepo    FavoriteRepository
	histRepo   HistoryRepository
	transactor irepo.Transactor
	authCfg    config.AuthConfig
}

// NewService 创建用户域应用服务。
func NewService(
	repo Repository,
	favRepo FavoriteRepository,
	histRepo HistoryRepository,
	transactor irepo.Transactor,
	authCfg config.AuthConfig,
) Service {
	return &serviceImpl{
		repo:       repo,
		favRepo:    favRepo,
		histRepo:   histRepo,
		transactor: transactor,
		authCfg:    authCfg,
	}
}

// GoogleLogin 通过 Google 用户信息查找或创建用户，返回 JWT token 对。
func (s *serviceImpl) GoogleLogin(ctx context.Context, info *GoogleUserInfo) (string, string, error) {
	if info == nil || info.GoogleID == "" {
		return "", "", errors.New("google user info is required")
	}

	// 尝试用 google_id 查找已有用户
	u, err := s.repo.GetByGoogleID(ctx, info.GoogleID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		// 数据库查询失败 — 不是"用户不存在"
		return "", "", fmt.Errorf("lookup google user: %w", err)
	}

	if errors.Is(err, ErrUserNotFound) {
		// 用户不存在 — 自动注册
		u = &User{
			ID:          uuid.New(),
			GoogleID:    info.GoogleID,
			Email:       info.Email,
			Nickname:    info.Name,
			AvatarURL:   info.AvatarURL,
			Role:        RoleStudent,
			LastLoginAt: time.Now(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		history := &HistoryItem{
			ID:         uuid.New(),
			UserID:     u.ID,
			ActionType: "register",
			Query:      "Google 账号注册",
			CreatedAt:  u.CreatedAt,
		}

		txErr := s.withTransaction(ctx, func(txCtx context.Context) error {
			if createErr := s.repo.Create(txCtx, u); createErr != nil {
				return fmt.Errorf("create user: %w", createErr)
			}

			if histErr := s.histRepo.Add(txCtx, history); histErr != nil {
				return fmt.Errorf("add register history: %w", histErr)
			}

			return nil
		})
		if txErr != nil {
			return "", "", txErr
		}

		slog.InfoContext(ctx, "user registered via google", "user_id", u.ID, "email", info.Email)
	}

	// 更新最后登录时间
	if updateErr := s.repo.UpdateLastLogin(ctx, u.ID); updateErr != nil {
		slog.WarnContext(ctx, "update last login failed", "error", updateErr)
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

func (s *serviceImpl) GetHistory(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*HistoryItem, int64, error) {
	return s.histRepo.ListByUser(ctx, userID, offset, limit)
}

func (s *serviceImpl) AddFavorite(ctx context.Context, fav *Favorite) error {
	fav.ID = uuid.New()
	fav.CreatedAt = time.Now()

	return s.favRepo.Add(ctx, fav)
}

func (s *serviceImpl) ListFavorites(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*Favorite, int64, error) {
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

func (s *serviceImpl) withTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.transactor == nil {
		return fn(ctx)
	}

	return s.transactor.Transaction(ctx, fn)
}
