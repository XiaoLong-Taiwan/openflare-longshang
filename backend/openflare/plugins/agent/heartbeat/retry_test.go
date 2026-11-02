package heartbeat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/httpclient"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
)

func TestReportingSplitBackoffAndAcknowledgement(t *testing.T) {
	status := http.StatusRequestEntityTooLarge
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error_msg":"","data":{}}`))
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	buffer := state.NewObservabilityBufferStore(filepath.Join(dir, "buffer.json"))
	window := time.Now().Unix()/60*60 - 60
	logs := make([]protocol.NodeAccessLog, 800)
	for i := range logs {
		logs[i] = protocol.NodeAccessLog{LoggedAtUnix: window + int64(i), Path: "/retry"}
	}
	if err := buffer.Upsert(state.ObservabilityBufferRecord{WindowStartedAtUnix: window, AccessLogs: logs}, 0); err != nil {
		t.Fatal(err)
	}
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, ObservabilityReplayMinutes: 60}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), ObservabilityBuffer: buffer, Heartbeat: New(httpclient.New(server.URL, "token", time.Second))}
	if _, err := c.Perform(context.Background(), "node", false, nil); err == nil {
		t.Fatal("Perform(413) = nil, want error")
	}
	if c.byteLimit != 512*1024 || c.logLimit != 500 || c.bufferLimit != 2 {
		t.Fatalf("Perform(413) limits = %d/%d/%d, want 524288/500/2", c.byteLimit, c.logLimit, c.bufferLimit)
	}
	status = http.StatusTooManyRequests
	c.lastCollection = time.Now().Add(-time.Minute)
	if _, err := c.Perform(context.Background(), "node", false, nil); err == nil {
		t.Fatal("Perform(429) = nil, want error")
	}
	if time.Until(c.nextAttempt) < 8*time.Second || c.retryDelay != 3*time.Second {
		t.Fatalf("Perform(429) delay = %v/%v, want Retry-After >=8s and backoff 3s", time.Until(c.nextAttempt), c.retryDelay)
	}
	c.lastCollection = time.Now().Add(-time.Minute)
	_, _ = c.Perform(context.Background(), "node", false, nil)
	if calls != 2 {
		t.Fatalf("Perform(during backoff) calls = %d, want 2", calls)
	}
	c.nextAttempt = time.Time{}
	_, _ = c.Perform(context.Background(), "node", false, nil)
	if c.retryDelay != 6*time.Second {
		t.Fatalf("Perform(second 429) backoff = %v, want 6s", c.retryDelay)
	}
	remaining, err := buffer.Replayable(0, 0)
	if err != nil || len(remaining[0].AccessLogs) != 800 {
		t.Fatalf("Replayable(after failures) = %v, %v, want all 800 logs retained", remaining, err)
	}
	status = http.StatusOK
	c.nextAttempt = time.Time{}
	c.lastCollection = time.Now().Add(-time.Minute)
	if _, err := c.Perform(context.Background(), "node", false, nil); err != nil {
		t.Fatal(err)
	}
	remaining, err = buffer.Replayable(0, 0)
	if err != nil || len(remaining[0].AccessLogs) != 300 || c.retryDelay != 0 {
		t.Fatalf("Replayable(after success) = %v, %v, want 300 logs retained and reset backoff", remaining, err)
	}
}
