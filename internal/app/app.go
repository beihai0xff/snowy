// Package app 提供应用装配与启动，是 DDD 的组合根（Composition Root）。
// 手动依赖注入：config → store clients → repositories → providers → domain services → handlers → router。
// 参考技术方案 §7.3。
package app

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/beihai0xff/snowy/internal/config"
	handler "github.com/beihai0xff/snowy/internal/handler/http"
	mysqlstore "github.com/beihai0xff/snowy/internal/store/mysql"
	redisstore "github.com/beihai0xff/snowy/internal/store/redis"
	"github.com/beihai0xff/snowy/internal/user"
)

// App 应用实例，持有所有依赖。
type App struct {
	cfg    *config.Config
	db     *sql.DB
	rdb    *goredis.Client
	router *gin.Engine
}

// New 创建应用实例，完成全部依赖装配。
func New(cfg *config.Config) (*App, error) {
	app := &App{cfg: cfg}

	// ── 1. 基础设施客户端 ──────────────────────────────
	db, err := mysqlstore.NewDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init mysql: %w", err)
	}
	app.db = db
	slog.Info("mysql connected", "host", cfg.Database.Host, "db", cfg.Database.Name)

	rdb, err := redisstore.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}
	app.rdb = rdb
	slog.Info("redis connected", "addr", cfg.Redis.Addr)

	// ── 2. Repository 实例化 ───────────────────────────
	userRepo := mysqlstore.NewUserRepository(db)
	_ = mysqlstore.NewAgentSessionRepository(db)
	_ = mysqlstore.NewAgentRunRepository(db)
	_ = mysqlstore.NewAgentMessageRepository(db)

	// ── 3. Redis 组件 ──────────────────────────────────
	rateLimiter := redisstore.NewRateLimiter(rdb)
	_ = redisstore.NewCacheStore(rdb)
	_ = redisstore.NewSessionStore(rdb)

	// ── 4. Provider 实例化 ─────────────────────────────
	// TODO: 初始化 LLM / Embedding / OpenSearch / MinIO providers

	// ── 5. Domain Service 实例化 ───────────────────────
	// User Service — FavoriteRepo/HistoryRepo 暂传 nil，后续补充实现
	userSvc := user.NewService(userRepo, nil, nil, cfg.Auth)

	// TODO: 初始化 Search / Physics / Biology / Agent Services

	// ── 6. Handler 实例化 ──────────────────────────────
	handlers := &handler.Handlers{
		Agent:   handler.NewAgentHandler(nil),   // TODO: 注入 Agent Service
		Search:  handler.NewSearchHandler(nil),  // TODO: 注入 Search Service
		Physics: handler.NewPhysicsHandler(nil), // TODO: 注入 Physics Service
		Biology: handler.NewBiologyHandler(nil), // TODO: 注入 Biology Service
		User:    handler.NewUserHandler(userSvc),
	}

	// ── 7. Router 装配 ────────────────────────────────
	app.router = handler.NewRouter(cfg, handlers, rateLimiter)

	slog.Info("app initialized", "mode", cfg.Server.Mode)
	return app, nil
}

// Router 返回 Gin Engine（供 Server 使用）。
func (a *App) Router() *gin.Engine {
	return a.router
}

// Close 释放资源。
func (a *App) Close() {
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.rdb != nil {
		_ = a.rdb.Close()
	}
	slog.Info("app resources released")
}
