package dashboard

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ofws "Wavelet/openflare/plugins/server/domain/fleet/websocket"
	"Wavelet/openflare/plugins/server/kernel/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestHeartbeatOnlineWhileWebSocketConnected(t *testing.T) {
	stale := time.Now().Add(-2 * time.Minute)
	node := &model.OpenFlareNode{NodeID: "dashboard-online-regression", NodeType: "edge_node", LastSeenAt: &stale}
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
	_ = conn.Close()
	deadline = time.Now().Add(time.Second)
	for ofws.IsAgentConnected(node.NodeID) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := computeNodeStatus(node); got != nodeStatusOffline {
		t.Errorf("computeNodeStatus(stale heartbeat, closed websocket) = %q, want offline", got)
	}
}
