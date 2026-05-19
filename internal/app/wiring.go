package app

import (
	"time"

	"github.com/beihai0xff/snowy/internal/agent"
	agentassembler "github.com/beihai0xff/snowy/internal/agent/assembler"
	agentcallback "github.com/beihai0xff/snowy/internal/agent/callback"
	agentgraph "github.com/beihai0xff/snowy/internal/agent/graph"
	agentpolicy "github.com/beihai0xff/snowy/internal/agent/policy"
	agentrouter "github.com/beihai0xff/snowy/internal/agent/router"
	agenttool "github.com/beihai0xff/snowy/internal/agent/tool"
	handler "github.com/beihai0xff/snowy/internal/handler/http"
	"github.com/beihai0xff/snowy/internal/handler/ws"
	biologyservice "github.com/beihai0xff/snowy/internal/modeling/biology/service"
	chemistryservice "github.com/beihai0xff/snowy/internal/modeling/chemistry/service"
	generativeservice "github.com/beihai0xff/snowy/internal/modeling/generative"
	physicscalculator "github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	physicsservice "github.com/beihai0xff/snowy/internal/modeling/physics/service"
	"github.com/beihai0xff/snowy/internal/monitoring"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	"github.com/beihai0xff/snowy/internal/pkg/middleware"
	irepo "github.com/beihai0xff/snowy/internal/repo"
	"github.com/beihai0xff/snowy/internal/repo/llm"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
	searchservice "github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/share"
	"github.com/beihai0xff/snowy/internal/user"
)

type apiRepositories struct {
	userRepo         user.Repository
	favoriteRepo     user.FavoriteRepository
	historyRepo      user.HistoryRepository
	sessionRepo      agent.SessionRepository
	messageRepo      agent.MessageRepository
	messageEventRepo agent.MessageEventRepository
	runRepo          agent.RunRepository
	toolCallRepo     agent.ToolCallRepository
	generativeRepo   generativeservice.Repository
	shareRepo        share.Repository
	answerRecordRepo searchservice.AnswerRecordRepository
	reactionUserRepo user.ReactionRepository
	reactionFeedback searchservice.FeedbackRepository
	transactor       irepo.Transactor
}

type llmStack struct {
	chain    llm.Provider
	recorder *monitoring.LLMRecorder
}

type domainServices struct {
	rateLimiter   middleware.RateLimiter
	userSvc       user.Service
	agentWriteSvc agent.WriteService
	searchSvc     searchservice.Service
	physicsSvc    physicsservice.PhysicsService
	biologySvc    biologyservice.BiologyService
	chemistrySvc  chemistryservice.Service
	generativeSvc generativeservice.Service
	agentSvc      agent.Service
}

func newAPISurface(shared *sharedDeps) *apiSurface {
	repos := newAPIRepositories(shared)
	llmStack := newLLMStack(shared)
	services := newDomainServices(shared, repos, llmStack.chain)
	handlers := newHTTPHandlers(shared, repos, services, llmStack.recorder)

	return &apiSurface{router: handler.NewRouter(shared.cfg, handlers, services.rateLimiter)}
}

func newAPIRepositories(shared *sharedDeps) *apiRepositories {
	reactionRepo := mysqlrepo.NewReactionRepository(shared.db)

	return &apiRepositories{
		userRepo:         mysqlrepo.NewUserRepository(shared.db),
		favoriteRepo:     mysqlrepo.NewFavoriteRepository(shared.db),
		historyRepo:      mysqlrepo.NewHistoryRepository(shared.db),
		sessionRepo:      mysqlrepo.NewAgentSessionRepository(shared.db),
		messageRepo:      mysqlrepo.NewAgentMessageRepository(shared.db),
		messageEventRepo: mysqlrepo.NewAgentMessageEventRepository(shared.db),
		runRepo:          mysqlrepo.NewAgentRunRepository(shared.db),
		toolCallRepo:     mysqlrepo.NewAgentToolCallRepository(shared.db),
		generativeRepo:   mysqlrepo.NewGenerativeModelPackageRepository(shared.db),
		shareRepo:        mysqlrepo.NewShareRepository(shared.db),
		answerRecordRepo: mysqlrepo.NewAnswerRecordRepository(shared.db),
		reactionUserRepo: reactionRepo,
		reactionFeedback: reactionRepo,
		transactor:       mysqlrepo.NewTransactor(shared.db),
	}
}

func newLLMStack(shared *sharedDeps) *llmStack {
	modelConfigs := shared.cfg.LLM.EffectiveModels()

	providerConfigs := make([]monitoring.LLMProviderConfig, 0, len(modelConfigs))
	for i, modelCfg := range modelConfigs {
		providerConfigs = append(providerConfigs, monitoring.ProviderConfigFromConfig(modelRole(i), modelCfg))
	}

	recorder := monitoring.NewLLMRecorder(
		monitoring.WithStore(mysqlrepo.NewLLMCallRecordRepository(shared.db)),
		monitoring.WithProviderConfigs(providerConfigs...),
		monitoring.WithPromptProfiles(monitoring.DefaultPromptProfiles(time.Now())...),
	)

	return &llmStack{chain: buildOrderedLLMChain(modelConfigs, recorder), recorder: recorder}
}

func newDomainServices(shared *sharedDeps, repos *apiRepositories, llmChain llm.Provider) *domainServices {
	userSvc := user.NewService(
		repos.userRepo,
		repos.favoriteRepo,
		repos.historyRepo,
		repos.transactor,
		shared.cfg.Auth,
		repos.reactionUserRepo,
	)
	searchSvc := searchservice.NewService(
		nil,
		searchservice.NewSimpleParser(),
		searchservice.NewScoreRanker(),
		nil,
		nil,
		searchservice.WithLLMProvider(llmChain),
		searchservice.WithAnswerRecordRepository(repos.answerRecordRepo),
		searchservice.WithFeedbackRepository(repos.reactionFeedback),
	)
	physicsSvc := physicsservice.NewService(
		physicscalculator.NewSimpleCalculator(),
		physicsservice.WithLLMProvider(llmChain),
	)
	biologySvc := biologyservice.NewService(
		biologyservice.NewSimpleAnalyzer(),
		biologyservice.NewSimpleDiagramBuilder(),
	)
	chemistrySvc := chemistryservice.NewService()
	generativeSvc := generativeservice.NewCompilerService(
		searchSvc,
		physicsSvc,
		biologySvc,
		repos.generativeRepo,
		generativeservice.WithLLMProvider(llmChain),
		generativeservice.WithChemistryService(chemistrySvc),
	)
	agentWriteSvc := agent.NewWriteService(
		repos.transactor,
		repos.sessionRepo,
		repos.messageRepo,
		repos.runRepo,
		repos.toolCallRepo,
		repos.messageEventRepo,
	)

	return &domainServices{
		rateLimiter:   redisrepo.NewRateLimiter(shared.rdb),
		userSvc:       userSvc,
		agentWriteSvc: agentWriteSvc,
		searchSvc:     searchSvc,
		physicsSvc:    physicsSvc,
		biologySvc:    biologySvc,
		chemistrySvc:  chemistrySvc,
		generativeSvc: generativeSvc,
		agentSvc: newAgentService(
			shared.cfg,
			repos,
			searchSvc,
			physicsSvc,
			biologySvc,
			chemistrySvc,
			generativeSvc,
			llmChain,
		),
	}
}

func newAgentService(
	cfg *config.Config,
	repos *apiRepositories,
	searchSvc searchservice.Service,
	physicsSvc physicsservice.PhysicsService,
	biologySvc biologyservice.BiologyService,
	chemistrySvc chemistryservice.Service,
	generativeSvc generativeservice.Service,
	llmChain llm.Provider,
) agent.Service {
	callbacks := []agentcallback.NodeCallback{
		agentcallback.NewAuditLogger(),
		agentcallback.NewMetricsCollector(),
		agentcallback.NewOTelTracer(),
	}

	return agentgraph.NewBuilder(
		agentgraph.WithRouter(agentrouter.NewStaticRouter(cfg.LLM)),
		agentgraph.WithPolicyEngine(agentpolicy.NewDefaultEngine()),
		agentgraph.WithAssembler(agentassembler.NewDefaultAssembler()),
		agentgraph.WithMessageRepository(repos.messageRepo),
		agentgraph.WithSearchTool(agenttool.NewSearchTool(searchSvc)),
		agentgraph.WithPhysicsAnalyzeTool(agenttool.NewPhysicsAnalyzeTool(physicsSvc)),
		agentgraph.WithRenderCodeTool(agenttool.NewRenderCodeTool(physicsSvc)),
		agentgraph.WithBiologyAnalyzeTool(agenttool.NewBiologyAnalyzeTool(biologySvc)),
		agentgraph.WithChemistryAnalyzeTool(agenttool.NewChemistryAnalyzeTool(chemistrySvc)),
		agentgraph.WithCitationTool(agenttool.NewCitationTool()),
		agentgraph.WithGenerativeService(generativeSvc),
		agentgraph.WithRegenerateClassifierLLM(llmChain),
		agentgraph.WithCallbacks(callbacks...),
	)
}

func newHTTPHandlers(
	shared *sharedDeps,
	repos *apiRepositories,
	services *domainServices,
	recorder *monitoring.LLMRecorder,
) *handler.Handlers {
	return &handler.Handlers{
		Agent: handler.NewAgentHandler(
			services.agentSvc,
			services.agentWriteSvc,
			repos.sessionRepo,
			repos.messageRepo,
			services.userSvc,
		).WithEventRepository(repos.messageEventRepo),
		Search:     handler.NewSearchHandler(services.searchSvc, services.userSvc),
		Physics:    handler.NewPhysicsHandler(services.physicsSvc, services.userSvc),
		Render:     handler.NewRenderHandler(services.physicsSvc),
		Biology:    handler.NewBiologyHandler(services.biologySvc, services.userSvc),
		Chemistry:  handler.NewChemistryHandler(services.chemistrySvc, services.userSvc),
		Generative: handler.NewGenerativeHandler(services.generativeSvc, services.userSvc),
		Share:      handler.NewShareHandler(repos.shareRepo, services.generativeSvc),
		WSManager:  ws.NewManager(shared.rdb),
		User:       handler.NewUserHandler(services.userSvc, repos.answerRecordRepo),
		Monitoring: handler.NewMonitoringHandler(recorder),
	}
}
