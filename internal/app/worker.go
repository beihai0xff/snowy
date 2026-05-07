package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

const (
	taskContentIngest = "content:ingest"
	taskContentIndex  = "content:index"
	taskBiologyGraph  = "biology:graph"
)

// RunWorker 启动异步任务 Worker（基于 Asynq），支持优雅关闭。
// 参考技术方案 §10.6。
func (a *App) RunWorker(ctx context.Context) error {
	if a.worker == nil || a.worker.server == nil || a.worker.mux == nil {
		return errors.New("worker surface is not configured")
	}

	slog.Info("worker starting", "redis", a.cfg.Redis.Addr)

	if err := a.worker.server.Start(a.worker.mux); err != nil {
		return fmt.Errorf("start asynq server: %w", err)
	}

	<-ctx.Done()
	slog.Info("worker shutting down...")
	a.worker.server.Shutdown()
	slog.Info("worker stopped gracefully")

	return nil
}

func newWorkerServer(cfg *config.Config) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB},
		asynq.Config{Concurrency: 10},
	)
}

func newWorkerMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(taskContentIngest, logTaskHandler(taskContentIngest))
	mux.HandleFunc(taskContentIndex, logTaskHandler(taskContentIndex))
	mux.HandleFunc(taskBiologyGraph, logTaskHandler(taskBiologyGraph))

	return mux
}

func logTaskHandler(taskName string) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		payload := map[string]any{}
		if len(task.Payload()) > 0 {
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				slog.WarnContext(ctx, "task payload is not json", "task", taskName, "error", err)
			}
		}

		slog.InfoContext(ctx, "worker task executed", "task", taskName, "payload", payload)

		return nil
	}
}
