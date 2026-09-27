package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordPusherSendWebhookPayload(t *testing.T) {
	received := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
			return
		}
		received <- payload
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	pusher := &DiscordPusher{}
	_, err := pusher.Send(
		context.Background(),
		Config{URL: server.URL},
		"",
		map[string]any{
			"title":   "登入通知",
			"content": "管理員已登入",
			"user":    map[string]any{"username": "admin"},
		},
		`{"username":"{{user.username}}","embeds":[{"title":"{{title}}","description":"{{content}}","color":3447003}]}`,
		nil,
	)
	if err != nil {
		t.Fatalf("send webhook: %v", err)
	}

	payload := <-received
	if payload["username"] != "admin" {
		t.Fatalf("username = %v, want admin", payload["username"])
	}
	embeds, ok := payload["embeds"].([]any)
	if !ok || len(embeds) != 1 {
		t.Fatalf("embeds = %v, want one embed", payload["embeds"])
	}
	embed := embeds[0].(map[string]any)
	if embed["title"] != "登入通知" || embed["description"] != "管理員已登入" {
		t.Fatalf("embed = %v, want rendered title and description", embed)
	}
}

func TestDiscordPusherValidateConfig(t *testing.T) {
	pusher := &DiscordPusher{}
	if err := pusher.ValidateConfig(Config{URL: "https://discord.com/api/webhooks/id/token", Other: `{"content":"ok"}`}); err != nil {
		t.Fatalf("validate valid config: %v", err)
	}
	if err := pusher.ValidateConfig(Config{URL: "https://discord.com/api/webhooks/id/token", Other: "{"}); err == nil {
		t.Fatal("validate invalid payload: expected error")
	}
}
