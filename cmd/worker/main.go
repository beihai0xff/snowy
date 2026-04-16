// Snowy Worker 入口 — 异步任务处理。
// 参考技术方案 §10.6。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/beihai0xff/snowy/internal/app"
	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "configs/config.yaml", "config file path")

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)

		return 1
	}

	common.InitLogger(cfg.Observability.LogLevel, cfg.Observability.LogFormat)
	slog.Info("snowy-worker starting",
		"version", Version,
		"build_time", BuildTime,
		"commit", Commit,
	)

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to create app", "error", err)

		return 1
	}
	defer application.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal", "signal", sig)
		cancel()
	}()

	if err := application.RunWorker(ctx); err != nil {
		slog.Error("worker error", "error", err)

		return 1
	}

	return 0
}
