// Snowy API Server 入口。
// 参考技术方案 §7.3。
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

// 编译注入的版本信息。
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

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)

		return 1
	}

	// 初始化日志
	common.InitLogger(cfg.Observability.LogLevel, cfg.Observability.LogFormat)
	slog.Info("snowy-api starting",
		"version", Version,
		"build_time", BuildTime,
		"commit", Commit,
	)

	// 创建应用
	application, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to create app", "error", err)

		return 1
	}
	defer application.Close()

	// 信号处理：优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal", "signal", sig)
		cancel()
	}()

	// 启动 API 服务
	if err := application.RunAPI(ctx); err != nil {
		slog.Error("api server error", "error", err)

		return 1
	}

	return 0
}
