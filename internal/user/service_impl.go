package user

import (
	"context"
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
	verifier   VerificationCodeChecker
}

// NewService 创建用户域应用服务。
func NewService(
	repo Repository,
	favRepo FavoriteRepository,
	histRepo HistoryRepository,
	transactor irepo.Transactor,
	authCfg config.AuthConfig,
) Service {
	return NewServiceWithVerifier(repo, favRepo, histRepo, transactor, authCfg, NewNoopVerificationCodeChecker())
}

// NewServiceWithVerifier 创建带验证码校验器的用户域应用服务。
func NewServiceWithVerifier(
	repo Repository,
	favRepo FavoriteRepository,
	histRepo HistoryRepository,
	transactor irepo.Transactor,
	authCfg config.AuthConfig,
	verifier VerificationCodeChecker,
) Service {
	if verifier == nil {
		verifier = NewNoopVerificationCodeChecker()
	}
	return &serviceImpl{
		repo:       repo,
		favRepo:    favRepo,
		histRepo:   histRepo,
		transactor: transactor,
		authCfg:    authCfg,
		verifier:   verifier,
	}
}

func (s *serviceImpl) Register(ctx context.Context, phone, nickname string) (*User, error) {
	u := &User{
		ID:          uuid.New(),
		Phone:       phone,
		Nickname:    nickname,
		Role:        RoleStudent,
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	history := &HistoryItem{
		ID:         uuid.New(),
		UserID:     u.ID,
		ActionType: "register",
		Query:      "用户注册",
		CreatedAt:  u.CreatedAt,
	}

	err := s.withTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, u); err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		if err := s.histRepo.Add(txCtx, history); err != nil {
			return fmt.Errorf("add register history: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "user registered", "user_id", u.ID, "phone", phone)

	return u, nil
}

func (s *serviceImpl) Login(ctx context.Context, phone, code string) (string, string, error) {
	if err := s.verifier.Verify(ctx, phone, code); err != nil {
		return "", "", fmt.Errorf("verify login code: %w", err)
	}
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
