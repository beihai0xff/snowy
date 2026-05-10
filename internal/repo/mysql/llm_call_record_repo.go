package mysql

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/monitoring"
)

type llmCallRecordRepo struct{ db *gorm.DB }

func NewLLMCallRecordRepository(db *gorm.DB) monitoring.LLMCallRecordStore {
	return &llmCallRecordRepo{db: db}
}

func (r *llmCallRecordRepo) Save(ctx context.Context, record monitoring.LLMCallRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("llm call record id is empty")
	}

	if err := dbFromContext(ctx, r.db).Create(newLLMCallRecordRow(record)).Error; err != nil {
		return fmt.Errorf("insert llm call record: %w", err)
	}

	return nil
}

func (r *llmCallRecordRepo) List(
	ctx context.Context,
	filter monitoring.LLMRecordFilter,
) ([]monitoring.LLMCallRecord, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	query := dbFromContext(ctx, r.db).Model(&llmCallRecordRow{})
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}
	if filter.Model != "" {
		query = query.Where("model = ?", filter.Model)
	}
	if filter.Operation != "" {
		query = query.Where("operation = ?", filter.Operation)
	}
	if !filter.Since.IsZero() {
		query = query.Where("finished_at >= ?", filter.Since)
	}
	if !filter.Until.IsZero() {
		query = query.Where("finished_at <= ?", filter.Until)
	}

	rows := []llmCallRecordRow{}
	if err := query.Order("finished_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list llm call records: %w", err)
	}

	records := make([]monitoring.LLMCallRecord, 0, len(rows))
	for i := range rows {
		records = append(records, rows[i].toMonitoring())
	}

	return records, nil
}
