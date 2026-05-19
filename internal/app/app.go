// Package app 提供应用装配与启动，是 DDD 的组合根（Composition Root）。
// 手动依赖注入：config → store clients → repositories → providers → domain services → handlers → runtime surfaces。
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/monitoring"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/pkg/llmroute"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
)

// App 应用实例，持有共享依赖与可选运行面。
type App struct {
	cfg    *config.Config
	db     *gorm.DB
	rdb    *goredis.Client
	api    *apiSurface
	worker *workerSurface
}

type apiSurface struct {
	router *gin.Engine
}

type workerSurface struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

type sharedDeps struct {
	cfg *config.Config
	db  *gorm.DB
	rdb *goredis.Client
}

// New 创建应用实例，按运行模式装配共享依赖与运行面。
func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if err := cfg.Server.ValidateRunMode(); err != nil {
		return nil, err
	}

	app := &App{cfg: cfg}

	db, err := mysqlrepo.NewDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init mysql: %w", err)
	}

	app.db = db

	slog.Info("mysql connected", "host", cfg.Database.Host, "db", cfg.Database.Name)

	rdb, err := redisrepo.NewClient(cfg.Redis)
	if err != nil {
		app.Close()

		return nil, fmt.Errorf("init redis: %w", err)
	}

	app.rdb = rdb

	slog.Info("redis connected", "addr", cfg.Redis.Addr)

	if cfg.Server.StartupMigrate {
		if err := mysqlrepo.RunMigrations(context.Background(), db); err != nil {
			app.Close()

			return nil, fmt.Errorf("run mysql migrations: %w", err)
		}

		slog.Info("mysql schema migrated", "db", cfg.Database.Name, "host", cfg.Database.Host)
	} else {
		slog.Info("mysql startup migration skipped", "db", cfg.Database.Name, "host", cfg.Database.Host)
	}

	shared := &sharedDeps{cfg: cfg, db: db, rdb: rdb}

	if cfg.Server.APIEnabled() {
		app.api = newAPISurface(shared)
	}

	if cfg.Server.WorkerEnabled() {
		app.worker = newWorkerSurface(shared)
	}

	slog.Info(
		"app initialized",
		"mode", cfg.Server.Mode,
		"run_mode", cfg.Server.EffectiveRunMode(),
		"api_enabled", cfg.Server.APIEnabled(),
		"worker_enabled", cfg.Server.WorkerEnabled(),
	)

	return app, nil
}

func newWorkerSurface(shared *sharedDeps) *workerSurface {
	return &workerSurface{
		server: newWorkerServer(shared.cfg),
		mux:    newWorkerMux(),
	}
}

// Run 按配置启动 API、Worker，支持单运行面或同进程联合运行。
func (a *App) Run(ctx context.Context) error {
	switch a.cfg.Server.EffectiveRunMode() {
	case config.RunModeAPI:
		return a.RunAPI(ctx)
	case config.RunModeWorker:
		return a.RunWorker(ctx)
	case config.RunModeAll:
		var (
			wg             sync.WaitGroup
			results        = make(chan runResult, 2)
			runCtx, cancel = context.WithCancel(ctx)
		)
		defer cancel()

		start := func(name string, fn func(context.Context) error) {
			wg.Go(func() {
				results <- runResult{name: name, err: fn(runCtx)}
			})
		}

		start(config.RunModeAPI, a.RunAPI)
		start(config.RunModeWorker, a.RunWorker)

		var firstErr error

		for remaining := 2; remaining > 0; remaining-- {
			result := <-results
			if result.err != nil && firstErr == nil {
				firstErr = fmt.Errorf("%s surface: %w", result.name, result.err)

				cancel()
			}
		}

		wg.Wait()

		return firstErr
	default:
		return fmt.Errorf("unsupported run mode %q", a.cfg.Server.EffectiveRunMode())
	}
}

type runResult struct {
	name string
	err  error
}

// Router 返回 API 路由；当 API 运行面关闭时返回 nil。
func (a *App) Router() *gin.Engine {
	if a.api == nil {
		return nil
	}

	return a.api.router
}

func modelRole(index int) string {
	return fmt.Sprintf("model_%d", index+1)
}

func buildOrderedLLMChain(modelConfigs []config.ModelProviderConfig, recorder *monitoring.LLMRecorder) llm.Provider {
	providers := make([]llm.Provider, 0, len(modelConfigs))
	for i, modelCfg := range modelConfigs {
		provider := llmroute.NewRetryingProvider(
			llm.NewOpenAIProvider(modelCfg),
			modelCfg.MaxRetries,
			modelCfg.RetryInterval,
		)
		providers = append(providers, monitoring.WrapProvider(provider, recorder, modelRole(i)))
	}

	return llmroute.NewChain("ordered-model-chain", providers...)
}

// Close 释放共享资源。
func (a *App) Close() {
	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}

	if a.rdb != nil {
		_ = a.rdb.Close()
	}

	slog.Info("app resources released")
}
