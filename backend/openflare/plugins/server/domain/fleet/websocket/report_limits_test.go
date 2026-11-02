package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"

	"Wavelet/openflare/plugins/server/domain/fleet/reporting"
)

func TestStatusReadPumpRejectsOversizedReports(t *testing.T) {
	for _, body := range []string{strings.Repeat("x", reporting.MaxPayloadBytes+1)} {
		called := make(chan struct{}, 1)
		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			ServeAgent(c, "limits-test", func(context.Context, string, string, json.RawMessage) error {
				called <- struct{}{}
				return nil
			})
		})
		server := httptest.NewServer(router)
		conn, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		_ = conn.WriteMessage(ws.TextMessage, []byte(body))
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, _, readErr := conn.ReadMessage()
		if readErr == nil {
			_, _, readErr = conn.ReadMessage()
		}
		if readErr == nil {
			t.Error("ReadMessage(oversized report) = nil error, want close")
		}
		select {
		case <-called:
			t.Error("status callback(oversized report) called, want rejected")
		default:
		}
		conn.Close()
		server.Close()
	}
}

func TestStatusAckReflectsHandlerResult(t *testing.T) {
	for _, tc := range []struct {
		name       string
		handlerErr bool
		accepted   bool
	}{
		{name: "accepted", accepted: true},
		{name: "rejected", handlerErr: true},
	} {
		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			ServeAgent(c, "ack-"+tc.name, func(context.Context, string, string, json.RawMessage) error {
				if tc.handlerErr {
					return context.DeadlineExceeded
				}
				return nil
			})
		})
		server := httptest.NewServer(router)
		conn, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		if err := conn.WriteJSON(map[string]any{"type": "status", "payload": map[string]any{}}); err != nil {
			t.Fatalf("WriteJSON(%s status) = %v", tc.name, err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage(%s ack) = %v", tc.name, err)
		}
		var message struct {
			Type    string `json:"type"`
			Payload struct {
				Accepted bool `json:"accepted"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("Unmarshal(%s ack) = %v", tc.name, err)
		}
		if message.Type != "status_ack" || message.Payload.Accepted != tc.accepted {
			t.Errorf("status ack(%s) = (%q, %t), want (%q, %t)", tc.name, message.Type, message.Payload.Accepted, "status_ack", tc.accepted)
		}
		conn.Close()
		server.Close()
	}
}

func TestValidStatusPayload(t *testing.T) {
	if validStatusPayload([]byte(`{"buffered":[` + strings.Repeat(`{},`, 500) + `{}]}`)) {
		t.Error("validStatusPayload(501 buffered records) = true, want false")
	}
}
