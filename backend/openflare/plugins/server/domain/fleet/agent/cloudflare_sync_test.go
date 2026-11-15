package agent

import (
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
)

func TestCloudflareNodeChanged(t *testing.T) {
	now := time.Now()
	previous := model.OpenFlareNode{IP: "192.0.2.1", Status: "online", OpenrestyStatus: "healthy", LastSeenAt: &now}
	current := previous
	seen := now.Add(15 * time.Second)
	current.LastSeenAt = &seen
	if cloudflareNodeChanged(previous, current) {
		t.Error("cloudflareNodeChanged(normal heartbeat) = true, want false")
	}
	current.IP = "192.0.2.2"
	if !cloudflareNodeChanged(previous, current) {
		t.Error("cloudflareNodeChanged(IP changed) = false, want true")
	}
	current = previous
	current.OpenrestyStatus = "unhealthy"
	if !cloudflareNodeChanged(previous, current) || !cloudflareNodeChanged(current, previous) {
		t.Error("cloudflareNodeChanged(health transition) = false, want true in both directions")
	}
	if cloudflareNodeChanged(current, current) {
		t.Error("cloudflareNodeChanged(still unhealthy) = true, want false")
	}
	previous.LastSeenAt = timePtr(time.Now().Add(-2 * time.Minute))
	current = previous
	now = time.Now()
	current.LastSeenAt = &now
	if !cloudflareNodeChanged(previous, current) {
		t.Error("cloudflareNodeChanged(recovered) = false, want true")
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
