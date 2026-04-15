package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const (
	taskContentIngest = "content:ingest"
	taskContentIndex  = "content:index"
	taskBiologyGraph  = "biology:graph"
)

// RunWorker 启动异步任务 Worker（基于 Asynq），支持优雅关闭。
// 参考技术方案 §10.6。
func (a *App) RunWorker(ctx context.Context) error {
	slog.Info("worker starting", "redis", a.cfg.Redis.Addr)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: a.cfg.Redis.Addr, Password: a.cfg.Redis.Password, DB: a.cfg.Redis.DB},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(taskContentIngest, logTaskHandler(taskContentIngest))
	mux.HandleFunc(taskContentIndex, logTaskHandler(taskContentIndex))
	mux.HandleFunc(taskBiologyGraph, logTaskHandler(taskBiologyGraph))

	if err := srv.Start(mux); err != nil {
		return fmt.Errorf("start asynq server: %w", err)
	}

	<-ctx.Done()
	slog.Info("worker shutting down...")
	srv.Shutdown()
	slog.Info("worker stopped gracefully")

	return nil
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
