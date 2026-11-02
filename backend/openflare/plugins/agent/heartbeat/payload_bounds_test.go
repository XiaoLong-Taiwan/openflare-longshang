package heartbeat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/httpclient"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
)

func TestPayloadLimitsApplyWithoutWorkingBuffer(t *testing.T) {
	for _, brokenBuffer := range []bool{false, true} {
		dir := t.TempDir()
		c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, NodeName: strings.Repeat("n", 1024*1024)}, StateStore: state.NewStore(filepath.Join(dir, "state.json"))}
		if brokenBuffer {
			blocker := filepath.Join(dir, "file")
			if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			c.ObservabilityBuffer = state.NewObservabilityBufferStore(filepath.Join(blocker, "buffer.json"))
		}
		payload, ack := c.PrepareHeartbeatPayload(context.Background(), "node")
		if payload.SchemaVersion != 0 || len(ack) != 0 || c.prepareError == nil {
			t.Fatalf("PrepareHeartbeatPayload(brokenBuffer=%v) schema=%d ack=%v error=%v, want rejected and visible error", brokenBuffer, payload.SchemaVersion, ack, c.prepareError)
		}
		c.lastCollection = time.Time{}
		if _, err := c.Perform(context.Background(), "node", false, nil); err == nil {
			t.Errorf("Perform(brokenBuffer=%v) = nil, want preparation error", brokenBuffer)
		}
	}
}

func TestOversizedBufferedLogDoesNotBlockLaterWindow(t *testing.T) {
	dir := t.TempDir()
	buffer := state.NewObservabilityBufferStore(filepath.Join(dir, "buffer.json"))
	window := time.Now().Unix()/60*60 - 120
	bad := protocol.NodeAccessLog{LoggedAtUnix: window, UserAgent: strings.Repeat("a", 17*1024)}
	good := protocol.NodeAccessLog{LoggedAtUnix: window + 60, Path: "/good"}
	for i, row := range []protocol.NodeAccessLog{bad, good} {
		if err := buffer.Upsert(state.ObservabilityBufferRecord{WindowStartedAtUnix: window + int64(i)*60, AccessLogs: []protocol.NodeAccessLog{row}}, 0); err != nil {
			t.Fatal(err)
		}
	}
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, ObservabilityReplayMinutes: 60}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), ObservabilityBuffer: buffer}
	payload, ack := c.PrepareHeartbeatPayload(context.Background(), "node")
	found := false
	for _, record := range payload.Buffered {
		for _, row := range record.AccessLogs {
			if row == bad {
				t.Fatal("PrepareHeartbeatPayload(old oversized row) sent oversized row")
			}
			found = found || row == good
		}
	}
	if !found || c.prepareError == nil {
		t.Fatalf("PrepareHeartbeatPayload(oversized row) found=%v error=%v, want later row and visible error", found, c.prepareError)
	}
	c.AckObservabilityWindows(ack)
	remaining, err := buffer.Replayable(0, 0)
	if err != nil || len(remaining) != 1 || len(remaining[0].AccessLogs) != 1 || remaining[0].AccessLogs[0] != bad {
		t.Fatalf("AckObservabilityWindows(oversized row) = %v, %v, want only oversized row retained", remaining, err)
	}
}

func TestRepeated413KeepsSendableFloorAndRecovers(t *testing.T) {
	status := http.StatusRequestEntityTooLarge
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error_msg":"","data":{}}`))
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), Heartbeat: New(httpclient.New(server.URL, "token", time.Second))}
	for i := 0; i < 15; i++ {
		c.lastCollection = time.Time{}
		c.nextAttempt = time.Time{}
		if _, err := c.Perform(context.Background(), "node", false, nil); err == nil {
			t.Fatalf("Perform(413 attempt %d) = nil, want error", i)
		}
	}
	if c.byteLimit != 64*1024 || calls != 15 || time.Until(c.nextAttempt) < 50*time.Second {
		t.Fatalf("Perform(repeated 413) bytes=%d calls=%d next=%v, want 64KiB/15/1min backoff", c.byteLimit, calls, c.nextAttempt)
	}
	status = http.StatusOK
	c.lastCollection = time.Time{}
	c.nextAttempt = time.Time{}
	if _, err := c.Perform(context.Background(), "node", false, nil); err != nil || calls != 16 {
		t.Fatalf("Perform(recovered server) = %v calls=%d, want nil/16", err, calls)
	}
}
