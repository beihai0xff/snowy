package app

import (
	"context"
	"log/slog"
)

// RunWorker 启动异步任务 Worker（基于 Asynq），支持优雅关闭。
// 参考技术方案 §10.6。
func (a *App) RunWorker(ctx context.Context) error {
	slog.Info("worker starting", "redis", a.cfg.Redis.Addr)

	// TODO: 初始化 Asynq Server
	// srv := asynq.NewServer(
	//     asynq.RedisClientOpt{Addr: a.cfg.Redis.Addr, Password: a.cfg.Redis.Password},
	//     asynq.Config{Concurrency: 10},
	// )

	// TODO: 注册任务处理函数
	// mux := asynq.NewServeMux()
	// mux.HandleFunc("content:ingest", contentIngestHandler)
	// mux.HandleFunc("content:index", contentIndexHandler)
	// mux.HandleFunc("biology:graph", biologyGraphHandler)

	// 等待 context 取消
	<-ctx.Done()
	slog.Info("worker shutting down...")

	// TODO: srv.Shutdown()

	slog.Info("worker stopped gracefully")
	return nil
}
