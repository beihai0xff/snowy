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

func TestFavoriteRepo_Add_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFavoriteRepository(db)

	fav := &user.Favorite{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		TargetType: "physics",
		TargetID:   "run_123",
		Title:      "平抛运动分析",
		CreatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO favorites").
		WithArgs(fav.ID, fav.UserID, fav.TargetType, fav.TargetID, fav.Title, fav.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Add(context.Background(), fav)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFavoriteRepo_Remove_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFavoriteRepository(db)
	favID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM favorites WHERE id = \\? AND user_id = \\?").
		WithArgs(favID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Remove(context.Background(), favID, userID)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFavoriteRepo_Remove_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFavoriteRepository(db)
	favID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM favorites WHERE id = \\? AND user_id = \\?").
		WithArgs(favID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Remove(context.Background(), favID, userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "favorite not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFavoriteRepo_ListByUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFavoriteRepository(db)
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM favorites WHERE user_id = \\?").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	rows := sqlmock.NewRows([]string{"id", "user_id", "target_type", "target_id", "title", "created_at"}).
		AddRow(uuid.New(), userID, "physics", "run_1", "测试1", now).
		AddRow(uuid.New(), userID, "biology", "run_2", "测试2", now)

	mock.ExpectQuery("SELECT .+ FROM favorites WHERE user_id = \\?").
		WithArgs(userID, 20, 0).
		WillReturnRows(rows)

	favs, total, err := repo.ListByUser(context.Background(), userID, 0, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, favs, 2)
	assert.Equal(t, "测试1", favs[0].Title)
	assert.NoError(t, mock.ExpectationsWereMet())
}
