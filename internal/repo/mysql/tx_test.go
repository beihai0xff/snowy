package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/user"
)

func TestTransactor_Transaction_CommitsAcrossRepositories(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	transactor := NewTransactor(db)
	userRepo := NewUserRepository(db)
	favoriteRepo := NewFavoriteRepository(db)

	now := time.Now()
	u := &user.User{
		ID:          uuid.New(),
		Phone:       "13800138001",
		Nickname:    "Tx User",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	fav := &user.Favorite{
		ID:         uuid.New(),
		UserID:     u.ID,
		TargetType: "physics",
		TargetID:   "run_tx_1",
		Title:      "事务收藏",
		CreatedAt:  now,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `users`").
		WithArgs(u.ID, u.GoogleID, u.Email, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO `favorites`").
		WithArgs(fav.ID, fav.UserID, fav.TargetType, fav.TargetID, fav.Title, fav.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := transactor.Transaction(context.Background(), func(ctx context.Context) error {
		if err := userRepo.Create(ctx, u); err != nil {
			return err
		}

		return favoriteRepo.Add(ctx, fav)
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactor_Transaction_RollsBackOnError(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	transactor := NewTransactor(db)
	userRepo := NewUserRepository(db)

	now := time.Now()
	u := &user.User{
		ID:          uuid.New(),
		Phone:       "13800138002",
		Nickname:    "Rollback User",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	expectedErr := errors.New("force rollback")

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `users`").
		WithArgs(u.ID, u.GoogleID, u.Email, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	err := transactor.Transaction(context.Background(), func(ctx context.Context) error {
		if err := userRepo.Create(ctx, u); err != nil {
			return err
		}

		return expectedErr
	})

	require.ErrorIs(t, err, expectedErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactor_Transaction_ReusesExistingTransaction(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	transactor := NewTransactor(db)
	userRepo := NewUserRepository(db)

	now := time.Now()
	u := &user.User{
		ID:          uuid.New(),
		Phone:       "13800138003",
		Nickname:    "Nested User",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `users`").
		WithArgs(u.ID, u.GoogleID, u.Email, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := transactor.Transaction(context.Background(), func(ctx context.Context) error {
		return transactor.Transaction(ctx, func(ctx context.Context) error {
			return userRepo.Create(ctx, u)
		})
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
