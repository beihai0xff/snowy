package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func newTestServer(t *testing.T) (*Manager, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mgr := NewManager(nil) // 无 Redis：走本地 fan-out
	r := gin.New()
	r.GET("/api/v1/ws/session/:id", mgr.Handle)
	r.GET("/api/v1/ws/session/:id/presence", mgr.PresenceHandler)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return mgr, srv
}

func wsURL(httpURL, path string) string {
	u := strings.Replace(httpURL, "http://", "ws://", 1)
	return u + path
}

func TestSessionHubFanout(t *testing.T) {
	_, srv := newTestServer(t)
	a, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/ws/session/s1"), nil)
	if err != nil {
		t.Fatalf("dial a: %v", err)
	}
	defer a.Close()
	b, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/ws/session/s1"), nil)
	if err != nil {
		t.Fatalf("dial b: %v", err)
	}
	defer b.Close()

	// 等待 hub 注册完成。
	time.Sleep(50 * time.Millisecond)

	// A 发送一条 message，B 应该收到（B 也会收到自己的回环 + A 的 join 通告）。
	out := map[string]any{"type": "demo.op", "payload": map[string]any{"hello": "world"}}
	data, _ := json.Marshal(out)
	if err := a.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	gotDemoOp := false
	for time.Now().Before(deadline) {
		_ = b.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, msg, err := b.ReadMessage()
		if err != nil {
			break
		}
		var evt Event
		if err := json.Unmarshal(msg, &evt); err != nil {
			continue
		}
		if evt.Type == "demo.op" {
			gotDemoOp = true
			break
		}
	}
	if !gotDemoOp {
		t.Fatalf("client B did not receive demo.op broadcast")
	}
}

func TestPresenceHandler(t *testing.T) {
	_, srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/ws/session/empty/presence")
	if err != nil {
		t.Fatalf("get presence: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			SessionID string   `json:"session_id"`
			Members   []string `json:"members"`
			Count     int      `json:"count"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.SessionID != "empty" {
		t.Fatalf("session id mismatch: %q", body.Data.SessionID)
	}
}

func TestManagerPublishNoRedisLocalBroadcast(t *testing.T) {
	mgr := NewManager(nil)
	// 直接调用 publish 验证不 panic。
	err := mgr.publish(context.Background(), &Event{Type: "x", SessionID: "s2", ClientID: "c", Timestamp: time.Now().UnixMilli()})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
}
