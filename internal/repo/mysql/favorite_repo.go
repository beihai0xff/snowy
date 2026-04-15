package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/user"
)

// favoriteRepo 实现 user.FavoriteRepository 接口。
type favoriteRepo struct {
	db *gorm.DB
}

// NewFavoriteRepository 创建 Favorite Repository。
func NewFavoriteRepository(db *gorm.DB) user.FavoriteRepository {
	return &favoriteRepo{db: db}
}

func (r *favoriteRepo) Add(ctx context.Context, fav *user.Favorite) error {
	err := dbFromContext(ctx, r.db).Create(newFavoriteRow(fav)).Error
	if err != nil {
		return fmt.Errorf("insert favorite: %w", err)
	}

	return nil
}

func (r *favoriteRepo) Remove(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result := dbFromContext(ctx, r.db).Where("id = ? AND user_id = ?", id, userID).Delete(&favoriteRow{})
	err := result.Error
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}

	if result.RowsAffected == 0 {
		return errors.New("favorite not found")
	}

	return nil
}

func (r *favoriteRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.Favorite, int64, error) {
	return listByUserRows[favoriteRow](ctx, r.db, &favoriteRow{}, userID, offset, limit,
		"created_at DESC",
		"favorites", "favorites",
		func(row *favoriteRow) (*user.Favorite, error) {
			return row.toDomain(), nil
		},
	)
}
