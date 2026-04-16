package http

import (
	"github.com/gin-gonic/gin"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/pkg/middleware"
)

// Handlers 聚合所有 HTTP Handler。
type Handlers struct {
	Agent   *AgentHandler
	Search  *SearchHandler
	Physics *PhysicsHandler
	Biology *BiologyHandler
	User    *UserHandler
}

// NewRouter 创建 Gin 路由，组装所有路由和中间件。
// 参考技术方案 §17 API 设计。
func NewRouter(cfg *config.Config, h *Handlers, limiter middleware.RateLimiter) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// ── 全局中间件 ─────────────────────────────────────
	r.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.CORS(),
		middleware.Logger(),
	)

	// ── 健康检查 ───────────────────────────────────────
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── API v1 路由组 ──────────────────────────────────
	v1 := r.Group("/api/v1")

	// 鉴权中间件（支持匿名通过）
	v1.Use(middleware.Auth(cfg.Auth))
	// 限流中间件
	v1.Use(middleware.RateLimit(limiter, cfg.RateLimit))

	// ── 公开接口（无需认证）─────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/login", h.User.Login)
		auth.POST("/register", h.User.Register)
		auth.POST("/send-code", h.User.SendCode)
	}

	// ── 首页推荐（无需认证）─────────────────────────────
	v1.GET("/recommendations", h.User.GetRecommendations)

	// ── Agent 接口 ─────────────────────────────────────
	agent := v1.Group("/agent")
	{
		agent.POST("/chat", h.Agent.Chat)
		agent.POST("/sessions", h.Agent.CreateSession)
		agent.GET("/sessions/:id", h.Agent.GetSession)
		agent.GET("/sessions/:id/messages", h.Agent.ListMessages)
	}

	// ── 搜索接口 ───────────────────────────────────────
	search := v1.Group("/search")
	{
		search.POST("/query", h.Search.Query)
	}

	// ── 建模接口 ───────────────────────────────────────
	modeling := v1.Group("/modeling")
	{
		physics := modeling.Group("/physics")
		{
			physics.POST("/analyze", h.Physics.Analyze)
			physics.POST("/simulate", h.Physics.Simulate)
		}

		biology := modeling.Group("/biology")
		{
			biology.POST("/analyze", h.Biology.Analyze)
		}
	}

	// ── 用户接口（需认证）──────────────────────────────
	user := v1.Group("")
	user.Use(middleware.RequireAuth())
	{
		user.GET("/user/profile", h.User.GetProfile)
		user.GET("/history", h.User.GetHistory)
		user.POST("/favorites", h.User.AddFavorite)
		user.GET("/favorites", h.User.ListFavorites)
	}

	return r
}
