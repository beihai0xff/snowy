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
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentSessionRepository(db)
	s := &agent.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Mode:      agent.ModeSearch,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO `agent_sessions`").
		WithArgs(s.ID, s.UserID, s.Mode, s.Status, sqlmock.AnyArg(), s.CreatedAt, s.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), s)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_GetByID(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentSessionRepository(db)
	sid := uuid.New()
	uid := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "mode", "status", "metadata", "created_at", "updated_at"}).
		AddRow(sid, uid, "search", "active", []byte(`{}`), now, now)

	mock.ExpectQuery("SELECT \\* FROM `agent_sessions` WHERE id = \\? LIMIT \\?").
		WithArgs(sid, 1).
		WillReturnRows(rows)

	s, err := repo.GetByID(context.Background(), sid)

	require.NoError(t, err)
	assert.Equal(t, sid, s.ID)
	assert.Equal(t, uid, s.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_UpdateStatus(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentSessionRepository(db)
	sid := uuid.New()

	mock.ExpectExec("UPDATE `agent_sessions` SET .* WHERE id = \\?").
		WithArgs("closed", sid).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateStatus(context.Background(), sid, "closed")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentSessionRepo_ListByUser(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentSessionRepository(db)
	uid := uuid.New()
	sid := uuid.New()
	now := time.Now()

	// COUNT query
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `agent_sessions` WHERE user_id = \\?").
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// SELECT query
	rows := sqlmock.NewRows([]string{"id", "user_id", "mode", "status", "metadata", "created_at", "updated_at"}).
		AddRow(sid, uid, "search", "active", []byte(`{}`), now, now)
	mock.ExpectQuery("SELECT \\* FROM `agent_sessions` WHERE user_id = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(uid, 10).
		WillReturnRows(rows)

	sessions, total, err := repo.ListByUser(context.Background(), uid, 0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, sessions, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── AgentMessageRepo Tests ───────────────────────────────

func TestAgentMessageRepo_Save(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentMessageRepository(db)
	msg := &agent.Message{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		Role:      "user",
		Content:   "hello",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO `agent_messages`").
		WithArgs(msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Save(context.Background(), msg)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentMessageRepo_ListBySession(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentMessageRepository(db)
	sid := uuid.New()
	mid := uuid.New()
	now := time.Now()

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `agent_messages` WHERE session_id = \\?").
		WithArgs(sid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "session_id", "role", "content", "created_at"}).
		AddRow(mid, sid, "user", "hello", now)
	mock.ExpectQuery("SELECT \\* FROM `agent_messages` WHERE session_id = \\? ORDER BY created_at ASC LIMIT \\?").
		WithArgs(sid, 20).
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
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

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

	mock.ExpectExec("INSERT INTO `agent_runs`").
		WithArgs(run.ID, run.SessionID, run.MessageID, run.Mode, run.ModelName, run.PromptVersion,
			run.InputTokens, run.OutputTokens, run.EstimatedCost, run.LatencyMS, run.Confidence,
			run.FallbackReason, run.Status, run.ErrorCode, run.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Save(context.Background(), run)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentRunRepo_GetByID(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentRunRepository(db)
	rid := uuid.New()
	sid := uuid.New()
	mid := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "session_id", "message_id", "mode", "model_name", "prompt_version", "input_tokens", "output_tokens", "estimated_cost", "latency_ms", "confidence", "fallback_reason", "status", "error_code", "created_at"}).
		AddRow(rid, sid, mid, "search", "gpt-4o", "v1", 100, 50, 0.01, 500, 0.9, "", "success", "", now)

	mock.ExpectQuery("SELECT \\* FROM `agent_runs` WHERE id = \\? LIMIT \\?").
		WithArgs(rid, 1).
		WillReturnRows(rows)

	run, err := repo.GetByID(context.Background(), rid)

	require.NoError(t, err)
	assert.Equal(t, rid, run.ID)
	assert.Equal(t, "gpt-4o", run.ModelName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAgentToolCallRepo_SaveAndListByRun(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()

	repo := NewAgentToolCallRepository(db)
	runID := uuid.New()
	callID := uuid.New()
	now := time.Now()

	tc := &agent.RunToolCall{
		ID:        callID,
		RunID:     runID,
		ToolName:  "SearchTool",
		Input:     map[string]any{"query": "牛顿第二定律"},
		Output:    map[string]any{"count": 1},
		LatencyMS: 20,
		Status:    "success",
		CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO `agent_tool_calls`").
		WithArgs(tc.ID, tc.RunID, tc.ToolName, sqlmock.AnyArg(), sqlmock.AnyArg(), tc.LatencyMS, tc.Status, tc.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Save(context.Background(), tc)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "run_id", "tool_name", "input", "output", "latency_ms", "status", "created_at"}).
		AddRow(callID, runID, "SearchTool", []byte(`{"query":"牛顿第二定律"}`), []byte(`{"count":1}`), 20, "success", now)

	mock.ExpectQuery("SELECT \\* FROM `agent_tool_calls` WHERE run_id = \\? ORDER BY created_at ASC").
		WithArgs(runID).
		WillReturnRows(rows)

	calls, err := repo.ListByRun(context.Background(), runID)
	require.NoError(t, err)
	assert.Len(t, calls, 1)
	assert.Equal(t, "SearchTool", calls[0].ToolName)
	assert.NoError(t, mock.ExpectationsWereMet())
}
