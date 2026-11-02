package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
)

func TestReportingLimitsAndPartialAcknowledgement(t *testing.T) {
	dir := t.TempDir()
	buffer := state.NewObservabilityBufferStore(filepath.Join(dir, "buffer.json"))
	window := time.Now().Unix()/60*60 - 120
	for i := 0; i < 6; i++ {
		logs := make([]protocol.NodeAccessLog, 1200)
		for j := range logs {
			logs[j] = protocol.NodeAccessLog{LoggedAtUnix: window, Path: fmt.Sprintf("/%d/%d", i, j), UserAgent: strings.Repeat("a", 1500)}
		}
		if err := buffer.Upsert(state.ObservabilityBufferRecord{WindowStartedAtUnix: window - int64(i)*60, AccessLogs: logs}, 0); err != nil {
			t.Fatal(err)
		}
	}
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, ObservabilityReplayMinutes: 60, RequestTimeout: config.MillisecondDuration(10 * time.Second)}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), ObservabilityBuffer: buffer}
	payload, ack := c.PrepareHeartbeatPayload(context.Background(), "node")
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	rows := len(payload.AccessLogs)
	for _, record := range payload.Buffered {
		rows += len(record.AccessLogs)
	}
	t.Logf("payload bytes=%d rows=%d windows=%d", len(encoded), rows, len(payload.Buffered))
	if len(encoded) > 1024*1024 || rows > 1000 || len(payload.Buffered) > 4 || rows == 0 {
		t.Fatalf("PrepareHeartbeatPayload(backlog) = %d bytes, %d rows, %d windows, want <=1MiB, <=1000 rows, <=4 windows and progress", len(encoded), rows, len(payload.Buffered))
	}
	before, err := buffer.Replayable(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 7 {
		t.Fatalf("Replayable(before ack) = %d windows, want 7", len(before))
	}
	c.AckObservabilityWindows(ack)
	after, err := buffer.Replayable(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	remaining := 0
	for _, record := range after {
		remaining += len(record.AccessLogs)
	}
	if remaining != 7200-rows {
		t.Fatalf("AckObservabilityWindows(partial) remaining = %d rows, want %d", remaining, 7200-rows)
	}
	c.lastCollection = time.Now().Add(-time.Minute)
	second, _ := c.PrepareHeartbeatPayload(context.Background(), "node")
	if len(second.Buffered) == 0 {
		t.Fatal("PrepareHeartbeatPayload(next interval) = no backlog, want continued replay")
	}
}

func TestReportingRetriesCurrentWindow(t *testing.T) {
	dir := t.TempDir()
	buffer := state.NewObservabilityBufferStore(filepath.Join(dir, "buffer.json"))
	window := time.Now().Unix() / 60 * 60
	row := protocol.NodeAccessLog{LoggedAtUnix: window, Path: "/failed"}
	if err := buffer.Upsert(state.ObservabilityBufferRecord{WindowStartedAtUnix: window, AccessLogs: []protocol.NodeAccessLog{row}}, 0); err != nil {
		t.Fatal(err)
	}
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, ObservabilityReplayMinutes: 60}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), ObservabilityBuffer: buffer}
	payload, ack := c.PrepareHeartbeatPayload(context.Background(), "node")
	found := false
	for _, record := range payload.Buffered {
		for _, log := range record.AccessLogs {
			found = found || log.Path == row.Path
		}
	}
	if !found {
		t.Fatal("PrepareHeartbeatPayload(current window) omits failed row")
	}
	c.AckObservabilityWindows(ack)
	remaining, err := buffer.Replayable(0, 0)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("AckObservabilityWindows(current window) = %v, %v, want empty buffer", remaining, err)
	}
}

func TestReportingCollectionFrequency(t *testing.T) {
	dir := t.TempDir()
	c := &Cycle{Config: &config.Config{DataDir: dir, NodeIPConfigured: true}, StateStore: state.NewStore(filepath.Join(dir, "state.json"))}
	first, _ := c.PrepareHeartbeatPayload(context.Background(), "node")
	second, _ := c.PrepareHeartbeatPayload(context.Background(), "node")
	if first.HostMetrics == nil || second.HostMetrics != nil || second.EdgeHealth != nil || len(second.AccessLogs) != 0 {
		t.Fatalf("NodePayload(immediate repeat) metrics = %v then %v, want first sample only", first.HostMetrics, second.HostMetrics)
	}
}
