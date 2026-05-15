//nolint:cyclop,funcorder // User service keeps auth and reaction workflows grouped by product flow.
package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	irepo "github.com/beihai0xff/snowy/internal/repo"
)

// serviceImpl 用户域应用服务实现。
type serviceImpl struct {
	repo         Repository
	favRepo      FavoriteRepository
	histRepo     HistoryRepository
	reactionRepo ReactionRepository
	transactor   irepo.Transactor
	authCfg      config.AuthConfig
}

// NewService 创建用户域应用服务。
func NewService(
	repo Repository,
	favRepo FavoriteRepository,
	histRepo HistoryRepository,
	transactor irepo.Transactor,
	authCfg config.AuthConfig,
	extraRepos ...ReactionRepository,
) Service {
	var reactionRepo ReactionRepository
	if len(extraRepos) > 0 {
		reactionRepo = extraRepos[0]
	}

	return &serviceImpl{
		repo:         repo,
		favRepo:      favRepo,
		histRepo:     histRepo,
		reactionRepo: reactionRepo,
		transactor:   transactor,
		authCfg:      authCfg,
	}
}

// GoogleLogin 通过 Google 用户信息查找或创建用户，返回 JWT token 对。
func (s *serviceImpl) GoogleLogin(ctx context.Context, info *GoogleUserInfo) (string, string, error) {
	if info == nil || info.GoogleID == "" {
		return "", "", errors.New("google user info is required")
	}

	u, err := s.findOrCreateGoogleUser(ctx, info)
	if err != nil {
		return "", "", err
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

func (s *serviceImpl) EnsureAnonymousUser(ctx context.Context) (*User, error) {
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	u, err := s.repo.GetByID(ctx, uid)
	if err == nil {
		return u, nil
	}

	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	now := time.Now()

	u = &User{
		ID:          uid,
		Nickname:    "Anonymous",
		Role:        RoleStudent,
		AvatarURL:   "",
		LastLoginAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if createErr := s.repo.Create(ctx, u); createErr != nil {
		// 并发创建时可能已存在，二次读取兜底。
		if existing, getErr := s.repo.GetByID(ctx, uid); getErr == nil {
			return existing, nil
		}

		return nil, createErr
	}

	return u, nil
}

func (s *serviceImpl) GetHistory(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*HistoryItem, int64, error) {
	return s.histRepo.ListByUser(ctx, userID, offset, limit)
}

func (s *serviceImpl) AddHistory(ctx context.Context, item *HistoryItem) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}

	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	return s.histRepo.Add(ctx, item)
}

func (s *serviceImpl) AddFavorite(ctx context.Context, fav *Favorite) error {
	if fav == nil {
		return errors.New("favorite is nil")
	}

	fav.ID = uuid.New()
	fav.CreatedAt = time.Now()
	fav.TargetType = strings.TrimSpace(fav.TargetType)
	fav.TargetID = strings.TrimSpace(fav.TargetID)
	fav.Title = strings.TrimSpace(fav.Title)
	fav.MetadataJSON = sanitizeFavoriteMetadata(fav.MetadataJSON)

	return s.favRepo.Add(ctx, fav)
}

func sanitizeFavoriteMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	cleaned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" || isSensitiveMetadataKey(key) {
			continue
		}

		cleaned[key] = sanitizeMetadataValue(value)
	}

	if len(cleaned) == 0 {
		return nil
	}

	return cleaned
}

func isSensitiveMetadataKey(key string) bool {
	lower := strings.ToLower(key)
	for _, part := range []string{"api_key", "apikey", "authorization", "password", "secret", "token"} {
		if strings.Contains(lower, part) {
			return true
		}
	}

	return false
}

func sanitizeMetadataValue(value any) any {
	switch v := value.(type) {
	case string:
		return truncateString(v, 2000)
	case []any:
		out := make([]any, 0, min(len(v), 20))
		for i, item := range v {
			if i >= 20 {
				break
			}

			out = append(out, sanitizeMetadataValue(item))
		}

		return out
	case map[string]any:
		return sanitizeFavoriteMetadata(v)
	default:
		return v
	}
}

func truncateString(text string, limit int) string {
	text = strings.TrimSpace(text)

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return string(runes[:limit]) + "…"
}

func (s *serviceImpl) ListFavorites(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*Favorite, int64, error) {
	return s.favRepo.ListByUser(ctx, userID, offset, limit)
}

// findOrCreateGoogleUser 根据 GoogleID 查找已有用户，不存在则自动注册。
func (s *serviceImpl) findOrCreateGoogleUser(ctx context.Context, info *GoogleUserInfo) (*User, error) {
	u, err := s.repo.GetByGoogleID(ctx, info.GoogleID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("lookup google user: %w", err)
	}

	if !errors.Is(err, ErrUserNotFound) {
		return u, nil
	}

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
		return nil, txErr
	}

	slog.InfoContext(ctx, "user registered via google", "user_id", u.ID, "email", info.Email)

	return u, nil
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

func (s *serviceImpl) EmailRegister(
	ctx context.Context,
	email, password, nickname string,
) (string, string, *User, error) {
	email = normalizeEmail(email)
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", nil, errors.New("valid email is required")
	}

	if len(password) < 8 {
		return "", "", nil, errors.New("password must be at least 8 characters")
	}

	if existing, err := s.repo.GetByEmail(ctx, email); err == nil && existing != nil {
		return "", "", nil, errors.New("email already registered")
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return "", "", nil, fmt.Errorf("lookup email user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()

	if strings.TrimSpace(nickname) == "" {
		nickname = email
	}

	u := &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Nickname:     strings.TrimSpace(nickname),
		Role:         RoleStudent,
		LastLoginAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	history := &HistoryItem{ID: uuid.New(), UserID: u.ID, ActionType: "register", Query: "邮箱账号注册", CreatedAt: now}

	err = s.withTransaction(ctx, func(txCtx context.Context) error {
		if createErr := s.repo.Create(txCtx, u); createErr != nil {
			return fmt.Errorf("create user: %w", createErr)
		}

		if s.histRepo != nil {
			if histErr := s.histRepo.Add(txCtx, history); histErr != nil {
				return fmt.Errorf("add register history: %w", histErr)
			}
		}

		return nil
	})
	if err != nil {
		return "", "", nil, err
	}

	access, refresh, err := s.tokenPair(u)
	if err != nil {
		return "", "", nil, err
	}

	return access, refresh, u, nil
}

func (s *serviceImpl) EmailLogin(ctx context.Context, email, password string) (string, string, *User, error) {
	email = normalizeEmail(email)

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", "", nil, errors.New("invalid email or password")
		}

		return "", "", nil, fmt.Errorf("lookup email user: %w", err)
	}

	if strings.TrimSpace(u.PasswordHash) == "" {
		return "", "", nil, errors.New("email account has no password login enabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, errors.New("invalid email or password")
	}

	if updateErr := s.repo.UpdateLastLogin(ctx, u.ID); updateErr != nil {
		slog.WarnContext(ctx, "update last login failed", "error", updateErr)
	}

	access, refresh, err := s.tokenPair(u)
	if err != nil {
		return "", "", nil, err
	}

	return access, refresh, u, nil
}

func (s *serviceImpl) SetReaction(ctx context.Context, reaction *Reaction) error {
	if s.reactionRepo == nil {
		return errors.New("reaction repository is nil")
	}

	if reaction == nil {
		return errors.New("reaction is nil")
	}

	if reaction.UserID == uuid.Nil {
		return errors.New("reaction user_id is required")
	}

	reaction.TargetType = strings.TrimSpace(reaction.TargetType)

	reaction.TargetID = strings.TrimSpace(reaction.TargetID)
	if reaction.TargetType == "" || reaction.TargetID == "" {
		return errors.New("reaction target is required")
	}

	if reaction.ReactionType != ReactionLike && reaction.ReactionType != ReactionDislike {
		return errors.New("reaction_type must be like or dislike")
	}

	if reaction.Visibility == "" {
		reaction.Visibility = ReactionVisibilityPublic
	}

	if reaction.Visibility != ReactionVisibilityPublic && reaction.Visibility != ReactionVisibilityPrivate {
		return errors.New("visibility must be public or private")
	}

	now := time.Now()

	if reaction.ID == uuid.Nil {
		reaction.ID = uuid.New()
	}

	if reaction.CreatedAt.IsZero() {
		reaction.CreatedAt = now
	}

	reaction.UpdatedAt = now

	return s.reactionRepo.Upsert(ctx, reaction)
}

func (s *serviceImpl) DeleteReaction(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error {
	if s.reactionRepo == nil {
		return errors.New("reaction repository is nil")
	}

	return s.reactionRepo.Delete(ctx, userID, strings.TrimSpace(targetType), strings.TrimSpace(targetID))
}

func (s *serviceImpl) ListReactions(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*Reaction, int64, error) {
	if s.reactionRepo == nil {
		return nil, 0, errors.New("reaction repository is nil")
	}

	return s.reactionRepo.ListByUser(ctx, userID, offset, limit)
}

func (s *serviceImpl) ReactionSummary(
	ctx context.Context,
	userID uuid.UUID,
	targetType string,
	targetID string,
	includeUsers bool,
) (*ReactionSummary, error) {
	if s.reactionRepo == nil {
		return nil, errors.New("reaction repository is nil")
	}

	return s.reactionRepo.Summary(ctx, userID, strings.TrimSpace(targetType), strings.TrimSpace(targetID), includeUsers)
}

func (s *serviceImpl) tokenPair(u *User) (string, string, error) {
	access, err := s.generateToken(u, s.authCfg.AccessTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refresh, err := s.generateToken(u, s.authCfg.RefreshTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return access, refresh, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *serviceImpl) withTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.transactor == nil {
		return fn(ctx)
	}

	return s.transactor.Transaction(ctx, fn)
}
