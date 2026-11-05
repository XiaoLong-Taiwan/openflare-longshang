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

func TestRunnerRestoresLocalRuntimeBeforeOfflineHeartbeat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	runtime := &fakeRuntimeManager{healthErr: errors.New("openresty is not running")}
	heartbeat := &fakeHeartbeatService{
		heartbeatErrs: []error{errors.New("control plane offline")},
		onHeartbeat: func(_ int) {
			if runtime.ensureCalls != 1 {
				t.Errorf("Run(offline) recovery calls before heartbeat = %d, want 1", runtime.ensureCalls)
			}
			cancel()
		},
	}
	syncService := &fakeSyncService{}
	runner := withHeartbeatCycle(&Runner{
		Config: &config.Config{
			AccessToken:       "agent-token",
			NodeIP:            "127.0.0.1",
			NodeIPConfigured:  true,
			HeartbeatInterval: config.MillisecondDuration(time.Hour),
		},
		StateStore:       store,
		RuntimeManager:   runtime,
		HeartbeatService: heartbeat,
		SyncService:      syncService,
	}, nil)
	if err := runner.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run(offline) = %v, want context.Canceled", err)
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.OpenrestyStatus != protocol.OpenrestyStatusHealthy {
		t.Errorf("Run(offline) runtime status = %q, want %q", snapshot.OpenrestyStatus, protocol.OpenrestyStatusHealthy)
	}
	if syncService.startupCalls != 0 {
		t.Errorf("Run(offline) startup sync calls = %d, want 0", syncService.startupCalls)
	}
}

func TestEnsureLocalRuntimeSkipsHealthyOrMissingConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "healthy"},
		{name: "missing_config", err: errors.New("openresty config not exists: waiting for initial sync")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := &fakeRuntimeManager{healthErr: tc.err}
			runner := &Runner{RuntimeManager: runtime}
			runner.ensureLocalRuntime(context.Background())
			if runtime.ensureCalls != 0 {
				t.Errorf("ensureLocalRuntime(%s) recovery calls = %d, want 0", tc.name, runtime.ensureCalls)
			}
		})
	}
}

func TestEnsureLocalRuntimeRecordsRecoveryFailure(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	runtime := &fakeRuntimeManager{healthErr: errors.New("openresty is not running"), ensureErr: errors.New("local recovery failed")}
	runner := &Runner{RuntimeManager: runtime, StateStore: store}
	runner.ensureLocalRuntime(context.Background())
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.OpenrestyStatus != protocol.OpenrestyStatusUnhealthy || snapshot.OpenrestyMessage != "local recovery failed" {
		t.Errorf("ensureLocalRuntime(failed) state = %+v, want unhealthy with recovery error", snapshot)
	}
}
