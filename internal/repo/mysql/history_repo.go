package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/beihai0xff/snowy/internal/user"
	"github.com/google/uuid"
)

// historyRepo 实现 user.HistoryRepository 接口。
type historyRepo struct {
	db *sql.DB
}

// NewHistoryRepository 创建 History Repository。
func NewHistoryRepository(db *sql.DB) user.HistoryRepository {
	return &historyRepo{db: db}
}

func (r *historyRepo) Add(ctx context.Context, item *user.HistoryItem) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO history_items (id, user_id, action_type, query, session_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.UserID, item.ActionType, item.Query, item.SessionID, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert history item: %w", err)
	}

	return nil
}

func (r *historyRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.HistoryItem, int64, error) {
	return listByUserRows(ctx, r.db, userID, offset, limit,
		`SELECT COUNT(*) FROM history_items WHERE user_id = ?`,
		`SELECT id, user_id, action_type, query, session_id, created_at
		 FROM history_items WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		"history items", "history items",
		func(rows *sql.Rows) (*user.HistoryItem, error) {
			h := &user.HistoryItem{}
			if err := rows.Scan(&h.ID, &h.UserID, &h.ActionType, &h.Query, &h.SessionID, &h.CreatedAt); err != nil {
				return nil, fmt.Errorf("scan history item: %w", err)
			}

			return h, nil
		},
	)
}
