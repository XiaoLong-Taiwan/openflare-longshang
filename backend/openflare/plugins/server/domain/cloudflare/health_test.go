package cloudflare

import (
	"context"
	"errors"
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestHealthCheckDispatchesOnlyChangedTargets(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return &fakeClient{} }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatal(err)
	}
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		return "health-task", nil
	}))
	if err := checkHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("checkHealth(unchanged) dispatches = %d, want 0", calls)
	}
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	node := state.Nodes[0].Node
	node.IP = "192.0.2.20"
	if err := repository.SaveOpenFlareNode(ctx, &node); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := checkHealth(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("checkHealth(changed twice) dispatches = %d, want 1", calls)
	}
}

func TestHealthCheckDispatchFailureCanRetry(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncOK}); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("queue unavailable")
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		return "", wantErr
	}))
	for range 2 {
		if err := checkHealth(ctx); !errors.Is(err, wantErr) {
			t.Errorf("checkHealth(queue failure) error = %v, want %v", err, wantErr)
		}
	}
	if calls != 2 {
		t.Errorf("checkHealth(queue failure) dispatches = %d, want 2", calls)
	}
}

func TestHealthCheckDebounceFailoverAndRecovery(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	backup := model.OpenFlareNode{NodeID: "backup", IP: "192.0.2.30", Status: "online", LastSeenAt: &now}
	if err := repository.CreateOpenFlareNode(ctx, &backup); err != nil {
		t.Fatal(err)
	}
	state.Group.DebounceSeconds = 120
	primary := state.Nodes[0].Node
	if err := repository.SaveCFPointingGroupWithNodes(ctx, &state.Group, []model.CFPointingGroupNode{
		{NodeID: primary.ID, Priority: 0}, {NodeID: backup.ID, Priority: 1},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return &fakeClient{} }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatal(err)
	}
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		return "health-task", nil
	}))
	primary.Status = "offline"
	lastSeen := now.Add(-30 * time.Second)
	primary.LastSeenAt = &lastSeen
	if err := repository.SaveOpenFlareNode(ctx, &primary); err != nil {
		t.Fatal(err)
	}
	if err := checkHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("checkHealth(within debounce) dispatches = %d, want 0", calls)
	}
	lastSeen = now.Add(-121 * time.Second)
	if err := repository.SaveOpenFlareNode(ctx, &primary); err != nil {
		t.Fatal(err)
	}
	if err := checkHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("checkHealth(expired debounce) dispatches = %d, want 1", calls)
	}
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatal(err)
	}
	primary.Status, primary.LastSeenAt = "online", &now
	if err := repository.SaveOpenFlareNode(ctx, &primary); err != nil {
		t.Fatal(err)
	}
	if err := checkHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("checkHealth(recovered primary) dispatches = %d, want 2", calls)
	}
}

func TestHealthClaimOnlySucceedsOnce(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncOK}); err != nil {
		t.Fatal(err)
	}
	claimed, err := repository.ClaimCFPointingMemberHealthSync(ctx, memberID)
	if err != nil || !claimed {
		t.Fatalf("ClaimCFPointingMemberHealthSync(first) = %v, %v, want true, nil", claimed, err)
	}
	claimed, err = repository.ClaimCFPointingMemberHealthSync(ctx, memberID)
	if err != nil || claimed {
		t.Errorf("ClaimCFPointingMemberHealthSync(second) = %v, %v, want false, nil", claimed, err)
	}
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncOK}); err != nil {
		t.Fatal(err)
	}
	claimed, err = repository.ClaimCFPointingMemberHealthSync(ctx, memberID)
	if err != nil || !claimed {
		t.Errorf("ClaimCFPointingMemberHealthSync(recovered) = %v, %v, want true, nil", claimed, err)
	}
}

func TestMemberDispatchFailurePreservesRetryableStatus(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncPending}); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("queue unavailable")
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		if calls == 1 {
			return "", wantErr
		}
		return "retry-task", nil
	}))
	if _, err := DispatchMemberSync(ctx, memberID, "test"); !errors.Is(err, wantErr) {
		t.Fatalf("DispatchMemberSync(queue failure) error = %v, want %v", err, wantErr)
	}
	if _, err := DispatchMemberSync(ctx, memberID, "test_retry"); err != nil {
		t.Errorf("DispatchMemberSync(retry after queue failure) error = %v, want nil", err)
	}
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if member.SyncStatus != model.CFMemberSyncPending {
		t.Errorf("member.SyncStatus after failed dispatch = %q, want unchanged %q", member.SyncStatus, model.CFMemberSyncPending)
	}
}

func TestHealthClaimKeepsPendingStateForWorkerLifecycle(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncOK}); err != nil {
		t.Fatal(err)
	}
	claimed, err := repository.ClaimCFPointingMemberHealthSync(ctx, memberID)
	if err != nil || !claimed {
		t.Fatalf("ClaimCFPointingMemberHealthSync() = %v, %v, want true, nil", claimed, err)
	}
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if member.SyncStatus != model.CFMemberSyncPending {
		t.Errorf("member.SyncStatus after health claim = %q, want %q", member.SyncStatus, model.CFMemberSyncPending)
	}
}

func TestHealthClaimSkipsConcurrentWorkerStates(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	for _, status := range []string{model.CFMemberSyncPending, model.CFMemberSyncing, model.CFMemberSyncError} {
		if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": status}); err != nil {
			t.Fatal(err)
		}
		claimed, err := repository.ClaimCFPointingMemberHealthSync(ctx, memberID)
		if err != nil || claimed {
			t.Errorf("ClaimCFPointingMemberHealthSync(status=%q) = %v, %v, want false, nil", status, claimed, err)
		}
	}
}

func TestHealthDispatchFailureDoesNotBlockOtherMembers(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	member.SyncStatus = model.CFMemberSyncOK
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"sync_status": model.CFMemberSyncOK}); err != nil {
		t.Fatal(err)
	}
	other := *member
	other.ID = 0
	domain := model.ZoneDomain{ZoneID: 1, Domain: "other.example.com"}
	if err := repository.DB(ctx).Create(&domain).Error; err != nil {
		t.Fatal(err)
	}
	other.ZoneDomainID = domain.ID
	if err := repository.DB(ctx).Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("first member queue failure")
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		if calls == 1 {
			return "", wantErr
		}
		return "task", nil
	}))
	if err := checkHealth(ctx); !errors.Is(err, wantErr) {
		t.Errorf("checkHealth(first member failure) = %v, want queue error", err)
	}
	if calls != 2 {
		t.Errorf("checkHealth(first member failure) dispatches = %d, want 2", calls)
	}
}
