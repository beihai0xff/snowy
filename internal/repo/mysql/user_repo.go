package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/user"
)

// userRepo 实现 user.Repository 接口。
type userRepo struct {
	db *sql.DB
}

// NewUserRepository 创建 User Repository。
func NewUserRepository(db *sql.DB) user.Repository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Phone, u.Nickname, u.Role, u.AvatarURL, u.LastLoginAt, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u := &user.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Phone, &u.Nickname, &u.Role, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByPhone(ctx context.Context, phone string) (*user.User, error) {
	u := &user.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at
		 FROM users WHERE phone = ?`, phone,
	).Scan(&u.ID, &u.Phone, &u.Nickname, &u.Role, &u.AvatarURL, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return u, nil
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}
