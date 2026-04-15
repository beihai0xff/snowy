package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type userRowScanner[T any] func(*sql.Rows) (*T, error)

func listByUserRows[T any](
	ctx context.Context,
	db *sql.DB,
	userID uuid.UUID,
	offset, limit int,
	countQuery, listQuery string,
	countLabel, listLabel string,
	scan userRowScanner[T],
) ([]*T, int64, error) {
	var total int64

	err := db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", countLabel, err)
	}

	rows, err := db.QueryContext(ctx, listQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list %s: %w", listLabel, err)
	}
	defer rows.Close()

	items := make([]*T, 0, limit)

	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate %s: %w", listLabel, err)
	}

	return items, total, nil
}
