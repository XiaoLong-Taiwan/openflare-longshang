package cloudflare

import (
	"context"
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestGroupTTLAffectsEverySelectedNode(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	node := model.OpenFlareNode{NodeID: "ttl-second", Name: "second", NodeType: "edge_node", IP: "203.0.113.11", Status: "online", LastSeenAt: &now}
	if err := repository.DB(ctx).Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	ttl := 600
	t.Cleanup(SetDispatchTaskForTest(func(_ context.Context, _ string, _ []byte, _ string) (string, error) { return "ttl-sync", nil }))
	group, err := UpdateGroup(ctx, state.Group.ID, GroupInput{Name: "multi", Nodes: []GroupNodeInput{{NodeID: state.Group.PrimaryNodeID}, {NodeID: node.ID}}, TTL: &ttl, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(group.Nodes) != 2 || group.TTL != ttl {
		t.Fatalf("UpdateGroup(multi) = %+v, want 2 nodes and TTL 600", group)
	}
	if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"proxied": false}); err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatal(err)
	}
	if len(fake.created) != 2 {
		t.Fatalf("created DNS records = %d, want 2", len(fake.created))
	}
	for _, record := range fake.created {
		if record.TTL != ttl {
			t.Errorf("record %s TTL = %d, want 600", record.Content, record.TTL)
		}
	}
}
