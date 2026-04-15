package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock Services ────────────────────────────────────────

type mockUserService struct {
	registerFn      func(ctx context.Context, phone, nickname string) (*user.User, error)
	loginFn         func(ctx context.Context, phone, code string) (string, string, error)
	getProfileFn    func(ctx context.Context, userID uuid.UUID) (*user.User, error)
	getHistoryFn    func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.HistoryItem, int64, error)
	addFavoriteFn   func(ctx context.Context, fav *user.Favorite) error
	listFavoritesFn func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.Favorite, int64, error)
}

func (m *mockUserService) Register(ctx context.Context, phone, nickname string) (*user.User, error) {
	return m.registerFn(ctx, phone, nickname)
}
func (m *mockUserService) Login(ctx context.Context, phone, code string) (string, string, error) {
	return m.loginFn(ctx, phone, code)
}
func (m *mockUserService) GetProfile(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return m.getProfileFn(ctx, userID)
}
func (m *mockUserService) GetHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.HistoryItem, int64, error) {
	return m.getHistoryFn(ctx, userID, offset, limit)
}
func (m *mockUserService) AddFavorite(ctx context.Context, fav *user.Favorite) error {
	return m.addFavoriteFn(ctx, fav)
}
func (m *mockUserService) ListFavorites(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*user.Favorite, int64, error) {
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
func (m *mockSessionRepo) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*agent.Session, int64, error) {
	return m.listByUserFn(ctx, userID, offset, limit)
}

type mockMessageRepo struct {
	saveFn          func(ctx context.Context, msg *agent.Message) error
	listBySessionFn func(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error)
}

func (m *mockMessageRepo) Save(ctx context.Context, msg *agent.Message) error {
	return m.saveFn(ctx, msg)
}
func (m *mockMessageRepo) ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error) {
	return m.listBySessionFn(ctx, sessionID, offset, limit)
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

func TestUserHandler_Login_Success(t *testing.T) {
	svc := &mockUserService{
		loginFn: func(_ context.Context, phone, code string) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.POST("/login", handler.Login)

	w := postJSON(r, "/login", map[string]string{"phone": "13800138000", "code": "1234"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestUserHandler_Login_BadJSON(t *testing.T) {
	handler := NewUserHandler(&mockUserService{})

	r := gin.New()
	r.POST("/login", handler.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_Login_ServiceError(t *testing.T) {
	svc := &mockUserService{
		loginFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("user not found")
		},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.POST("/login", handler.Login)

	w := postJSON(r, "/login", map[string]string{"phone": "13800138000", "code": "1234"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_Register_Success(t *testing.T) {
	svc := &mockUserService{
		registerFn: func(_ context.Context, phone, nickname string) (*user.User, error) {
			return &user.User{ID: uuid.New(), Phone: phone, Nickname: nickname}, nil
		},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.POST("/register", handler.Register)

	w := postJSON(r, "/register", map[string]string{"phone": "13800138000", "nickname": "Alice"})

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUserHandler_Register_MissingField(t *testing.T) {
	handler := NewUserHandler(&mockUserService{})

	r := gin.New()
	r.POST("/register", handler.Register)

	w := postJSON(r, "/register", map[string]string{"phone": "13800138000"}) // missing nickname

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetProfile_Success(t *testing.T) {
	uid := uuid.New()
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

func TestUserHandler_GetProfile_NoUserID(t *testing.T) {
	handler := NewUserHandler(&mockUserService{})

	r := gin.New()
	r.GET("/profile", handler.GetProfile)

	w := getRequest(r, "/profile")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
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
	handler := NewAgentHandler(svc, nil, nil)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	w := postJSON(r, "/chat", map[string]string{"message": "What is gravity?"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_Chat_BadJSON(t *testing.T) {
	handler := NewAgentHandler(&mockAgentService{}, nil, nil)

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
	handler := NewAgentHandler(svc, nil, nil)

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

// ── AgentHandler Session Tests ───────────────────────────

func TestAgentHandler_CreateSession_Success(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		createFn: func(_ context.Context, s *agent.Session) error {
			return nil
		},
	}
	handler := NewAgentHandler(nil, sessionRepo, nil)

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
	handler := NewAgentHandler(nil, nil, nil)

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
	handler := NewAgentHandler(nil, sessionRepo, nil)

	r := gin.New()
	r.GET("/sessions/:id", handler.GetSession)

	w := getRequest(r, "/sessions/"+sid.String())

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAgentHandler_GetSession_InvalidID(t *testing.T) {
	handler := NewAgentHandler(nil, nil, nil)

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
	handler := NewAgentHandler(nil, sessionRepo, nil)

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
	handler := NewAgentHandler(nil, nil, messageRepo)

	r := gin.New()
	r.GET("/sessions/:id/messages", handler.ListMessages)

	w := getRequest(r, "/sessions/"+sid.String()+"/messages")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_ListMessages_InvalidID(t *testing.T) {
	handler := NewAgentHandler(nil, nil, nil)

	r := gin.New()
	r.GET("/sessions/:id/messages", handler.ListMessages)

	w := getRequest(r, "/sessions/not-a-uuid/messages")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
