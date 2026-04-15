package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/user"
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

func (r *historyRepo) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.HistoryItem, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM history_items WHERE user_id = ?`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count history items: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, action_type, query, session_id, created_at
		 FROM history_items WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list history items: %w", err)
	}
	defer rows.Close()

	var items []*user.HistoryItem
	for rows.Next() {
		h := &user.HistoryItem{}
		if err := rows.Scan(&h.ID, &h.UserID, &h.ActionType, &h.Query, &h.SessionID, &h.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan history item: %w", err)
		}
		items = append(items, h)
	}
	return items, total, nil
}
