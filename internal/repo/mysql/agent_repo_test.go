package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/agent"
)

// ── AgentSessionRepo Tests ───────────────────────────────

func TestAgentSessionRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentSessionRepository(db)
	s := &agent.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Mode:      agent.ModeSearch,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO agent_sessions").
		WithArgs(s.ID, s.UserID, s.Mode, s.Status, sqlmock.AnyArg(), s.CreatedAt, s.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(context.Background(), s)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentSessionRepository(db)
	sid := uuid.New()
	uid := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "mode", "status", "metadata", "created_at", "updated_at"}).
		AddRow(sid, uid, "search", "active", []byte(`{}`), now, now)

	mock.ExpectQuery("SELECT .+ FROM agent_sessions WHERE id = \\?").
		WithArgs(sid).
		WillReturnRows(rows)

	s, err := repo.GetByID(context.Background(), sid)

	require.NoError(t, err)
	assert.Equal(t, sid, s.ID)
	assert.Equal(t, uid, s.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentSessionRepository(db)
	sid := uuid.New()

	mock.ExpectExec("UPDATE agent_sessions SET status").
		WithArgs("closed", sid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateStatus(context.Background(), sid, "closed")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_ListByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentSessionRepository(db)
	uid := uuid.New()
	sid := uuid.New()
	now := time.Now()

	// COUNT query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM agent_sessions").
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// SELECT query
	rows := sqlmock.NewRows([]string{"id", "user_id", "mode", "status", "metadata", "created_at", "updated_at"}).
		AddRow(sid, uid, "search", "active", []byte(`{}`), now, now)
	mock.ExpectQuery("SELECT .+ FROM agent_sessions WHERE user_id = \\?").
		WithArgs(uid, 10, 0).
		WillReturnRows(rows)

	sessions, total, err := repo.ListByUser(context.Background(), uid, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, sessions, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── AgentMessageRepo Tests ───────────────────────────────

func TestAgentMessageRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentMessageRepository(db)
	msg := &agent.Message{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		Role:      "user",
		Content:   "hello",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO agent_messages").
		WithArgs(msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Save(context.Background(), msg)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentMessageRepo_ListBySession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentMessageRepository(db)
	sid := uuid.New()
	mid := uuid.New()
	now := time.Now()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM agent_messages").
		WithArgs(sid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "session_id", "role", "content", "created_at"}).
		AddRow(mid, sid, "user", "hello", now)
	mock.ExpectQuery("SELECT .+ FROM agent_messages WHERE session_id = \\?").
		WithArgs(sid, 20, 0).
		WillReturnRows(rows)

	msgs, total, err := repo.ListBySession(context.Background(), sid, 0, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "hello", msgs[0].Content)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── AgentRunRepo Tests ───────────────────────────────────

func TestAgentRunRepo_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAgentRunRepository(db)
	run := &agent.Run{
		ID:            uuid.New(),
		SessionID:     uuid.New(),
		MessageID:     uuid.New(),
		Mode:          agent.ModeSearch,
		ModelName:     "gpt-4o",
		PromptVersion: "v1",
		InputTokens:   100,
		OutputTokens:  50,
		EstimatedCost: 0.01,
		LatencyMS:     500,
		Confidence:    0.9,
		Status:        "success",
		CreatedAt:     time.Now(),
	}

	mock.ExpectExec("INSERT INTO agent_runs").
		WithArgs(run.ID, run.SessionID, run.MessageID, run.Mode, run.ModelName, run.PromptVersion,
			run.InputTokens, run.OutputTokens, run.EstimatedCost, run.LatencyMS, run.Confidence,
			run.FallbackReason, run.Status, run.ErrorCode, run.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Save(context.Background(), run)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
