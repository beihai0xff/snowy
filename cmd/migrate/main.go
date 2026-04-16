package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
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

	db, err := mysqlrepo.NewDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect mysql", "error", err)

		return 1
	}

	sqlDB, err := db.DB()
	if err == nil {
		defer func() { _ = sqlDB.Close() }()
	}

	if err := mysqlrepo.RunMigrations(context.Background(), db); err != nil {
		slog.Error("failed to run mysql migrations", "error", err)

		return 1
	}

	slog.Info("mysql schema migrated", "db", cfg.Database.Name, "host", cfg.Database.Host)

	return 0
}
