package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
)

func TestApplySettingsBoundsHeartbeatInterval(t *testing.T) {
	r := &Runner{Config: &config.Config{HeartbeatInterval: config.MillisecondDuration(10 * time.Second)}}
	r.applySettings(&protocol.AgentSettings{HeartbeatInterval: 1})
	if got := r.Config.HeartbeatInterval.Duration(); got != 3*time.Second {
		t.Fatalf("applySettings(1ms) interval = %v, want 3s", got)
	}
}

func TestWebSocketReportingWaitsForHTTPAcknowledgement(t *testing.T) {
	dir := t.TempDir()
	buffer := state.NewObservabilityBufferStore(filepath.Join(dir, "buffer.json"))
	service := &fakeHeartbeatService{heartbeatErrs: []error{errors.New("offline"), nil}}
	r := withHeartbeatCycle(&Runner{Config: &config.Config{DataDir: dir, NodeIPConfigured: true, ObservabilityReplayMinutes: 60}, StateStore: state.NewStore(filepath.Join(dir, "state.json")), HeartbeatService: service, SyncService: &fakeSyncService{}}, buffer)
	conn := &fakeWebSocketConnection{}
	if err := r.sendWebSocketStatus(context.Background(), "node", conn); err != nil {
		t.Fatal(err)
	}
	records, err := buffer.Replayable(0, 0)
	if err != nil || len(records) == 0 || service.heartbeatCalls != 1 || conn.pongCalls != 1 {
		t.Fatalf("sendWebSocketStatus(failed HTTP) = %v, %v, calls=%d pong=%d, want retained buffer and live WS", records, err, service.heartbeatCalls, conn.pongCalls)
	}
	time.Sleep(3 * time.Second)
	if err := r.sendWebSocketStatus(context.Background(), "node", conn); err != nil {
		t.Fatal(err)
	}
	records, err = buffer.Replayable(0, 0)
	if err != nil || len(records) != 0 || service.heartbeatCalls != 2 {
		t.Fatalf("sendWebSocketStatus(successful HTTP) = %v, %v, calls=%d, want acknowledged buffer", records, err, service.heartbeatCalls)
	}
}
