package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/user"
)

// historyRepo 实现 user.HistoryRepository 接口。
type historyRepo struct {
	db *gorm.DB
}

// NewHistoryRepository 创建 History Repository。
func NewHistoryRepository(db *gorm.DB) user.HistoryRepository {
	return &historyRepo{db: db}
}

func (r *historyRepo) Add(ctx context.Context, item *user.HistoryItem) error {
	err := dbFromContext(ctx, r.db).Create(newHistoryRow(item)).Error
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
	return listByUserRows[historyRow](ctx, r.db, &historyRow{}, userID, offset, limit,
		"created_at DESC",
		"history items", "history items",
		func(row *historyRow) (*user.HistoryItem, error) {
			return row.toDomain(), nil
		},
	)
}
