// Package app 提供应用装配与启动，是 DDD 的组合根（Composition Root）。
// 手动依赖注入：config → store clients → repositories → providers → domain services → handlers → router。
// 参考技术方案 §7.3。
package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
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
	physicscalculator "github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	physicsservice "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/repo/embedding"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	"github.com/beihai0xff/snowy/internal/repo/opensearch"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
	searchservice "github.com/beihai0xff/snowy/internal/repo/search"
	searchquery "github.com/beihai0xff/snowy/internal/repo/search/query"
	searchranking "github.com/beihai0xff/snowy/internal/repo/search/ranking"
	"github.com/beihai0xff/snowy/internal/repo/storage"
	"github.com/beihai0xff/snowy/internal/user"
)

// App 应用实例，持有所有依赖。
type App struct {
	cfg    *config.Config
	db     *gorm.DB
	rdb    *goredis.Client
	router *gin.Engine
}

// New 创建应用实例，完成全部依赖装配。
func New(cfg *config.Config) (*App, error) {
	app := &App{cfg: cfg}

	// ── 1. 基础设施客户端 ──────────────────────────────
	db, err := mysqlrepo.NewDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init mysql: %w", err)
	}

	app.db = db

	slog.Info("mysql connected", "host", cfg.Database.Host, "db", cfg.Database.Name)

	rdb, err := redisrepo.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	app.rdb = rdb

	slog.Info("redis connected", "addr", cfg.Redis.Addr)

	// ── 2. Repository 实例化 ───────────────────────────
	userRepo := mysqlrepo.NewUserRepository(db)
	favoriteRepo := mysqlrepo.NewFavoriteRepository(db)
	historyRepo := mysqlrepo.NewHistoryRepository(db)
	sessionRepo := mysqlrepo.NewAgentSessionRepository(db)
	messageRepo := mysqlrepo.NewAgentMessageRepository(db)
	runRepo := mysqlrepo.NewAgentRunRepository(db)
	toolCallRepo := mysqlrepo.NewAgentToolCallRepository(db)
	transactor := mysqlrepo.NewTransactor(db)

	// ── 3. Redis 组件 ──────────────────────────────────
	rateLimiter := redisrepo.NewRateLimiter(rdb)
	_ = redisrepo.NewCacheStore(rdb)
	_ = redisrepo.NewSessionStore(rdb)

	// ── 4. Provider 实例化 ─────────────────────────────
	primaryLLM := newLLMProvider(cfg.LLM.Primary)
	fallbackLLM := newLLMProvider(cfg.LLM.Fallback)
	embeddingProvider := newEmbeddingProvider(cfg.Embedding)
	openSearchAdapter := opensearch.NewOpenSearchAdapter(cfg.OpenSearch)
	objectStorage := storage.NewMinIOStorage(cfg.MinIO)
	_ = objectStorage

	// ── 5. Domain Service 实例化 ───────────────────────
	userSvc := user.NewService(userRepo, favoriteRepo, historyRepo, transactor, cfg.Auth)
	agentWriteSvc := agent.NewWriteService(transactor, sessionRepo, messageRepo, runRepo, toolCallRepo)
	searchSvc := searchservice.NewService(
		openSearchAdapter,
		searchquery.NewSimpleParser(),
		searchranking.NewScoreRanker(),
		embeddingProvider,
		nil,
	)
	physicsSvc := physicsservice.NewService(physicscalculator.NewSimpleCalculator())
	biologySvc := biologyservice.NewService(
		biologyexperiment.NewSimpleAnalyzer(),
		biologygraph.NewSimpleDiagramBuilder(),
	)

	modelRouter := agentrouter.NewStaticRouter(cfg.LLM)
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
		agentgraph.WithBiologyAnalyzeTool(agenttool.NewBiologyAnalyzeTool(biologySvc)),
		agentgraph.WithCitationTool(agenttool.NewCitationTool()),
		agentgraph.WithCallbacks(callbacks...),
		agentgraph.WithLLMProviders(primaryLLM, fallbackLLM),
	)

	var agentSvc agent.Service = graphBuilder

	// ── 6. Handler 实例化 ──────────────────────────────
	handlers := &handler.Handlers{
		Agent:   handler.NewAgentHandler(agentSvc, agentWriteSvc, sessionRepo, messageRepo),
		Search:  handler.NewSearchHandler(searchSvc),
		Physics: handler.NewPhysicsHandler(physicsSvc),
		Biology: handler.NewBiologyHandler(biologySvc),
		User:    handler.NewUserHandler(userSvc),
	}

	// ── 7. Router 装配 ────────────────────────────────
	app.router = handler.NewRouter(cfg, handlers, rateLimiter)

	slog.Info("app initialized", "mode", cfg.Server.Mode)

	return app, nil
}

func newLLMProvider(cfg config.ModelProviderConfig) llm.Provider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai":
		return llm.NewOpenAIProvider(cfg)
	case "google", "gemini":
		return llm.NewGeminiProvider(cfg)
	default:
		return llm.NewOpenAIProvider(cfg)
	}
}

func newEmbeddingProvider(cfg config.EmbeddingConfig) embedding.Provider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai", "":
		return embedding.NewOpenAIEmbedding(cfg)
	default:
		return embedding.NewOpenAIEmbedding(cfg)
	}
}

// Router 返回 Gin Engine（供 Server 使用）。
func (a *App) Router() *gin.Engine {
	return a.router
}

// Close 释放资源。
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
