package domain

import (
	"context"

	"github.com/google/uuid"
)

// RunRepository 物理建模运行记录持久化端口。
type RunRepository interface {
	Save(ctx context.Context, run *PhysicsRun) error
	GetByID(ctx context.Context, id uuid.UUID) (*PhysicsRun, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*PhysicsRun, int64, error)
}

// PhysicsRun 一次物理建模运行记录。
type PhysicsRun struct {
	ID             uuid.UUID     `json:"id"`
	SessionID      uuid.UUID     `json:"session_id"`
	Question       string        `json:"question"`
	Model          *PhysicsModel `json:"model"`
	ModelName      string        `json:"model_name"`
	LatencyMS      int           `json:"latency_ms"`
	Status         string        `json:"status"` // success / failed / fallback
	FallbackReason string        `json:"fallback_reason,omitempty"`
}
