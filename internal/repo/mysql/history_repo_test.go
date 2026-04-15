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

func TestHistoryRepo_Add_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewHistoryRepository(db)

	item := &user.HistoryItem{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		ActionType: "search",
		Query:      "牛顿第二定律",
		SessionID:  uuid.New(),
		CreatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO history_items").
		WithArgs(item.ID, item.UserID, item.ActionType, item.Query, item.SessionID, item.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Add(context.Background(), item)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHistoryRepo_ListByUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewHistoryRepository(db)
	userID := uuid.New()
	now := time.Now()
	sid := uuid.New()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM history_items WHERE user_id = \\?").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "user_id", "action_type", "query", "session_id", "created_at"}).
		AddRow(uuid.New(), userID, "physics", "平抛运动", sid, now)

	mock.ExpectQuery("SELECT .+ FROM history_items WHERE user_id = \\?").
		WithArgs(userID, 10, 0).
		WillReturnRows(rows)

	items, total, err := repo.ListByUser(context.Background(), userID, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)
	assert.Equal(t, "平抛运动", items[0].Query)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHistoryRepo_ListByUser_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewHistoryRepository(db)
	userID := uuid.New()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM history_items WHERE user_id = \\?").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	rows := sqlmock.NewRows([]string{"id", "user_id", "action_type", "query", "session_id", "created_at"})

	mock.ExpectQuery("SELECT .+ FROM history_items WHERE user_id = \\?").
		WithArgs(userID, 20, 0).
		WillReturnRows(rows)

	items, total, err := repo.ListByUser(context.Background(), userID, 0, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, items)
	assert.NoError(t, mock.ExpectationsWereMet())
}
