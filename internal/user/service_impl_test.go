package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// ── Mock Repositories ────────────────────────────────────

type mockRepo struct {
	createFn          func(ctx context.Context, u *User) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*User, error)
	getByPhoneFn      func(ctx context.Context, phone string) (*User, error)
	updateLastLoginFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRepo) Create(ctx context.Context, u *User) error {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) GetByPhone(ctx context.Context, phone string) (*User, error) {
	if m.getByPhoneFn != nil {
		return m.getByPhoneFn(ctx, phone)
	}
	return nil, nil
}

func (m *mockRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(ctx, id)
	}
	return nil
}

type mockFavRepo struct {
	addFn        func(ctx context.Context, fav *Favorite) error
	removeFn     func(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	listByUserFn func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error)
}

func (m *mockFavRepo) Add(ctx context.Context, fav *Favorite) error {
	if m.addFn != nil {
		return m.addFn(ctx, fav)
	}
	return nil
}

func (m *mockFavRepo) Remove(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, id, userID)
	}
	return nil
}

func (m *mockFavRepo) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, offset, limit)
	}
	return nil, 0, nil
}

type mockHistRepo struct {
	addFn        func(ctx context.Context, item *HistoryItem) error
	listByUserFn func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error)
}

func (m *mockHistRepo) Add(ctx context.Context, item *HistoryItem) error {
	if m.addFn != nil {
		return m.addFn(ctx, item)
	}
	return nil
}

func (m *mockHistRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*HistoryItem, int64, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, offset, limit)
	}
	return nil, 0, nil
}

// ── Helper ───────────────────────────────────────────────

var testAuthCfg = config.AuthConfig{
	JWTSecret:       "test-secret-key-1234567890",
	AccessTokenTTL:  15 * time.Minute,
	RefreshTokenTTL: 7 * 24 * time.Hour,
}

func newTestService(repo *mockRepo, favRepo *mockFavRepo, histRepo *mockHistRepo) Service {
	return NewService(repo, favRepo, histRepo, testAuthCfg)
}

// ── Register Tests ───────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	var savedUser *User
	repo := &mockRepo{
		createFn: func(_ context.Context, u *User) error {
			savedUser = u
			return nil
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	u, err := svc.Register(context.Background(), "13800138000", "Alice")

	require.NoError(t, err)
	assert.Equal(t, "13800138000", u.Phone)
	assert.Equal(t, "Alice", u.Nickname)
	assert.Equal(t, RoleStudent, u.Role)
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.Equal(t, savedUser, u)
}

func TestRegister_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ *User) error {
			return errors.New("duplicate phone")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	u, err := svc.Register(context.Background(), "13800138000", "Alice")

	assert.Nil(t, u)
	assert.ErrorContains(t, err, "duplicate phone")
}

// ── Login Tests ──────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	testUser := &User{
		ID:    uuid.New(),
		Phone: "13800138000",
		Role:  RoleStudent,
	}
	repo := &mockRepo{
		getByPhoneFn: func(_ context.Context, phone string) (*User, error) {
			assert.Equal(t, "13800138000", phone)
			return testUser, nil
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	access, refresh, err := svc.Login(context.Background(), "13800138000", "1234")

	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)

	// 验证 access token claims
	token, err := jwt.Parse(access, func(t *jwt.Token) (any, error) {
		return []byte(testAuthCfg.JWTSecret), nil
	})
	require.NoError(t, err)
	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, testUser.ID.String(), claims["user_id"])
	assert.Equal(t, string(RoleStudent), claims["role"])
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getByPhoneFn: func(_ context.Context, _ string) (*User, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	_, _, err := svc.Login(context.Background(), "13800138000", "1234")

	assert.Error(t, err)
	assert.ErrorContains(t, err, "get user by phone")
}

func TestLogin_UpdateLastLoginError_NonFatal(t *testing.T) {
	testUser := &User{ID: uuid.New(), Phone: "13800138000", Role: RoleStudent}
	repo := &mockRepo{
		getByPhoneFn: func(_ context.Context, _ string) (*User, error) {
			return testUser, nil
		},
		updateLastLoginFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("db timeout")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	access, refresh, err := svc.Login(context.Background(), "13800138000", "1234")

	// UpdateLastLogin 失败不应阻止登录
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

// ── GetProfile Tests ─────────────────────────────────────

func TestGetProfile_Success(t *testing.T) {
	uid := uuid.New()
	expected := &User{ID: uid, Nickname: "Bob"}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*User, error) {
			assert.Equal(t, uid, id)
			return expected, nil
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{})

	u, err := svc.GetProfile(context.Background(), uid)

	require.NoError(t, err)
	assert.Equal(t, expected, u)
}

// ── AddFavorite Tests ────────────────────────────────────

func TestAddFavorite_SetsIDAndTimestamp(t *testing.T) {
	favRepo := &mockFavRepo{
		addFn: func(_ context.Context, fav *Favorite) error {
			assert.NotEqual(t, uuid.Nil, fav.ID)
			assert.False(t, fav.CreatedAt.IsZero())
			return nil
		},
	}
	svc := newTestService(&mockRepo{}, favRepo, &mockHistRepo{})

	fav := &Favorite{
		UserID:     uuid.New(),
		TargetType: "search",
		TargetID:   "doc-123",
		Title:      "Test",
	}
	err := svc.AddFavorite(context.Background(), fav)

	require.NoError(t, err)
}

// ── GetHistory Tests ─────────────────────────────────────

func TestGetHistory_Success(t *testing.T) {
	uid := uuid.New()
	items := []*HistoryItem{{ID: uuid.New(), UserID: uid, Query: "test"}}
	histRepo := &mockHistRepo{
		listByUserFn: func(_ context.Context, userID uuid.UUID, offset, limit int) ([]*HistoryItem, int64, error) {
			assert.Equal(t, uid, userID)
			assert.Equal(t, 0, offset)
			assert.Equal(t, 20, limit)
			return items, 1, nil
		},
	}
	svc := newTestService(&mockRepo{}, &mockFavRepo{}, histRepo)

	result, total, err := svc.GetHistory(context.Background(), uid, 0, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}

// ── ListFavorites Tests ──────────────────────────────────

func TestListFavorites_Success(t *testing.T) {
	uid := uuid.New()
	favs := []*Favorite{{ID: uuid.New(), UserID: uid}}
	favRepo := &mockFavRepo{
		listByUserFn: func(_ context.Context, userID uuid.UUID, offset, limit int) ([]*Favorite, int64, error) {
			return favs, 1, nil
		},
	}
	svc := newTestService(&mockRepo{}, favRepo, &mockHistRepo{})

	result, total, err := svc.ListFavorites(context.Background(), uid, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}
