package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const wsEventMessage = "message"

// client 单个 WebSocket 连接的状态。
type client struct {
	id      string
	userID  string
	conn    *websocket.Conn
	send    chan []byte
	manager *Manager
	session string
	closed  atomic.Bool
}

func (c *client) onJoin() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if c.manager.rdb != nil {
		c.manager.rdb.SAdd(ctx, c.manager.presenceKey(c.session), c.id)
		c.manager.rdb.Expire(ctx, c.manager.presenceKey(c.session), presenceTTL)
	}
	// 回放最近 N 条 op log（最新在尾部）
	if c.manager.rdb != nil {
		items, err := c.manager.rdb.LRange(ctx, c.manager.oplogKey(c.session), 0, oplogMaxLen-1).Result()
		if err == nil {
			for _, v := range slices.Backward(items) {
				c.trySend([]byte(v))
			}
		}
	}
	// 通告新成员加入
	joined := &Event{
		Type:      "presence.join",
		SessionID: c.session,
		ClientID:  c.id,
		UserID:    c.userID,
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.manager.publish(context.Background(), joined); err != nil {
		slog.Warn("ws publish join failed", "err", err)
	}
}

func (c *client) onLeave() {
	if c.closed.Swap(true) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if c.manager.rdb != nil {
		c.manager.rdb.SRem(ctx, c.manager.presenceKey(c.session), c.id)
	}

	left := &Event{
		Type:      "presence.leave",
		SessionID: c.session,
		ClientID:  c.id,
		UserID:    c.userID,
		Timestamp: time.Now().UnixMilli(),
	}
	_ = c.manager.publish(context.Background(), left)
	// hub 注销
	if hub, ok := c.manager.hubs[c.session]; ok {
		hub.unregister(c)
		c.manager.removeHubIfEmpty(hub)
	}
}

func (c *client) trySend(data []byte) {
	defer func() { _ = recover() }() // send on closed channel

	select {
	case c.send <- data:
	default:
	}
}

func (c *client) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()

		_ = c.conn.Close()
		c.onLeave()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})

				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// 续期 presence
			if c.manager.rdb != nil {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				c.manager.rdb.Expire(ctx, c.manager.presenceKey(c.session), presenceTTL)
				cancel()
			}
		}
	}
}

func (c *client) readLoop() {
	c.conn.SetReadLimit(readMaxBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))

		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var incoming Event
		if err := json.Unmarshal(data, &incoming); err != nil {
			continue
		}

		incoming.SessionID = c.session
		incoming.ClientID = c.id

		incoming.UserID = c.userID
		if incoming.Timestamp == 0 {
			incoming.Timestamp = time.Now().UnixMilli()
		}

		if incoming.Type == "" {
			incoming.Type = wsEventMessage
		}

		if err := c.manager.publish(context.Background(), &incoming); err != nil {
			slog.Warn("ws publish failed", "err", err)
		}
	}
}
