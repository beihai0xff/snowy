package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/agent"
	biologydomain "github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	physicsdomain "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock Services ────────────────────────────────────────

type mockUserService struct {
	getProfileFn      func(ctx context.Context, userID uuid.UUID) (*user.User, error)
	getHistoryFn      func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.HistoryItem, int64, error)
	addHistoryFn      func(ctx context.Context, item *user.HistoryItem) error
	addFavoriteFn     func(ctx context.Context, fav *user.Favorite) error
	listFavoritesFn   func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.Favorite, int64, error)
	emailRegisterFn   func(ctx context.Context, email, password, nickname string) (string, string, *user.User, error)
	emailLoginFn      func(ctx context.Context, email, password string) (string, string, *user.User, error)
	setReactionFn     func(ctx context.Context, reaction *user.Reaction) error
	deleteReactionFn  func(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error
	listReactionsFn   func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.Reaction, int64, error)
	reactionSummaryFn func(ctx context.Context, userID uuid.UUID, targetType string, targetID string, includeUsers bool) (*user.ReactionSummary, error)
}

func (m *mockUserService) GoogleLogin(_ context.Context, _ *user.GoogleUserInfo) (string, string, error) {
	return "", "", nil
}

func (m *mockUserService) EmailRegister(ctx context.Context, email, password, nickname string) (string, string, *user.User, error) {
	if m.emailRegisterFn != nil {
		return m.emailRegisterFn(ctx, email, password, nickname)
	}

	return "access", "refresh", &user.User{ID: uuid.New(), Email: email, Nickname: nickname}, nil
}

func (m *mockUserService) EmailLogin(ctx context.Context, email, password string) (string, string, *user.User, error) {
	if m.emailLoginFn != nil {
		return m.emailLoginFn(ctx, email, password)
	}

	return "access", "refresh", &user.User{ID: uuid.New(), Email: email, Nickname: email}, nil
}

func (m *mockUserService) SetReaction(ctx context.Context, reaction *user.Reaction) error {
	if m.setReactionFn != nil {
		return m.setReactionFn(ctx, reaction)
	}

	return nil
}

func (m *mockUserService) DeleteReaction(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error {
	if m.deleteReactionFn != nil {
		return m.deleteReactionFn(ctx, userID, targetType, targetID)
	}

	return nil
}

func (m *mockUserService) ListReactions(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.Reaction, int64, error) {
	if m.listReactionsFn != nil {
		return m.listReactionsFn(ctx, userID, offset, limit)
	}

	return nil, 0, nil
}

func (m *mockUserService) ReactionSummary(ctx context.Context, userID uuid.UUID, targetType string, targetID string, includeUsers bool) (*user.ReactionSummary, error) {
	if m.reactionSummaryFn != nil {
		return m.reactionSummaryFn(ctx, userID, targetType, targetID, includeUsers)
	}

	return &user.ReactionSummary{TargetType: targetType, TargetID: targetID}, nil
}

func (m *mockUserService) GetProfile(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return m.getProfileFn(ctx, userID)
}

func (m *mockUserService) EnsureAnonymousUser(ctx context.Context) (*user.User, error) {
	if m.getProfileFn != nil {
		return m.getProfileFn(ctx, uuid.MustParse(common.DefaultUserID))
	}
	return &user.User{ID: uuid.MustParse(common.DefaultUserID), Nickname: "Anonymous"}, nil
}

func (m *mockUserService) GetHistory(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.HistoryItem, int64, error) {
	return m.getHistoryFn(ctx, userID, offset, limit)
}

func (m *mockUserService) AddHistory(ctx context.Context, item *user.HistoryItem) error {
	if m.addHistoryFn == nil {
		return nil
	}

	return m.addHistoryFn(ctx, item)
}

func (m *mockUserService) AddFavorite(ctx context.Context, fav *user.Favorite) error {
	return m.addFavoriteFn(ctx, fav)
}

func (m *mockUserService) ListFavorites(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.Favorite, int64, error) {
	return m.listFavoritesFn(ctx, userID, offset, limit)
}

type mockAgentService struct {
	chatFn       func(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error)
	chatStreamFn func(ctx context.Context, req *agent.ChatRequest, events chan<- agent.SSEEvent) error
}

func (m *mockAgentService) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	return m.chatFn(ctx, req)
}

func (m *mockAgentService) ChatStream(ctx context.Context, req *agent.ChatRequest, events chan<- agent.SSEEvent) error {
	return m.chatStreamFn(ctx, req, events)
}

type mockSearchService struct {
	queryFn func(ctx context.Context, q *search.Query) (*search.Response, error)
}

func (m *mockSearchService) Query(ctx context.Context, q *search.Query) (*search.Response, error) {
	return m.queryFn(ctx, q)
}

type mockPhysicsService struct {
	analyzeFn func(ctx context.Context, question string, sessionContext string) (*physicsdomain.PhysicsModel, error)
}

func (m *mockPhysicsService) Analyze(
	ctx context.Context,
	question string,
	sessionContext string,
) (*physicsdomain.PhysicsModel, error) {
	return m.analyzeFn(ctx, question, sessionContext)
}

func (m *mockPhysicsService) Simulate(
	context.Context,
	physicsdomain.ModelType,
	map[string]float64,
) (*physicsdomain.ComputeResult, error) {
	return nil, nil
}

func (m *mockPhysicsService) GenerateRender(
	context.Context,
	*physicsdomain.SceneSpec,
	string,
) (*physicsdomain.RenderArtifact, error) {
	return nil, nil
}

type mockBiologyService struct {
	analyzeFn func(ctx context.Context, question string, sessionContext string) (*biologydomain.BiologyModel, error)
}

func (m *mockBiologyService) Analyze(
	ctx context.Context,
	question string,
	sessionContext string,
) (*biologydomain.BiologyModel, error) {
	return m.analyzeFn(ctx, question, sessionContext)
}

type mockAgentWriteService struct {
	createSessionFn       func(ctx context.Context, input *agent.CreateSessionInput) (*agent.Session, error)
	persistConversationFn func(ctx context.Context, input *agent.PersistConversationInput) (*agent.PersistConversationResult, error)
}

func (m *mockAgentWriteService) CreateSession(ctx context.Context, input *agent.CreateSessionInput) (*agent.Session, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, input)
	}

	return nil, nil
}

func (m *mockAgentWriteService) PersistConversation(ctx context.Context, input *agent.PersistConversationInput) (*agent.PersistConversationResult, error) {
	if m.persistConversationFn != nil {
		return m.persistConversationFn(ctx, input)
	}

	return nil, nil
}

type mockSessionRepo struct {
	createFn     func(ctx context.Context, s *agent.Session) error
	getByIDFn    func(ctx context.Context, id uuid.UUID) (*agent.Session, error)
	updateFn     func(ctx context.Context, id uuid.UUID, status string) error
	listByUserFn func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*agent.Session, int64, error)
}

func (m *mockSessionRepo) Create(ctx context.Context, s *agent.Session) error {
	return m.createFn(ctx, s)
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*agent.Session, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.updateFn(ctx, id, status)
}

func (m *mockSessionRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*agent.Session, int64, error) {
	return m.listByUserFn(ctx, userID, offset, limit)
}

type mockMessageRepo struct {
	saveFn          func(ctx context.Context, msg *agent.Message) error
	listBySessionFn func(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error)
}

func (m *mockMessageRepo) Save(ctx context.Context, msg *agent.Message) error {
	return m.saveFn(ctx, msg)
}

func (m *mockMessageRepo) ListBySession(
	ctx context.Context,
	sessionID uuid.UUID,
	offset, limit int,
) ([]*agent.Message, int64, error) {
	return m.listBySessionFn(ctx, sessionID, offset, limit)
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeCh chan bool
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeCh:          make(chan bool, 1),
	}
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
	return r.closeCh
}

// ── helpers ──────────────────────────────────────────────

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func getRequest(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ── UserHandler Tests ────────────────────────────────────

func TestUserHandler_GetProfile_Success(t *testing.T) {
	uid := uuid.MustParse(common.DefaultUserID)
	svc := &mockUserService{
		getProfileFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, Nickname: "Bob"}, nil
		},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.GET("/profile", func(c *gin.Context) {
		ctx := common.WithUserID(c.Request.Context(), uid.String())
		c.Request = c.Request.WithContext(ctx)
		handler.GetProfile(c)
	})

	w := getRequest(r, "/profile")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetProfile_DefaultUser(t *testing.T) {
	svc := &mockUserService{
		getProfileFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			assert.Equal(t, uuid.MustParse(common.DefaultUserID), id)
			return &user.User{ID: id, Nickname: "Anonymous"}, nil
		},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.GET("/profile", handler.GetProfile) // no user ID in context

	w := getRequest(r, "/profile")

	assert.Equal(t, http.StatusOK, w.Code)
}

// ── AgentHandler Tests ───────────────────────────────────

func TestAgentHandler_Chat_Success(t *testing.T) {
	svc := &mockAgentService{
		chatFn: func(_ context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
			return &agent.ChatResponse{
				Mode:       agent.ModeSearch,
				Answer:     "The answer is 42",
				Confidence: 0.95,
			}, nil
		},
	}
	handler := NewAgentHandler(svc, nil, nil, nil)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	w := postJSON(r, "/chat", map[string]string{"message": "What is gravity?"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_Chat_BadJSON(t *testing.T) {
	handler := NewAgentHandler(&mockAgentService{}, nil, nil, nil)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	// missing required "message" field
	w := postJSON(r, "/chat", map[string]string{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_Chat_ServiceError(t *testing.T) {
	svc := &mockAgentService{
		chatFn: func(_ context.Context, _ *agent.ChatRequest) (*agent.ChatResponse, error) {
			return nil, errors.New("llm timeout")
		},
	}
	handler := NewAgentHandler(svc, nil, nil, nil)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	w := postJSON(r, "/chat", map[string]string{"message": "hello"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── SearchHandler Tests ──────────────────────────────────

func TestSearchHandler_Query_Success(t *testing.T) {
	svc := &mockSearchService{
		queryFn: func(_ context.Context, q *search.Query) (*search.Response, error) {
			return &search.Response{
				Answer:     "Newton's second law",
				Confidence: 0.88,
			}, nil
		},
	}
	handler := NewSearchHandler(svc)

	r := gin.New()
	r.POST("/query", handler.Query)

	w := postJSON(r, "/query", map[string]string{"query": "Newton's law"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestSearchHandler_Query_BadJSON(t *testing.T) {
	handler := NewSearchHandler(&mockSearchService{})

	r := gin.New()
	r.POST("/query", handler.Query)

	w := postJSON(r, "/query", map[string]string{}) // missing required "query"

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchHandler_Query_ServiceError(t *testing.T) {
	svc := &mockSearchService{
		queryFn: func(_ context.Context, _ *search.Query) (*search.Response, error) {
			return nil, errors.New("opensearch down")
		},
	}
	handler := NewSearchHandler(svc)

	r := gin.New()
	r.POST("/query", handler.Query)

	w := postJSON(r, "/query", map[string]string{"query": "test"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSearchHandler_Query_RecordsHistory(t *testing.T) {
	var recorded *user.HistoryItem
	svc := &mockSearchService{
		queryFn: func(_ context.Context, _ *search.Query) (*search.Response, error) {
			return &search.Response{Answer: "ok", Confidence: 0.8}, nil
		},
	}
	userSvc := &mockUserService{
		addHistoryFn: func(_ context.Context, item *user.HistoryItem) error {
			recorded = item
			return nil
		},
	}
	handler := NewSearchHandler(svc, userSvc)

	r := gin.New()
	r.POST("/query", handler.Query)

	w := postJSON(r, "/query", map[string]string{"query": "Newton"})

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, recorded)
	assert.Equal(t, "search", recorded.ActionType)
	assert.Equal(t, "Newton", recorded.Query)
}

func TestPhysicsHandler_Analyze_RecordsHistory(t *testing.T) {
	var recorded *user.HistoryItem
	physicsSvc := &mockPhysicsService{
		analyzeFn: func(_ context.Context, question string, _ string) (*physicsdomain.PhysicsModel, error) {
			return &physicsdomain.PhysicsModel{
				ModelType:     physicsdomain.ModelProjectileMotion,
				ResultSummary: question,
			}, nil
		},
	}
	userSvc := &mockUserService{
		addHistoryFn: func(_ context.Context, item *user.HistoryItem) error {
			recorded = item
			return nil
		},
	}
	handler := NewPhysicsHandler(physicsSvc, userSvc)

	r := gin.New()
	r.POST("/physics", handler.Analyze)

	w := postJSON(r, "/physics", map[string]string{"question": "平抛运动"})

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, recorded)
	assert.Equal(t, "physics", recorded.ActionType)
	assert.Equal(t, "平抛运动", recorded.Query)
}

func TestBiologyHandler_Analyze_RecordsHistory(t *testing.T) {
	var recorded *user.HistoryItem
	biologySvc := &mockBiologyService{
		analyzeFn: func(_ context.Context, question string, _ string) (*biologydomain.BiologyModel, error) {
			return &biologydomain.BiologyModel{Topic: question, ResultSummary: "ok"}, nil
		},
	}
	userSvc := &mockUserService{
		addHistoryFn: func(_ context.Context, item *user.HistoryItem) error {
			recorded = item
			return nil
		},
	}
	handler := NewBiologyHandler(biologySvc, userSvc)

	r := gin.New()
	r.POST("/biology", handler.Analyze)

	w := postJSON(r, "/biology", map[string]string{"question": "光合作用"})

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, recorded)
	assert.Equal(t, "biology", recorded.ActionType)
	assert.Equal(t, "光合作用", recorded.Query)
}

// ── AgentHandler Session Tests ───────────────────────────

func TestAgentHandler_CreateSession_Success(t *testing.T) {
	writeSvc := &mockAgentWriteService{
		createSessionFn: func(_ context.Context, input *agent.CreateSessionInput) (*agent.Session, error) {
			return &agent.Session{
				ID:        uuid.New(),
				UserID:    input.UserID,
				Mode:      input.Mode,
				Status:    "active",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	handler := NewAgentHandler(nil, writeSvc, nil, nil)

	r := gin.New()
	r.POST("/sessions", func(c *gin.Context) {
		ctx := common.WithUserID(c.Request.Context(), uuid.New().String())
		c.Request = c.Request.WithContext(ctx)
		handler.CreateSession(c)
	})

	w := postJSON(r, "/sessions", map[string]string{"mode": "search"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_CreateSession_BadMode(t *testing.T) {
	handler := NewAgentHandler(nil, nil, nil, nil)

	r := gin.New()
	r.POST("/sessions", handler.CreateSession)

	w := postJSON(r, "/sessions", map[string]string{"mode": "invalid"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_GetSession_Success(t *testing.T) {
	sid := uuid.New()
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*agent.Session, error) {
			return &agent.Session{
				ID:     id,
				Mode:   agent.ModeSearch,
				Status: "active",
			}, nil
		},
	}
	handler := NewAgentHandler(nil, nil, sessionRepo, nil)

	r := gin.New()
	r.GET("/sessions/:id", handler.GetSession)

	w := getRequest(r, "/sessions/"+sid.String())

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAgentHandler_GetSession_InvalidID(t *testing.T) {
	handler := NewAgentHandler(nil, nil, nil, nil)

	r := gin.New()
	r.GET("/sessions/:id", handler.GetSession)

	w := getRequest(r, "/sessions/not-a-uuid")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_GetSession_NotFound(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*agent.Session, error) {
			return nil, errors.New("not found")
		},
	}
	handler := NewAgentHandler(nil, nil, sessionRepo, nil)

	r := gin.New()
	r.GET("/sessions/:id", handler.GetSession)

	w := getRequest(r, "/sessions/"+uuid.New().String())

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAgentHandler_ListMessages_Success(t *testing.T) {
	sid := uuid.New()
	messageRepo := &mockMessageRepo{
		listBySessionFn: func(_ context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error) {
			return []*agent.Message{
				{ID: uuid.New(), SessionID: sessionID, Role: "user", Content: "hello"},
			}, 1, nil
		},
	}
	handler := NewAgentHandler(nil, nil, nil, messageRepo)

	r := gin.New()
	r.GET("/sessions/:id/messages", handler.ListMessages)

	w := getRequest(r, "/sessions/"+sid.String()+"/messages")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_ListMessages_InvalidID(t *testing.T) {
	handler := NewAgentHandler(nil, nil, nil, nil)

	r := gin.New()
	r.GET("/sessions/:id/messages", handler.ListMessages)

	w := getRequest(r, "/sessions/not-a-uuid/messages")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentHandler_Chat_PersistsConversation(t *testing.T) {
	userID := uuid.New()
	persistCalled := false
	svc := &mockAgentService{
		chatFn: func(_ context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
			return &agent.ChatResponse{Mode: agent.ModeSearch, Answer: "F=ma", Confidence: 0.9}, nil
		},
	}
	writeSvc := &mockAgentWriteService{
		persistConversationFn: func(_ context.Context, input *agent.PersistConversationInput) (*agent.PersistConversationResult, error) {
			persistCalled = true
			assert.Equal(t, userID, input.UserID)
			assert.Equal(t, "牛顿第二定律", input.Message)
			return &agent.PersistConversationResult{Session: &agent.Session{ID: uuid.New()}}, nil
		},
	}
	handler := NewAgentHandler(svc, writeSvc, nil, nil)

	r := gin.New()
	r.POST("/chat", func(c *gin.Context) {
		ctx := common.WithUserID(c.Request.Context(), userID.String())
		c.Request = c.Request.WithContext(ctx)
		handler.Chat(c)
	})

	w := postJSON(r, "/chat", map[string]string{"message": "牛顿第二定律", "mode": "search"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, persistCalled)
	assert.NotEmpty(t, w.Header().Get("X-Session-ID"))
}

func TestAgentHandler_ChatStream_PersistsAggregatedConversation(t *testing.T) {
	userID := uuid.New()
	persistCalled := false
	svc := &mockAgentService{
		chatStreamFn: func(_ context.Context, _ *agent.ChatRequest, events chan<- agent.SSEEvent) error {
			events <- agent.SSEEvent{Event: agent.SSEEventContent, Data: "F="}
			events <- agent.SSEEvent{Event: agent.SSEEventContent, Data: map[string]any{"content": "ma"}}
			events <- agent.SSEEvent{Event: agent.SSEEventToolCall, Data: map[string]any{"tool": "SearchTool", "status": "success"}}
			events <- agent.SSEEvent{Event: agent.SSEEventDone, Data: map[string]any{"mode": "search", "confidence": 0.91}}
			return nil
		},
	}
	writeSvc := &mockAgentWriteService{
		persistConversationFn: func(_ context.Context, input *agent.PersistConversationInput) (*agent.PersistConversationResult, error) {
			persistCalled = true
			assert.Equal(t, userID, input.UserID)
			require.NotNil(t, input.Response)
			assert.Equal(t, "F=ma", input.Response.Answer)
			assert.Equal(t, 0.91, input.Response.Confidence)
			require.Len(t, input.Response.ToolCalls, 1)
			assert.Equal(t, "SearchTool", input.Response.ToolCalls[0].Tool)
			return &agent.PersistConversationResult{Session: &agent.Session{ID: uuid.New()}}, nil
		},
	}
	handler := NewAgentHandler(svc, writeSvc, nil, nil)

	r := gin.New()
	r.POST("/chat", func(c *gin.Context) {
		ctx := common.WithUserID(c.Request.Context(), userID.String())
		c.Request = c.Request.WithContext(ctx)
		handler.Chat(c)
	})

	b, err := json.Marshal(map[string]string{"message": "牛顿第二定律", "mode": "search"})
	require.NoError(t, err)
	w := newCloseNotifyRecorder()
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, persistCalled)
	assert.Contains(t, w.Body.String(), "event:content")
	assert.Contains(t, w.Body.String(), "event:done")
}

func TestAgentHandler_ChatStream_DoesNotPersistWithoutDone(t *testing.T) {
	persistCalled := false
	svc := &mockAgentService{
		chatStreamFn: func(_ context.Context, _ *agent.ChatRequest, events chan<- agent.SSEEvent) error {
			events <- agent.SSEEvent{Event: agent.SSEEventContent, Data: "partial"}
			return nil
		},
	}
	writeSvc := &mockAgentWriteService{
		persistConversationFn: func(_ context.Context, _ *agent.PersistConversationInput) (*agent.PersistConversationResult, error) {
			persistCalled = true
			return nil, nil
		},
	}
	handler := NewAgentHandler(svc, writeSvc, nil, nil)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	b, err := json.Marshal(map[string]string{"message": "partial", "mode": "search"})
	require.NoError(t, err)
	w := newCloseNotifyRecorder()
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, persistCalled)
}
