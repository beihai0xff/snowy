// Package ws 实现 v7 §10 D2 实时协同：会话级 WebSocket Hub + Redis presence。
//
// 设计要点：
//   - 每个会话（session_id）维护一个进程内 Hub，承载多客户端的 fan-out 广播；
//   - presence 存储在 Redis Set：snowy:ws:session:<id> = {client_id...}，
//     TTL=ConfigKeepalive；客户端断开或 ping 失败后从 set 中 SREM；
//   - 进入会话时回放最近 N 条 op log（Redis List），随后由 Hub 广播实时事件；
//   - 跨进程：使用 Redis Pub/Sub 通道 snowy:ws:bus:<session>，每个进程订阅一次，
//     把收到的消息 fan-out 到本地连接（一致性弱但延迟低，符合协同推演定位）。
//
// 单元测试目前依靠 redismock；端到端 (e2e) 的多客户端连接验证留给集成测试。
//
//nolint:funcorder // Manager helpers are kept near lifecycle code to preserve websocket flow readability.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	presenceKeyPrefix  = "snowy:ws:session:"
	busChannelPrefix   = "snowy:ws:bus:"
	oplogKeyPrefix     = "snowy:ws:oplog:"
	presenceTTL        = 90 * time.Second
	oplogMaxLen        = 200
	writeTimeout       = 5 * time.Second
	pongTimeout        = 60 * time.Second
	pingPeriod         = (pongTimeout * 8) / 10
	readMaxBytes       = 64 * 1024
	responseCodeKey    = "code"
	responseMessageKey = "message"
)

// Event 协同总线事件。payload 由客户端 / 服务端自由扩展。
type Event struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	ClientID  string          `json:"client_id"`
	UserID    string          `json:"user_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// Hub 单个会话的本地连接集合。
type sessionHub struct {
	sessionID string
	clients   map[*client]struct{}
	mu        sync.RWMutex
	cancel    context.CancelFunc
}

// Manager 进程级管理器：会话 hub 注册表 + Redis presence/bus。
type Manager struct {
	rdb      *redis.Client
	upgrader websocket.Upgrader
	hubs     map[string]*sessionHub
	mu       sync.Mutex
}

// NewManager 构造 Manager；upgrader 默认允许跨域（Gin CORS 中间件已在更外层处理）。
func NewManager(rdb *redis.Client) *Manager {
	return &Manager{
		rdb:  rdb,
		hubs: make(map[string]*sessionHub),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(_ *http.Request) bool { return true },
		},
	}
}

func (m *Manager) presenceKey(session string) string { return presenceKeyPrefix + session }
func (m *Manager) busChannel(session string) string  { return busChannelPrefix + session }
func (m *Manager) oplogKey(session string) string    { return oplogKeyPrefix + session }

// Handle 是 gin 路由处理函数：GET /api/v1/ws/session/:id。
func (m *Manager) Handle(c *gin.Context) {
	session := c.Param("id")
	if session == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session id"})

		return
	}

	conn, err := m.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Warn("ws upgrade failed", "err", err)

		return
	}

	userID, _ := c.Get("user_id")
	clientID := uuid.NewString()
	cl := &client{
		id:      clientID,
		userID:  asString(userID),
		conn:    conn,
		send:    make(chan []byte, 32),
		manager: m,
		session: session,
	}

	hub := m.getOrCreateHub(session)
	hub.register(cl)

	// 异步 add presence + 回放 oplog + 启动读写循环。
	go cl.writeLoop()
	go cl.readLoop()

	cl.onJoin()
}

func (m *Manager) getOrCreateHub(session string) *sessionHub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.hubs[session]; ok {
		return h
	}

	ctx, cancel := context.WithCancel(context.Background())
	h := &sessionHub{
		sessionID: session,
		clients:   make(map[*client]struct{}),
		cancel:    cancel,
	}

	m.hubs[session] = h
	if m.rdb != nil {
		go m.subscribe(ctx, h)
	}

	return h
}

// subscribe 启动 Redis Pub/Sub 监听，将跨进程消息 fan-out 到本地连接。
func (m *Manager) subscribe(ctx context.Context, h *sessionHub) {
	sub := m.rdb.Subscribe(ctx, m.busChannel(h.sessionID))
	defer sub.Close()

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			h.broadcastLocal([]byte(msg.Payload))
		}
	}
}

func (m *Manager) removeHubIfEmpty(h *sessionHub) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h.mu.RLock()
	empty := len(h.clients) == 0
	h.mu.RUnlock()

	if empty {
		if h.cancel != nil {
			h.cancel()
		}

		delete(m.hubs, h.sessionID)
	}
}

func (h *sessionHub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *sessionHub) unregister(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

func (h *sessionHub) broadcastLocal(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// 慢消费者：丢弃，避免阻塞 hub。
		}
	}
}

// publish 把事件写入 Redis Pub/Sub 与 oplog；本地由 subscribe 回放。
func (m *Manager) publish(ctx context.Context, evt *Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	if m.rdb == nil {
		// 测试 / 无 Redis 模式：直接本地广播。
		if h, ok := m.hubs[evt.SessionID]; ok {
			h.broadcastLocal(data)
		}

		return nil
	}

	pipe := m.rdb.TxPipeline()
	pipe.Publish(ctx, m.busChannel(evt.SessionID), data)
	pipe.LPush(ctx, m.oplogKey(evt.SessionID), data)
	pipe.LTrim(ctx, m.oplogKey(evt.SessionID), 0, oplogMaxLen-1)
	pipe.Expire(ctx, m.oplogKey(evt.SessionID), 24*time.Hour)
	_, err = pipe.Exec(ctx)

	return err
}

// Presence 返回当前会话在线客户端 ID 列表（去重）。
func (m *Manager) Presence(ctx context.Context, session string) ([]string, error) {
	if m.rdb == nil {
		return nil, nil
	}

	return m.rdb.SMembers(ctx, m.presenceKey(session)).Result()
}

// PresenceCount 返回当前在线客户端数。
func (m *Manager) PresenceCount(ctx context.Context, session string) (int64, error) {
	if m.rdb == nil {
		return 0, nil
	}

	return m.rdb.SCard(ctx, m.presenceKey(session)).Result()
}

// PresenceHandler 是 gin 路由：GET /api/v1/ws/session/:id/presence。
func (m *Manager) PresenceHandler(c *gin.Context) {
	session := c.Param("id")
	if session == "" {
		c.JSON(http.StatusBadRequest, gin.H{responseCodeKey: 400, responseMessageKey: "missing session id"})

		return
	}

	members, err := m.Presence(c.Request.Context(), session)
	if err != nil && !errors.Is(err, redis.Nil) {
		c.JSON(http.StatusInternalServerError, gin.H{responseCodeKey: 500, responseMessageKey: err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		responseCodeKey: 0,
		"data": gin.H{
			"session_id": session,
			"members":    members,
			"count":      len(members),
		},
	})
}

func asString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", v)
}
