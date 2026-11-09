package node

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ofws "Wavelet/openflare/plugins/server/domain/fleet/websocket"
	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestHeartbeatOnlineWhileWebSocketConnected(t *testing.T) {
	stale := time.Now().Add(-2 * time.Minute)
	node := &model.OpenFlareNode{NodeID: "node-online-regression", NodeType: nodeTypeEdgeNode, LastSeenAt: &stale}
	router := gin.New()
	router.GET("/ws", func(c *gin.Context) { ofws.ServeAgent(c, node.NodeID, nil) })
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("Dial(agent websocket) = %v, want nil", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	deadline := time.Now().Add(time.Second)
	for !ofws.IsAgentConnected(node.NodeID) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !ofws.IsAgentConnected(node.NodeID) {
		t.Fatal("IsAgentConnected(after dial) = false, want true")
	}
	if got := computeNodeStatus(node); got != nodeStatusOnline {
		t.Errorf("computeNodeStatus(stale heartbeat, live websocket) = %q, want online", got)
	}
	db := setupNodePageDB(t, 0)
	if err := db.Create(node).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{nodeStatusOnline, nodeStatusOffline, nodeStatusPending} {
		page, err := ListPage(context.Background(), ListQuery{Status: status})
		want := int64(0)
		if status == nodeStatusOnline {
			want = 1
		}
		if err != nil || page.Total != want {
			t.Fatalf("ListPage(%s) = %+v, %v, want total %d", status, page, err, want)
		}
	}
	if err := repository.UpdateOpenFlareNodeColumns(context.Background(), node, map[string]any{"last_seen_at": nil}); err != nil {
		t.Fatal(err)
	}
	page, err := ListPage(context.Background(), ListQuery{Status: nodeStatusOnline})
	if err != nil || page.Total != 1 {
		t.Fatalf("ListPage(connected without heartbeat) = %+v, %v, want total 1", page, err)
	}
	node.LastSeenAt = &stale
	_ = conn.Close()
	deadline = time.Now().Add(time.Second)
	for ofws.IsAgentConnected(node.NodeID) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := computeNodeStatus(node); got != nodeStatusOffline {
		t.Errorf("computeNodeStatus(stale heartbeat, closed websocket) = %q, want offline", got)
	}
}
