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
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

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

	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), u)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	uid := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "phone", "nickname", "role", "avatar_url", "last_login_at", "created_at", "updated_at"}).
		AddRow(uid, "13800138000", "Alice", "student", "", now, now, now)

	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\?").
		WithArgs(uid).
		WillReturnRows(rows)

	u, err := repo.GetByID(context.Background(), uid)

	require.NoError(t, err)
	assert.Equal(t, uid, u.ID)
	assert.Equal(t, "13800138000", u.Phone)
	assert.Equal(t, "Alice", u.Nickname)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_GetByPhone_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	uid := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "phone", "nickname", "role", "avatar_url", "last_login_at", "created_at", "updated_at"}).
		AddRow(uid, "13800138000", "Bob", "student", "", now, now, now)

	mock.ExpectQuery("SELECT .+ FROM users WHERE phone = \\?").
		WithArgs("13800138000").
		WillReturnRows(rows)

	u, err := repo.GetByPhone(context.Background(), "13800138000")

	require.NoError(t, err)
	assert.Equal(t, "Bob", u.Nickname)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepo_UpdateLastLogin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	uid := uuid.New()

	mock.ExpectExec("UPDATE users SET last_login_at").
		WithArgs(uid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateLastLogin(context.Background(), uid)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
