package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/beihai0xff/snowy/internal/user"
	"github.com/google/uuid"
)

// favoriteRepo 实现 user.FavoriteRepository 接口。
type favoriteRepo struct {
	db *sql.DB
}

// NewFavoriteRepository 创建 Favorite Repository。
func NewFavoriteRepository(db *sql.DB) user.FavoriteRepository {
	return &favoriteRepo{db: db}
}

func (r *favoriteRepo) Add(ctx context.Context, fav *user.Favorite) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO favorites (id, user_id, target_type, target_id, title, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fav.ID, fav.UserID, fav.TargetType, fav.TargetID, fav.Title, fav.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert favorite: %w", err)
	}

	return nil
}

func (r *favoriteRepo) Remove(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM favorites WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.New("favorite not found")
	}

	return nil
}

func (r *favoriteRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.Favorite, int64, error) {
	return listByUserRows(ctx, r.db, userID, offset, limit,
		`SELECT COUNT(*) FROM favorites WHERE user_id = ?`,
		`SELECT id, user_id, target_type, target_id, title, created_at
		 FROM favorites WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		"favorites", "favorites",
		func(rows *sql.Rows) (*user.Favorite, error) {
			f := &user.Favorite{}
			if err := rows.Scan(&f.ID, &f.UserID, &f.TargetType, &f.TargetID, &f.Title, &f.CreatedAt); err != nil {
				return nil, fmt.Errorf("scan favorite: %w", err)
			}

			return f, nil
		},
	)
}
