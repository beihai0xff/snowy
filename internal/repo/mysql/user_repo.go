package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/user"
)

// userRepo 实现 user.Repository 接口。
type userRepo struct {
	db *gorm.DB
}

// NewUserRepository 创建 User Repository。
func NewUserRepository(db *gorm.DB) user.Repository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	err := dbFromContext(ctx, r.db).Create(newUserRow(u)).Error
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row := &userRow{}

	err := dbFromContext(ctx, r.db).Where("id = ?", id).Take(row).Error
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return row.toDomain(), nil
}

func (r *userRepo) GetByPhone(ctx context.Context, phone string) (*user.User, error) {
	row := &userRow{}

	err := dbFromContext(ctx, r.db).Where("phone = ?", phone).Take(row).Error
	if err != nil {
		return nil, fmt.Errorf("get user by phone: %w", err)
	}

	return row.toDomain(), nil
}

func (r *userRepo) GetByGoogleID(ctx context.Context, googleID string) (*user.User, error) {
	row := &userRow{}

	err := dbFromContext(ctx, r.db).Where("google_id = ?", googleID).Take(row).Error
	if err != nil {
		return nil, fmt.Errorf("get user by google_id: %w", err)
	}

	return row.toDomain(), nil
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	err := dbFromContext(ctx, r.db).
		Model(&userRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_login_at": gorm.Expr("NOW(3)"),
			"updated_at":    gorm.Expr("NOW(3)"),
		}).Error
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}

	return nil
}
