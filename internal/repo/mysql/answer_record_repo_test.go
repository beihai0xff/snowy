package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
)

func TestAnswerRecordRepository_Save(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAnswerRecordRepository(db)
	record := &searchdomain.AnswerRecord{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		Query:         "牛顿第二定律",
		AnswerSummary: "结论：F=ma",
		KnowledgeTags: []string{"physics"},
		Citations:     []searchdomain.Citation{{DocID: "doc-1", SourceType: "textbook", Snippet: "F=ma", Score: 0.9}},
		Confidence:    0.88,
		Source:        "llm_direct",
		ModelName:     "model-a",
		Metadata:      map[string]any{"intent": "explain"},
		CreatedAt:     time.Now(),
	}

	mock.ExpectExec("INSERT INTO `answer_records`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Save(context.Background(), record))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAnswerRecordRepository_ListByUser(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAnswerRecordRepository(db)
	userID := uuid.New()
	recordID := uuid.New()
	now := time.Now()

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `answer_records` WHERE user_id = \\?").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "user_id", "session_id", "query", "answer_summary", "knowledge_tags", "citations", "confidence", "source", "model_name", "metadata", "created_at"}).
		AddRow(recordID, userID, nil, "平抛", "结论", `["physics"]`, `[{"doc_id":"doc-1","source_type":"runtime","snippet":"s","score":0.8}]`, 0.8, "retrieval", "", `{"intent":"explain"}`, now)

	mock.ExpectQuery("SELECT \\* FROM `answer_records` WHERE user_id = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(userID, 20).
		WillReturnRows(rows)

	items, total, err := repo.ListByUser(context.Background(), userID.String(), 0, 20)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, recordID, items[0].ID)
	require.Equal(t, "平抛", items[0].Query)
	require.Equal(t, []string{"physics"}, items[0].KnowledgeTags)
	require.Len(t, items[0].Citations, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
