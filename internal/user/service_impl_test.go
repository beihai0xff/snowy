package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	irepo "github.com/beihai0xff/snowy/internal/repo"
)

// ── Mock Repositories ────────────────────────────────────

type mockRepo struct {
	createFn           func(ctx context.Context, u *User) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*User, error)
	getByPhoneFn       func(ctx context.Context, phone string) (*User, error)
	getByGoogleIDFn    func(ctx context.Context, googleID string) (*User, error)
	updateLastLoginFn  func(ctx context.Context, id uuid.UUID) error
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

func (m *mockRepo) GetByGoogleID(ctx context.Context, googleID string) (*User, error) {
	if m.getByGoogleIDFn != nil {
		return m.getByGoogleIDFn(ctx, googleID)
	}
	return nil, errors.New("not found")
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

type mockTransactor struct {
	transactionFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactor) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.transactionFn != nil {
		return m.transactionFn(ctx, fn)
	}

	return fn(ctx)
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

func newTestService(repo *mockRepo, favRepo *mockFavRepo, histRepo *mockHistRepo, transactor irepo.Transactor) Service {
	return NewService(repo, favRepo, histRepo, transactor, testAuthCfg)
}

// ── GoogleLogin Tests — New User (auto-register) ─────────

func TestGoogleLogin_NewUser_Success(t *testing.T) {
	var savedUser *User
	var savedHistory *HistoryItem
	repo := &mockRepo{
		getByGoogleIDFn: func(_ context.Context, _ string) (*User, error) {
			return nil, errors.New("not found")
		},
		createFn: func(_ context.Context, u *User) error {
			savedUser = u
			return nil
		},
	}
	histRepo := &mockHistRepo{
		addFn: func(_ context.Context, item *HistoryItem) error {
			savedHistory = item
			return nil
		},
	}
	transactorCalled := false
	svc := newTestService(repo, &mockFavRepo{}, histRepo, &mockTransactor{
		transactionFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			transactorCalled = true
			return fn(ctx)
		},
	})

	info := &GoogleUserInfo{
		GoogleID:  "google-123",
		Email:     "alice@gmail.com",
		Name:      "Alice",
		AvatarURL: "https://example.com/photo.jpg",
	}
	access, refresh, err := svc.GoogleLogin(context.Background(), info)

	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	assert.True(t, transactorCalled)
	require.NotNil(t, savedUser)
	assert.Equal(t, "google-123", savedUser.GoogleID)
	assert.Equal(t, "alice@gmail.com", savedUser.Email)
	assert.Equal(t, "Alice", savedUser.Nickname)
	assert.Equal(t, RoleStudent, savedUser.Role)
	assert.NotEqual(t, uuid.Nil, savedUser.ID)
	require.NotNil(t, savedHistory)
	assert.Equal(t, savedUser.ID, savedHistory.UserID)
	assert.Equal(t, "register", savedHistory.ActionType)
}

func TestGoogleLogin_NewUser_RepoError(t *testing.T) {
	repo := &mockRepo{
		getByGoogleIDFn: func(_ context.Context, _ string) (*User, error) {
			return nil, errors.New("not found")
		},
		createFn: func(_ context.Context, _ *User) error {
			return errors.New("duplicate google_id")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{}, &mockTransactor{})

	info := &GoogleUserInfo{GoogleID: "google-123", Email: "alice@gmail.com", Name: "Alice"}
	_, _, err := svc.GoogleLogin(context.Background(), info)

	assert.ErrorContains(t, err, "duplicate google_id")
}

func TestGoogleLogin_NewUser_HistoryError(t *testing.T) {
	repo := &mockRepo{
		getByGoogleIDFn: func(_ context.Context, _ string) (*User, error) {
			return nil, errors.New("not found")
		},
	}
	histRepo := &mockHistRepo{
		addFn: func(_ context.Context, _ *HistoryItem) error {
			return errors.New("history unavailable")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, histRepo, &mockTransactor{})

	info := &GoogleUserInfo{GoogleID: "google-123", Email: "alice@gmail.com", Name: "Alice"}
	_, _, err := svc.GoogleLogin(context.Background(), info)

	assert.ErrorContains(t, err, "add register history")
}

// ── GoogleLogin Tests — Existing User ────────────────────

func TestGoogleLogin_ExistingUser_Success(t *testing.T) {
	testUser := &User{
		ID:       uuid.New(),
		GoogleID: "google-456",
		Email:    "bob@gmail.com",
		Role:     RoleStudent,
	}
	repo := &mockRepo{
		getByGoogleIDFn: func(_ context.Context, googleID string) (*User, error) {
			assert.Equal(t, "google-456", googleID)
			return testUser, nil
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{}, nil)

	info := &GoogleUserInfo{GoogleID: "google-456", Email: "bob@gmail.com", Name: "Bob"}
	access, refresh, err := svc.GoogleLogin(context.Background(), info)

	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

func TestGoogleLogin_UpdateLastLoginError_NonFatal(t *testing.T) {
	testUser := &User{ID: uuid.New(), GoogleID: "google-789", Role: RoleStudent}
	repo := &mockRepo{
		getByGoogleIDFn: func(_ context.Context, _ string) (*User, error) {
			return testUser, nil
		},
		updateLastLoginFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("db timeout")
		},
	}
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{}, nil)

	info := &GoogleUserInfo{GoogleID: "google-789", Email: "test@gmail.com", Name: "Test"}
	access, refresh, err := svc.GoogleLogin(context.Background(), info)

	// UpdateLastLogin 失败不应阻止登录
	require.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
}

func TestGoogleLogin_NilInfo(t *testing.T) {
	svc := newTestService(&mockRepo{}, &mockFavRepo{}, &mockHistRepo{}, nil)

	_, _, err := svc.GoogleLogin(context.Background(), nil)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "google user info is required")
}

func TestGoogleLogin_EmptyGoogleID(t *testing.T) {
	svc := newTestService(&mockRepo{}, &mockFavRepo{}, &mockHistRepo{}, nil)

	info := &GoogleUserInfo{GoogleID: "", Email: "test@gmail.com", Name: "Test"}
	_, _, err := svc.GoogleLogin(context.Background(), info)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "google user info is required")
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
	svc := newTestService(repo, &mockFavRepo{}, &mockHistRepo{}, nil)

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
	svc := newTestService(&mockRepo{}, favRepo, &mockHistRepo{}, nil)

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
	svc := newTestService(&mockRepo{}, &mockFavRepo{}, histRepo, nil)

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
	svc := newTestService(&mockRepo{}, favRepo, &mockHistRepo{}, nil)

	result, total, err := svc.ListFavorites(context.Background(), uid, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}
