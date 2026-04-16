package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRowMapper[R any, T any] func(*R) (*T, error)

func listByUserRows[R any, T any](
	ctx context.Context,
	db *gorm.DB,
	model any,
	userID uuid.UUID,
	offset, limit int,
	order string,
	countLabel, listLabel string,
	mapRow userRowMapper[R, T],
) ([]*T, int64, error) {
	var total int64

	err := dbFromContext(ctx, db).Model(model).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", countLabel, err)
	}

	rows := make([]R, 0, limit)

	err = dbFromContext(ctx, db).
		Model(model).
		Where("user_id = ?", userID).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list %s: %w", listLabel, err)
	}

	items := make([]*T, 0, limit)

	for i := range rows {
		item, mapErr := mapRow(&rows[i])
		if mapErr != nil {
			return nil, 0, mapErr
		}

		items = append(items, item)
	}

	return items, total, nil
}
