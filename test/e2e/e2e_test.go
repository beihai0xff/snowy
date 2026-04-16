//go:build e2e

// Package e2e 提供端到端测试。
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	handler "github.com/beihai0xff/snowy/internal/handler/http"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

type noopLimiter struct{}

func (noopLimiter) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
	return true, nil
}

func TestE2EHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := handler.NewRouter(&config.Config{Server: config.ServerConfig{Mode: gin.TestMode}}, &handler.Handlers{
		Agent:   handler.NewAgentHandler(nil, nil, nil, nil),
		Search:  handler.NewSearchHandler(nil),
		Physics: handler.NewPhysicsHandler(nil),
		Biology: handler.NewBiologyHandler(nil),
		User:    handler.NewUserHandler(nil),
	}, noopLimiter{})

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var payload map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	assert.Equal(t, "ok", payload["status"])
}
