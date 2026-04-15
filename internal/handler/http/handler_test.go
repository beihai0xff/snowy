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
	handler := NewAgentHandler(svc)

	r := gin.New()
	r.POST("/chat", handler.Chat)

	w := postJSON(r, "/chat", map[string]string{"message": "What is gravity?"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp.Code)
}

func TestAgentHandler_Chat_BadJSON(t *testing.T) {
	handler := NewAgentHandler(&mockAgentService{})

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
	handler := NewAgentHandler(svc)

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
