package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
)

type answerRecordRepo struct{ db *gorm.DB }

func NewAnswerRecordRepository(db *gorm.DB) searchdomain.AnswerRecordRepository {
	return &answerRecordRepo{db: db}
}

func (r *answerRecordRepo) Save(ctx context.Context, record *searchdomain.AnswerRecord) error {
	if record == nil {
		return errors.New("answer record is nil")
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	if err := dbFromContext(ctx, r.db).Create(newAnswerRecordRow(record)).Error; err != nil {
		return fmt.Errorf("insert answer record: %w", err)
	}

	return nil
}

func (r *answerRecordRepo) GetByID(ctx context.Context, id string) (*searchdomain.AnswerRecord, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse answer record id: %w", err)
	}

	row := &answerRecordRow{}
	if err := dbFromContext(ctx, r.db).Where("id = ?", uid).First(row).Error; err != nil {
		return nil, fmt.Errorf("get answer record: %w", err)
	}

	return row.toDomain(), nil
}

func (r *answerRecordRepo) ListByUser(
	ctx context.Context,
	userID string,
	offset, limit int,
) ([]*searchdomain.AnswerRecord, int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("parse answer record user id: %w", err)
	}

	return listByUserRows[answerRecordRow](ctx, r.db, &answerRecordRow{}, uid, offset, limit,
		"created_at DESC",
		"answer records", "answer records",
		func(row *answerRecordRow) (*searchdomain.AnswerRecord, error) { return row.toDomain(), nil },
	)
}

func (r *answerRecordRepo) ListByQuery(
	ctx context.Context, query string, offset, limit int,
) ([]*searchdomain.AnswerRecord, int64, error) {
	var total int64
	gdb := dbFromContext(ctx, r.db)

	if err := gdb.Model(&answerRecordRow{}).Where("query = ?", query).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count answer records by query: %w", err)
	}

	rows := make([]answerRecordRow, 0, limit)
	if err := gdb.Model(&answerRecordRow{}).Where("query = ?", query).Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list answer records by query: %w", err)
	}

	items := make([]*searchdomain.AnswerRecord, 0, len(rows))
	for i := range rows {
		items = append(items, rows[i].toDomain())
	}

	return items, total, nil
}
