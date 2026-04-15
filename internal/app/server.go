package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// RunAPI 启动 HTTP API 服务，支持优雅关闭。
func (a *App) RunAPI(ctx context.Context) error {
	srv := &http.Server{
		Addr:         a.cfg.Server.Addr(),
		Handler:      a.router,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
	}

	// 在独立 goroutine 中启动 HTTP 服务
	errCh := make(chan error, 1)

	go func() {
		slog.Info("API server starting", "addr", srv.Addr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http listen: %w", err)
		}
	}()

	// 等待 context 取消（来自信号处理）或服务错误
	select {
	case <-ctx.Done():
		slog.Info("shutting down API server...")

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}

		slog.Info("API server stopped gracefully")

		return nil
	case err := <-errCh:
		return err
	}
}
