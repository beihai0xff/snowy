package monitoring

import (
	"context"
	"time"
)

// LLMRecordFilter filters persisted LLM call records for the dashboard.
type LLMRecordFilter struct {
	UserID    string
	Provider  string
	Model     string
	Operation string
	Since     time.Time
	Until     time.Time
	Limit     int
}

// LLMCallRecordStore persists and queries LLM call records.
type LLMCallRecordStore interface {
	Save(ctx context.Context, record LLMCallRecord) error
	List(ctx context.Context, filter LLMRecordFilter) ([]LLMCallRecord, error)
}
