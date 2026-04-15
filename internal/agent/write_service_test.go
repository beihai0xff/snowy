package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	irepo "github.com/beihai0xff/snowy/internal/repo"
)

type mockTransactor struct {
	transactionFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactor) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.transactionFn != nil {
		return m.transactionFn(ctx, fn)
	}

	return fn(ctx)
}

type mockSessionRepository struct {
	createFn  func(ctx context.Context, session *Session) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*Session, error)
}

func (m *mockSessionRepository) Create(ctx context.Context, session *Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockSessionRepository) UpdateStatus(context.Context, uuid.UUID, string) error { return nil }
func (m *mockSessionRepository) ListByUser(context.Context, uuid.UUID, int, int) ([]*Session, int64, error) {
	return nil, 0, nil
}

type mockMessageRepository struct {
	saveFn func(ctx context.Context, msg *Message) error
}

func (m *mockMessageRepository) Save(ctx context.Context, msg *Message) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, msg)
	}
	return nil
}
func (m *mockMessageRepository) ListBySession(context.Context, uuid.UUID, int, int) ([]*Message, int64, error) {
	return nil, 0, nil
}

type mockRunRepository struct {
	saveFn func(ctx context.Context, run *Run) error
}

func (m *mockRunRepository) Save(ctx context.Context, run *Run) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, run)
	}
	return nil
}
func (m *mockRunRepository) GetByID(context.Context, uuid.UUID) (*Run, error) { return nil, nil }

type mockToolCallRepository struct {
	saveFn func(ctx context.Context, tc *RunToolCall) error
}

func (m *mockToolCallRepository) Save(ctx context.Context, tc *RunToolCall) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, tc)
	}
	return nil
}
func (m *mockToolCallRepository) ListByRun(context.Context, uuid.UUID) ([]*RunToolCall, error) {
	return nil, nil
}

func TestWriteService_CreateSession_Success(t *testing.T) {
	var saved *Session
	svc := NewWriteService(
		&mockTransactor{},
		&mockSessionRepository{createFn: func(_ context.Context, session *Session) error {
			saved = session
			return nil
		}},
		&mockMessageRepository{},
		&mockRunRepository{},
		&mockToolCallRepository{},
	)

	session, err := svc.CreateSession(context.Background(), &CreateSessionInput{
		UserID: uuid.New(),
		Mode:   ModeSearch,
	})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, saved, session)
	assert.Equal(t, "active", session.Status)
}

func TestWriteService_PersistConversation_CreatesAllRecordsAtomically(t *testing.T) {
	var (
		savedSession   *Session
		savedMessages  []*Message
		savedRun       *Run
		savedToolCalls []*RunToolCall
	)
	ctx := context.WithValue(context.Background(), struct{}{}, "trace")
	transactorCalled := false
	transactor := &mockTransactor{
		transactionFn: func(txCtx context.Context, fn func(ctx context.Context) error) error {
			transactorCalled = true
			assert.Equal(t, ctx, txCtx)
			return fn(txCtx)
		},
	}

	svc := NewWriteService(
		transactor,
		&mockSessionRepository{createFn: func(_ context.Context, session *Session) error {
			savedSession = session
			return nil
		}},
		&mockMessageRepository{saveFn: func(_ context.Context, msg *Message) error {
			savedMessages = append(savedMessages, msg)
			return nil
		}},
		&mockRunRepository{saveFn: func(_ context.Context, run *Run) error {
			savedRun = run
			return nil
		}},
		&mockToolCallRepository{saveFn: func(_ context.Context, tc *RunToolCall) error {
			savedToolCalls = append(savedToolCalls, tc)
			return nil
		}},
	)

	userID := uuid.New()
	result, err := svc.PersistConversation(ctx, &PersistConversationInput{
		UserID:  userID,
		Mode:    ModeSearch,
		Message: "牛顿第二定律怎么用？",
		Filters: Filters{Subject: "physics", Grade: "high-school"},
		Response: &ChatResponse{
			Mode:       ModeSearch,
			Answer:     "F=ma",
			Confidence: 0.95,
			ToolCalls:  []ToolCall{{Tool: "SearchTool", Status: "success"}},
		},
	})

	require.NoError(t, err)
	assert.True(t, transactorCalled)
	require.NotNil(t, result)
	require.NotNil(t, savedSession)
	assert.Equal(t, userID, savedSession.UserID)
	assert.Equal(t, "physics", savedSession.Metadata["subject"])
	require.Len(t, savedMessages, 2)
	assert.Equal(t, "user", savedMessages[0].Role)
	assert.Equal(t, "assistant", savedMessages[1].Role)
	require.NotNil(t, savedRun)
	assert.Equal(t, savedMessages[0].ID, savedRun.MessageID)
	require.Len(t, savedToolCalls, 1)
	assert.Equal(t, savedRun.ID, savedToolCalls[0].RunID)
	assert.Equal(t, savedSession.ID, result.Session.ID)
}

func TestWriteService_PersistConversation_RollsUpError(t *testing.T) {
	expectedErr := errors.New("tool call failed")
	svc := NewWriteService(
		&mockTransactor{transactionFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		}},
		&mockSessionRepository{createFn: func(_ context.Context, _ *Session) error { return nil }},
		&mockMessageRepository{saveFn: func(_ context.Context, _ *Message) error { return nil }},
		&mockRunRepository{saveFn: func(_ context.Context, _ *Run) error { return nil }},
		&mockToolCallRepository{saveFn: func(_ context.Context, _ *RunToolCall) error { return expectedErr }},
	)

	result, err := svc.PersistConversation(context.Background(), &PersistConversationInput{
		UserID:  uuid.New(),
		Mode:    ModeSearch,
		Message: "test",
		Response: &ChatResponse{
			Answer:    "ok",
			ToolCalls: []ToolCall{{Tool: "SearchTool", Status: "failed"}},
		},
	})

	assert.Nil(t, result)
	assert.ErrorContains(t, err, "save agent tool call")
}

var _ irepo.Transactor = (*mockTransactor)(nil)
