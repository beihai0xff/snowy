package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/user"
)

func TestUserRepo_Create_Success(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewUserRepository(db)

	u := &user.User{
		ID:          uuid.New(),
		Phone:       "13800138000",
		Nickname:    "Alice",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO `users`").
		WithArgs(u.ID, u.GoogleID, u.Email, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), u)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetByID_Success(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewUserRepository(db)

	uid := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "google_id", "email", "phone", "nickname", "role", "avatar_url", "last_login_at", "created_at", "updated_at",
	}).AddRow(uid, "", "", "13800138000", "Alice", "student", "", now, now, now)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE id = \\? LIMIT \\?").
		WithArgs(uid, 1).
		WillReturnRows(rows)

	u, err := repo.GetByID(context.Background(), uid)

	require.NoError(t, err)
	assert.Equal(t, uid, u.ID)
	assert.Equal(t, "13800138000", u.Phone)
	assert.Equal(t, "Alice", u.Nickname)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetByPhone_Success(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewUserRepository(db)

	uid := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "google_id", "email", "phone", "nickname", "role", "avatar_url", "last_login_at", "created_at", "updated_at",
	}).AddRow(uid, "", "", "13800138000", "Bob", "student", "", now, now, now)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE phone = \\? LIMIT \\?").
		WithArgs("13800138000", 1).
		WillReturnRows(rows)

	u, err := repo.GetByPhone(context.Background(), "13800138000")

	require.NoError(t, err)
	assert.Equal(t, "Bob", u.Nickname)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetByGoogleID_Success(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewUserRepository(db)

	uid := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "google_id", "email", "phone", "nickname", "role", "avatar_url", "last_login_at", "created_at", "updated_at",
	}).AddRow(uid, "google-abc", "test@gmail.com", "", "Charlie", "student", "", now, now, now)

	mock.ExpectQuery("SELECT \\* FROM `users` WHERE google_id = \\? LIMIT \\?").
		WithArgs("google-abc", 1).
		WillReturnRows(rows)

	u, err := repo.GetByGoogleID(context.Background(), "google-abc")

	require.NoError(t, err)
	assert.Equal(t, "Charlie", u.Nickname)
	assert.Equal(t, "google-abc", u.GoogleID)
	assert.Equal(t, "test@gmail.com", u.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_UpdateLastLogin_Success(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewUserRepository(db)
	uid := uuid.New()

	mock.ExpectExec("UPDATE `users` SET .* WHERE id = \\?").
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastLogin(context.Background(), uid)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
