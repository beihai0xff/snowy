//go:build integration

package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/app"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

func TestAppRun_AllModeServesHTTPAndConsumesTasks(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, resetRedis(context.Background()))
	})

	port := 18080
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            port,
			RunMode:         config.RunModeAll,
			Mode:            "test",
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			ShutdownTimeout: 3 * time.Second,
		},
		Database: integrationDatabaseConfig(),
		Redis:    integrationRedisConfig(),
		Auth:     integrationAuthConfig(),
		RateLimit: config.RateLimitConfig{
			AuthenticatedRPM: 60,
			AnonymousRPM:     10,
		},
		Observability: config.ObservabilityConfig{
			PrometheusPath: "/metrics",
			LogLevel:       "error",
			LogFormat:      "text",
		},
	}

	application, err := app.New(cfg)
	require.NoError(t, err)
	defer application.Close()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- application.Run(runCtx)
	}()

	httpClient := &http.Client{Timeout: 2 * time.Second}
	require.NoError(t, withRetry(context.Background(), 20, 200*time.Millisecond, func() error {
		resp, err := httpClient.Get("http://127.0.0.1:18080/healthz")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errors.New("healthz not ready")
		}
		return nil
	}))

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	defer client.Close()

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	defer inspector.Close()

	_, err = client.EnqueueContext(context.Background(), asynq.NewTask("biology:graph", []byte(`{"topic":"photosynthesis"}`)))
	require.NoError(t, err)

	require.NoError(t, withRetry(context.Background(), 20, 200*time.Millisecond, func() error {
		info, err := inspector.GetQueueInfo("default")
		if err != nil {
			return err
		}
		if info.ProcessedTotal < 1 {
			return errors.New("worker has not processed task yet")
		}
		return nil
	}))

	cancel()
	require.NoError(t, <-runErrCh)
}
