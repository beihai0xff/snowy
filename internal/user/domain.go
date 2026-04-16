// Package user 定义用户域的领域模型与接口。
// 有界上下文：User — 负责登录态、历史记录、收藏、学习行为。
// 参考技术方案 §9.6。
package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrUserNotFound 用户不存在。
var ErrUserNotFound = errors.New("user not found")

// Role 用户角色。
type Role string

const (
	RoleStudent Role = "student"
	RoleTeacher Role = "teacher"
	RoleAdmin   Role = "admin"
)

// User 用户实体。
type User struct {
	ID          uuid.UUID `json:"id"`
	GoogleID    string    `json:"google_id,omitempty"`
	Email       string    `json:"email,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Nickname    string    `json:"nickname"`
	Role        Role      `json:"role"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Favorite 收藏条目。
type Favorite struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	TargetType string    `json:"target_type"` // search / physics / biology
	TargetID   string    `json:"target_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

// HistoryItem 历史记录条目。
type HistoryItem struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	ActionType string    `json:"action_type"` // search / physics / biology
	Query      string    `json:"query"`
	SessionID  uuid.UUID `json:"session_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
