//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/agent"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
	"github.com/beihai0xff/snowy/internal/user"
)

func TestUserRepositoryIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMySQL(ctx))

	repo := mysqlrepo.NewUserRepository(integrationDB)
	phone := fmt.Sprintf("138%08d", time.Now().UnixNano()%100000000)
	u := &user.User{
		ID:          uuid.New(),
		Phone:       phone,
		Nickname:    "integration-user",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	require.NoError(t, repo.Create(ctx, u))

	gotByPhone, err := repo.GetByPhone(ctx, phone)
	require.NoError(t, err)
	assert.Equal(t, u.ID, gotByPhone.ID)
	assert.Equal(t, u.Nickname, gotByPhone.Nickname)

	gotByID, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, phone, gotByID.Phone)
}

func TestUserServiceRegisterIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMySQL(ctx))

	userRepo := mysqlrepo.NewUserRepository(integrationDB)
	favoriteRepo := mysqlrepo.NewFavoriteRepository(integrationDB)
	historyRepo := mysqlrepo.NewHistoryRepository(integrationDB)
	transactor := mysqlrepo.NewTransactor(integrationDB)

	svc := user.NewService(userRepo, favoriteRepo, historyRepo, transactor, integrationAuthConfig())
	phone := fmt.Sprintf("136%08d", time.Now().UnixNano()%100000000)

	u, err := svc.Register(ctx, phone, "tx-user")
	require.NoError(t, err)
	require.NotNil(t, u)

	gotByID, err := userRepo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, phone, gotByID.Phone)

	historyItems, total, err := historyRepo.ListByUser(ctx, u.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, historyItems, 1)
	assert.Equal(t, "register", historyItems[0].ActionType)
	assert.Equal(t, "用户注册", historyItems[0].Query)
}

func TestFavoriteRepositoryIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMySQL(ctx))

	userRepo := mysqlrepo.NewUserRepository(integrationDB)
	favoriteRepo := mysqlrepo.NewFavoriteRepository(integrationDB)

	u := &user.User{
		ID:          uuid.New(),
		Phone:       fmt.Sprintf("139%08d", time.Now().UnixNano()%100000000),
		Nickname:    "favorite-owner",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, userRepo.Create(ctx, u))

	fav := &user.Favorite{
		ID:         uuid.New(),
		UserID:     u.ID,
		TargetType: "physics",
		TargetID:   "run_123",
		Title:      "平抛运动分析",
		CreatedAt:  time.Now(),
	}

	require.NoError(t, favoriteRepo.Add(ctx, fav))

	favorites, total, err := favoriteRepo.ListByUser(ctx, u.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, favorites, 1)
	assert.Equal(t, fav.Title, favorites[0].Title)

	require.NoError(t, favoriteRepo.Remove(ctx, fav.ID, u.ID))

	favorites, total, err = favoriteRepo.ListByUser(ctx, u.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, favorites)
}

func TestAgentRepositoriesIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMySQL(ctx))

	userRepo := mysqlrepo.NewUserRepository(integrationDB)
	sessionRepo := mysqlrepo.NewAgentSessionRepository(integrationDB)
	messageRepo := mysqlrepo.NewAgentMessageRepository(integrationDB)
	runRepo := mysqlrepo.NewAgentRunRepository(integrationDB)
	toolCallRepo := mysqlrepo.NewAgentToolCallRepository(integrationDB)

	u := &user.User{
		ID:          uuid.New(),
		Phone:       fmt.Sprintf("137%08d", time.Now().UnixNano()%100000000),
		Nickname:    "agent-owner",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, userRepo.Create(ctx, u))

	session := &agent.Session{
		ID:        uuid.New(),
		UserID:    u.ID,
		Mode:      agent.ModeSearch,
		Status:    "active",
		Metadata:  map[string]any{"subject": "physics"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, sessionRepo.Create(ctx, session))

	gotSession, err := sessionRepo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, gotSession.UserID)
	assert.Equal(t, "physics", gotSession.Metadata["subject"])

	message := &agent.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Role:      "user",
		Content:   "牛顿第二定律怎么用？",
		CreatedAt: time.Now(),
	}
	require.NoError(t, messageRepo.Save(ctx, message))

	messages, total, err := messageRepo.ListBySession(ctx, session.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, messages, 1)

	run := &agent.Run{
		ID:            uuid.New(),
		SessionID:     session.ID,
		MessageID:     message.ID,
		Mode:          agent.ModeSearch,
		ModelName:     "gpt5",
		PromptVersion: "v1",
		InputTokens:   128,
		OutputTokens:  64,
		EstimatedCost: 0.001,
		LatencyMS:     120,
		Confidence:    0.91,
		Status:        "success",
		CreatedAt:     time.Now(),
	}
	require.NoError(t, runRepo.Save(ctx, run))

	gotRun, err := runRepo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	assert.Equal(t, run.ModelName, gotRun.ModelName)

	toolCall := &agent.RunToolCall{
		ID:        uuid.New(),
		RunID:     run.ID,
		ToolName:  "SearchTool",
		Input:     map[string]any{"query": "牛顿第二定律"},
		Output:    map[string]any{"count": 1},
		LatencyMS: 20,
		Status:    "success",
		CreatedAt: time.Now(),
	}
	require.NoError(t, toolCallRepo.Save(ctx, toolCall))

	toolCalls, err := toolCallRepo.ListByRun(ctx, run.ID)
	require.NoError(t, err)
	assert.Len(t, toolCalls, 1)
	assert.Equal(t, "SearchTool", toolCalls[0].ToolName)
}

func TestAgentWriteServiceIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMySQL(ctx))

	userRepo := mysqlrepo.NewUserRepository(integrationDB)
	sessionRepo := mysqlrepo.NewAgentSessionRepository(integrationDB)
	messageRepo := mysqlrepo.NewAgentMessageRepository(integrationDB)
	runRepo := mysqlrepo.NewAgentRunRepository(integrationDB)
	toolCallRepo := mysqlrepo.NewAgentToolCallRepository(integrationDB)
	transactor := mysqlrepo.NewTransactor(integrationDB)
	writeSvc := agent.NewWriteService(transactor, sessionRepo, messageRepo, runRepo, toolCallRepo)

	u := &user.User{
		ID:          uuid.New(),
		Phone:       fmt.Sprintf("135%08d", time.Now().UnixNano()%100000000),
		Nickname:    "write-owner",
		Role:        user.RoleStudent,
		AvatarURL:   "",
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, userRepo.Create(ctx, u))

	persisted, err := writeSvc.PersistConversation(ctx, &agent.PersistConversationInput{
		UserID:  u.ID,
		Mode:    agent.ModeSearch,
		Message: "牛顿第二定律怎么用？",
		Filters: agent.Filters{Subject: "physics", Grade: "high-school"},
		Response: &agent.ChatResponse{
			Mode:       agent.ModeSearch,
			Answer:     "F=ma",
			Confidence: 0.92,
			ToolCalls:  []agent.ToolCall{{Tool: "SearchTool", Status: "success"}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, persisted)
	require.NotNil(t, persisted.Session)

	gotSession, err := sessionRepo.GetByID(ctx, persisted.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, gotSession.UserID)
	assert.Equal(t, "physics", gotSession.Metadata["subject"])

	messages, total, err := messageRepo.ListBySession(ctx, persisted.Session.ID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)

	gotRun, err := runRepo.GetByID(ctx, persisted.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, persisted.UserMessage.ID, gotRun.MessageID)

	toolCalls, err := toolCallRepo.ListByRun(ctx, persisted.Run.ID)
	require.NoError(t, err)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "SearchTool", toolCalls[0].ToolName)
}

func TestRedisCacheStoreIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetRedis(ctx))

	store := redisrepo.NewCacheStore(integrationRedis)
	key := fmt.Sprintf("integration:cache:%d", time.Now().UnixNano())
	value := map[string]any{"name": "Alice", "score": 95}

	require.NoError(t, store.Set(ctx, key, value, time.Minute))

	exists, err := store.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)

	var got map[string]any
	require.NoError(t, store.Get(ctx, key, &got))
	assert.Equal(t, "Alice", got["name"])

	require.NoError(t, store.Delete(ctx, key))
	exists, err = store.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisSessionStoreIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetRedis(ctx))

	store := redisrepo.NewSessionStore(integrationRedis)
	sessionID := uuid.NewString()
	data := map[string]any{"mode": "physics", "subject": "mechanics"}

	require.NoError(t, store.Set(ctx, sessionID, data))

	got, err := store.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "physics", got["mode"])

	require.NoError(t, store.Delete(ctx, sessionID))
	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err)
}

func TestRedisRateLimiterIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetRedis(ctx))

	limiter := redisrepo.NewRateLimiter(integrationRedis)
	key := fmt.Sprintf("rate:test:%d", time.Now().UnixNano())

	allowed, err := limiter.Allow(ctx, key, 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.Allow(ctx, key, 2, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.Allow(ctx, key, 2, time.Minute)
	require.NoError(t, err)
	assert.False(t, allowed)
}
