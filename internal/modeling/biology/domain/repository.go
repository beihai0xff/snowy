package domain

import (
	"context"

	"github.com/google/uuid"
)

// RunRepository 生物建模运行记录持久化端口。
type RunRepository interface {
	Save(ctx context.Context, run *BiologyRun) error
	GetByID(ctx context.Context, id uuid.UUID) (*BiologyRun, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*BiologyRun, int64, error)
}

// BiologyRun 一次生物建模运行记录。
type BiologyRun struct {
	ID             uuid.UUID     `json:"id"`
	SessionID      uuid.UUID     `json:"session_id"`
	Question       string        `json:"question"`
	Model          *BiologyModel `json:"model"`
	ModelName      string        `json:"model_name"`
	LatencyMS      int           `json:"latency_ms"`
	Status         string        `json:"status"` // success / failed / fallback
	FallbackReason string        `json:"fallback_reason,omitempty"`
}
