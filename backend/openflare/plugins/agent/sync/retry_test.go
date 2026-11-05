package sync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/nginx"
	"Wavelet/openflare/plugins/agent/protocol"
	"Wavelet/openflare/plugins/agent/state"
)

func TestApplyRetryDelay(t *testing.T) {
	for _, tc := range []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: time.Minute},
		{attempt: 0, want: time.Minute},
		{attempt: 1, want: time.Minute},
		{attempt: 2, want: 2 * time.Minute},
		{attempt: 3, want: 4 * time.Minute},
		{attempt: 4, want: 5 * time.Minute},
		{attempt: 6, want: 5 * time.Minute},
		{attempt: 1000, want: 5 * time.Minute},
	} {
		if got := applyRetryDelay(tc.attempt); got != tc.want {
			t.Errorf("applyRetryDelay(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestBlockedRetryExpiry(t *testing.T) {
	for _, tc := range []struct {
		name    string
		startup bool
		target  *protocol.ActiveConfigMeta
		retryAt time.Time
	}{
		{name: "periodic_expired", target: &protocol.ActiveConfigMeta{Version: "new", Checksum: "new-sum"}, retryAt: time.Now().Add(-time.Minute)},
		{name: "startup_expired", startup: true, target: &protocol.ActiveConfigMeta{Version: "new", Checksum: "new-sum"}, retryAt: time.Now().Add(-time.Minute)},
		{name: "startup_without_summary", startup: true, retryAt: time.Now().Add(-time.Minute)},
		{name: "legacy_without_deadline", target: &protocol.ActiveConfigMeta{Version: "new", Checksum: "new-sum"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
			if err := store.Save(&state.Snapshot{CurrentVersion: "old", CurrentChecksum: "old-sum", BlockedVersion: "new", BlockedChecksum: "new-sum", ApplyRetryCount: 3, ApplyRetryAt: tc.retryAt}); err != nil {
				t.Fatal(err)
			}
			client := &fakeClient{config: protocol.ActiveConfigResponse{Version: "new", Checksum: "new-sum", SourceConfigJSON: testSourceConfigJSON("1", 80)}}
			manager := &fakeManager{currentChecksum: "old-sum", applyOutcome: nginx.ApplyOutcome{Status: nginx.ApplyStatusSuccess}}
			service := New(client, manager, store)
			if err := service.sync(context.Background(), tc.startup, tc.target); err != nil {
				t.Fatalf("sync(%s) = %v, want nil", tc.name, err)
			}
			if client.fetchCalls != 1 || len(manager.applyMainContents) != 1 {
				t.Errorf("sync(%s) fetch/apply = %d/%d, want 1/1", tc.name, client.fetchCalls, len(manager.applyMainContents))
			}
			snapshot, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			if hasBlockedTarget(snapshot) || snapshot.ApplyRetryCount != 0 || !snapshot.ApplyRetryAt.IsZero() || snapshot.CurrentVersion != "new" {
				t.Errorf("sync(%s) state = %+v, want new version with cleared retry state", tc.name, snapshot)
			}
		})
	}
}

func TestBlockedRetryFailurePreservesBackoff(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	client := &fakeClient{config: protocol.ActiveConfigResponse{Version: "new", Checksum: "new-sum", SourceConfigJSON: testSourceConfigJSON("1", 80)}}
	manager := &fakeManager{currentChecksum: "old-sum", applyOutcome: nginx.ApplyOutcome{Status: nginx.ApplyStatusFatal, Message: "apply failed"}}
	for attempt := 1; attempt <= 6; attempt++ {
		if err := store.Save(&state.Snapshot{CurrentVersion: "old", CurrentChecksum: "old-sum", BlockedVersion: "new", BlockedChecksum: "new-sum", ApplyRetryCount: attempt, ApplyRetryAt: time.Now().Add(-time.Second)}); err != nil {
			t.Fatal(err)
		}
		before := time.Now()
		service := New(client, manager, store)
		if err := service.SyncOnce(context.Background(), &protocol.ActiveConfigMeta{Version: "new", Checksum: "new-sum"}); err == nil {
			t.Fatalf("SyncOnce(attempt=%d) = nil, want apply error", attempt)
		}
		snapshot, err := store.Load()
		if err != nil {
			t.Fatal(err)
		}
		wantDelay := applyRetryDelay(attempt + 1)
		if snapshot.ApplyRetryCount != attempt+1 || snapshot.ApplyRetryAt.Before(before.Add(wantDelay)) || snapshot.ApplyRetryAt.After(time.Now().Add(wantDelay)) {
			t.Errorf("SyncOnce(attempt=%d) retry = %d at %v, want count %d and delay %v", attempt, snapshot.ApplyRetryCount, snapshot.ApplyRetryAt, attempt+1, wantDelay)
		}
		if snapshot.CurrentVersion != "old" || !isBlockedTarget(snapshot, "new", "new-sum") {
			t.Errorf("SyncOnce(attempt=%d) state = %+v, want old version and blocked new target", attempt, snapshot)
		}
	}
}

func TestStartupWithoutSummaryHonorsBlockedRetryDeadline(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	if err := store.Save(&state.Snapshot{CurrentVersion: "old", CurrentChecksum: "old-sum", BlockedVersion: "new", BlockedChecksum: "new-sum", ApplyRetryCount: 2, ApplyRetryAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{config: protocol.ActiveConfigResponse{Version: "new", Checksum: "new-sum"}}
	manager := &fakeManager{currentChecksum: "old-sum"}
	if err := New(client, manager, store).SyncOnStartup(context.Background(), nil); err != nil {
		t.Fatalf("SyncOnStartup(nil) = %v, want nil", err)
	}
	if client.fetchCalls != 1 || len(manager.applyMainContents) != 0 || len(manager.ensureCalls) != 1 {
		t.Errorf("SyncOnStartup(nil) fetch/apply/recover = %d/%d/%d, want 1/0/1", client.fetchCalls, len(manager.applyMainContents), len(manager.ensureCalls))
	}
}

func TestForcedSyncResetsBlockedBackoff(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	if err := store.Save(&state.Snapshot{BlockedVersion: "new", BlockedChecksum: "new-sum", ApplyRetryCount: 6, ApplyRetryAt: time.Now().Add(5 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{config: protocol.ActiveConfigResponse{Version: "new", Checksum: "new-sum", SourceConfigJSON: testSourceConfigJSON("1", 80)}}
	manager := &fakeManager{applyOutcome: nginx.ApplyOutcome{Status: nginx.ApplyStatusFatal, Message: "apply failed"}}
	before := time.Now()
	if err := New(client, manager, store).ForceSyncOnce(context.Background(), nil); err == nil {
		t.Fatal("ForceSyncOnce(blocked) = nil, want apply error")
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if client.fetchCalls != 1 || len(manager.applyMainContents) != 1 || snapshot.ApplyRetryCount != 1 || snapshot.ApplyRetryAt.Before(before.Add(time.Minute)) || snapshot.ApplyRetryAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("ForceSyncOnce(blocked) fetch/apply/retry = %d/%d/%+v, want 1/1 and first retry in one minute", client.fetchCalls, len(manager.applyMainContents), snapshot)
	}
}

func TestFetchedNewTargetResetsBlockedBackoff(t *testing.T) {
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	if err := store.Save(&state.Snapshot{BlockedVersion: "old", BlockedChecksum: "old-sum", ApplyRetryCount: 6, ApplyRetryAt: time.Now().Add(5 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{config: protocol.ActiveConfigResponse{Version: "new", Checksum: "new-sum", SourceConfigJSON: testSourceConfigJSON("1", 80)}}
	manager := &fakeManager{applyOutcome: nginx.ApplyOutcome{Status: nginx.ApplyStatusWarning, Message: "rolled back"}}
	if err := New(client, manager, store).SyncOnStartup(context.Background(), nil); err == nil {
		t.Fatal("SyncOnStartup(new target) = nil, want rollback error")
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ApplyRetryCount != 1 || !isBlockedTarget(snapshot, "new", "new-sum") {
		t.Errorf("SyncOnStartup(new target) state = %+v, want blocked new target with retry count 1", snapshot)
	}
}
