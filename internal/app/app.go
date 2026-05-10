// Package app 提供应用装配与启动，是 DDD 的组合根（Composition Root）。
// 手动依赖注入：config → store clients → repositories → providers → domain services → handlers → runtime surfaces。
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/agent"
	agentassembler "github.com/beihai0xff/snowy/internal/agent/assembler"
	agentcallback "github.com/beihai0xff/snowy/internal/agent/callback"
	agentgraph "github.com/beihai0xff/snowy/internal/agent/graph"
	agentpolicy "github.com/beihai0xff/snowy/internal/agent/policy"
	agentrouter "github.com/beihai0xff/snowy/internal/agent/router"
	agenttool "github.com/beihai0xff/snowy/internal/agent/tool"
	handler "github.com/beihai0xff/snowy/internal/handler/http"
	biologyexperiment "github.com/beihai0xff/snowy/internal/modeling/biology/experiment"
	biologygraph "github.com/beihai0xff/snowy/internal/modeling/biology/graph"
	biologyservice "github.com/beihai0xff/snowy/internal/modeling/biology/service"
	generativeservice "github.com/beihai0xff/snowy/internal/modeling/generative"
	physicscalculator "github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	physicsservice "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/monitoring"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
	searchservice "github.com/beihai0xff/snowy/internal/repo/search"
	searchquery "github.com/beihai0xff/snowy/internal/repo/search/query"
	searchranking "github.com/beihai0xff/snowy/internal/repo/search/ranking"
	"github.com/beihai0xff/snowy/internal/user"
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

func newAPISurface(shared *sharedDeps) *apiSurface {
	userRepo := mysqlrepo.NewUserRepository(shared.db)
	favoriteRepo := mysqlrepo.NewFavoriteRepository(shared.db)
	historyRepo := mysqlrepo.NewHistoryRepository(shared.db)
	sessionRepo := mysqlrepo.NewAgentSessionRepository(shared.db)
	messageRepo := mysqlrepo.NewAgentMessageRepository(shared.db)
	runRepo := mysqlrepo.NewAgentRunRepository(shared.db)
	toolCallRepo := mysqlrepo.NewAgentToolCallRepository(shared.db)
	generativeRepo := mysqlrepo.NewGenerativeModelPackageRepository(shared.db)
	transactor := mysqlrepo.NewTransactor(shared.db)

	rateLimiter := redisrepo.NewRateLimiter(shared.rdb)
	_ = redisrepo.NewCacheStore(shared.rdb)
	_ = redisrepo.NewSessionStore(shared.rdb)

	llmRecorder := monitoring.NewLLMRecorder(
		monitoring.WithProviderConfigs(
			monitoring.ProviderConfigFromConfig("primary", shared.cfg.LLM.Primary),
			monitoring.ProviderConfigFromConfig("fallback", shared.cfg.LLM.Fallback),
		),
		monitoring.WithPromptProfiles(monitoring.DefaultPromptProfiles(time.Now())...),
	)
	primaryLLM := monitoring.WrapProvider(newLLMProvider(shared.cfg.LLM.Primary), llmRecorder, "primary")
	fallbackLLM := monitoring.WrapProvider(newLLMProvider(shared.cfg.LLM.Fallback), llmRecorder, "fallback")

	userSvc := user.NewService(userRepo, favoriteRepo, historyRepo, transactor, shared.cfg.Auth)
	agentWriteSvc := agent.NewWriteService(transactor, sessionRepo, messageRepo, runRepo, toolCallRepo)
	searchSvc := searchservice.NewService(
		nil,
		searchquery.NewSimpleParser(),
		searchranking.NewScoreRanker(),
		nil,
		nil,
		searchservice.WithLLMProviders(primaryLLM, fallbackLLM),
	)
	physicsSvc := physicsservice.NewService(
		physicscalculator.NewSimpleCalculator(),
		physicsservice.WithLLMProviders(primaryLLM, fallbackLLM),
	)
	biologySvc := biologyservice.NewService(
		biologyexperiment.NewSimpleAnalyzer(),
		biologygraph.NewSimpleDiagramBuilder(),
	)
	generativeSvc := generativeservice.NewCompilerService(
		searchSvc,
		physicsSvc,
		biologySvc,
		generativeRepo,
		generativeservice.WithLLMProviders(primaryLLM, fallbackLLM),
	)

	modelRouter := agentrouter.NewStaticRouter(shared.cfg.LLM)
	policyEngine := agentpolicy.NewDefaultEngine()
	responseAssembler := agentassembler.NewDefaultAssembler()
	callbacks := []agentcallback.NodeCallback{
		agentcallback.NewAuditLogger(),
		agentcallback.NewMetricsCollector(),
		agentcallback.NewOTelTracer(),
	}

	graphBuilder := agentgraph.NewBuilder(
		agentgraph.WithRouter(modelRouter),
		agentgraph.WithPolicyEngine(policyEngine),
		agentgraph.WithAssembler(responseAssembler),
		agentgraph.WithMessageRepository(messageRepo),
		agentgraph.WithSearchTool(agenttool.NewSearchTool(searchSvc)),
		agentgraph.WithPhysicsAnalyzeTool(agenttool.NewPhysicsAnalyzeTool(physicsSvc)),
		agentgraph.WithRenderCodeTool(agenttool.NewRenderCodeTool(physicsSvc)),
		agentgraph.WithBiologyAnalyzeTool(agenttool.NewBiologyAnalyzeTool(biologySvc)),
		agentgraph.WithCitationTool(agenttool.NewCitationTool()),
		agentgraph.WithCallbacks(callbacks...),
		agentgraph.WithLLMProviders(primaryLLM, fallbackLLM),
	)

	var agentSvc agent.Service = graphBuilder

	handlers := &handler.Handlers{
		Agent:      handler.NewAgentHandler(agentSvc, agentWriteSvc, sessionRepo, messageRepo, userSvc),
		Search:     handler.NewSearchHandler(searchSvc, userSvc),
		Physics:    handler.NewPhysicsHandler(physicsSvc, userSvc),
		Render:     handler.NewRenderHandler(physicsSvc),
		Biology:    handler.NewBiologyHandler(biologySvc, userSvc),
		Generative: handler.NewGenerativeHandler(generativeSvc, userSvc),
		User:       handler.NewUserHandler(userSvc),
		Monitoring: handler.NewMonitoringHandler(llmRecorder),
	}

	return &apiSurface{router: handler.NewRouter(shared.cfg, handlers, rateLimiter)}
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

func newLLMProvider(cfg config.ModelProviderConfig) llm.Provider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "mimo", "xiaomi", "xiaomi-mimo":
		return llm.NewMiMoProvider(cfg)
	case "openai":
		return llm.NewOpenAIProvider(cfg)
	case "google", "gemini":
		return llm.NewGeminiProvider(cfg)
	default:
		return llm.NewUnsupportedProvider(cfg.Provider)
	}
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
