package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── helpers ──────────────────────────────────────────────

func performRequest(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r.ServeHTTP(w, req)
	return w
}

var testAuthCfg = config.AuthConfig{
	JWTSecret:       "test-secret-key",
	AccessTokenTTL:  15 * time.Minute,
	RefreshTokenTTL: 7 * 24 * time.Hour,
}

// ── RequestID Tests ──────────────────────────────────────

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		reqID, _ := c.Get("request_id")
		c.String(200, reqID.(string))
	})

	w := performRequest(r, "GET", "/test", nil)

	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, w.Header().Get("X-Request-ID"), w.Body.String())
}

func TestRequestID_PreservesClientHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		reqID, _ := c.Get("request_id")
		c.String(200, reqID.(string))
	})

	w := performRequest(r, "GET", "/test", map[string]string{"X-Request-ID": "client-req-123"})

	assert.Equal(t, "client-req-123", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "client-req-123", w.Body.String())
}

// ── CORS Tests ───────────────────────────────────────────

func TestCORS_SetsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/test", nil)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORS_OptionsPreflight(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "OPTIONS", "/test", nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── Auth Tests ───────────────────────────────────────────

func TestAuth_AlwaysSetsDefaultUser(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Auth(testAuthCfg))
	r.GET("/test", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		role, _ := c.Get("role")
		anon, _ := c.Get("anonymous")
		assert.Equal(t, common.DefaultUserID, uid)
		assert.Equal(t, "student", role)
		assert.False(t, anon.(bool))
		c.Status(200)
	})

	w := performRequest(r, "GET", "/test", nil)
	assert.Equal(t, 200, w.Code)
}

func TestAuth_SetsDefaultUserEvenWithToken(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Auth(testAuthCfg))
	r.GET("/test", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		assert.Equal(t, common.DefaultUserID, uid)
		c.Status(200)
	})

	w := performRequest(r, "GET", "/test", map[string]string{
		"Authorization": "Bearer some-token",
	})
	assert.Equal(t, 200, w.Code)
}

// ── RequireAuth Tests ────────────────────────────────────

func TestRequireAuth_AlwaysPasses(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(RequireAuth())
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/test", nil)
	assert.Equal(t, 200, w.Code)
}

// ── RateLimit Tests ──────────────────────────────────────

type mockLimiter struct {
	allowFn func(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return m.allowFn(ctx, key, limit, window)
}

func TestRateLimit_Allowed(t *testing.T) {
	limiter := &mockLimiter{
		allowFn: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return true, nil
		},
	}
	cfg := config.RateLimitConfig{AuthenticatedRPM: 60, AnonymousRPM: 10}

	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) { c.Set("anonymous", false); c.Set("user_id", "uid-1"); c.Next() })
	r.Use(RateLimit(limiter, cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/test", nil)
	assert.Equal(t, 200, w.Code)
}

func TestRateLimit_Denied(t *testing.T) {
	limiter := &mockLimiter{
		allowFn: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, nil
		},
	}
	cfg := config.RateLimitConfig{AuthenticatedRPM: 60, AnonymousRPM: 10}

	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) { c.Set("anonymous", true); c.Next() })
	r.Use(RateLimit(limiter, cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/test", nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var body common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "RATE_LIMITED", body.Code)
}

func TestRateLimit_LimiterError_FailOpen(t *testing.T) {
	limiter := &mockLimiter{
		allowFn: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, errors.New("redis down")
		},
	}
	cfg := config.RateLimitConfig{AuthenticatedRPM: 60, AnonymousRPM: 10}

	r := gin.New()
	r.Use(RequestID())
	r.Use(func(c *gin.Context) { c.Set("anonymous", false); c.Set("user_id", "uid-1"); c.Next() })
	r.Use(RateLimit(limiter, cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := performRequest(r, "GET", "/test", nil)
	// 限流器故障时放行
	assert.Equal(t, 200, w.Code)
}

// ── Recovery Tests ───────────────────────────────────────

func TestRecovery_PanicReturns500(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Recovery())
	r.GET("/test", func(c *gin.Context) {
		panic("something went wrong")
	})

	w := performRequest(r, "GET", "/test", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body common.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body.Code)
}
