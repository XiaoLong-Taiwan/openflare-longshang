package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"

	"gorm.io/gorm"
)

func TestMissingGroupNodesPreserveIDsInListAndDetail(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	group := state.Group
	backup := uint(902)
	group.PrimaryNodeID, group.ActiveNodeID, group.BackupNodeID = 900, 901, &backup
	if err := repository.SaveCFPointingGroupWithNodes(ctx, &group, []model.CFPointingGroupNode{{NodeID: 903, Priority: 2}}); err != nil {
		t.Fatal(err)
	}
	groups, err := ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups(missing nodes) error = %v, want nil", err)
	}
	detail, err := GetGroup(ctx, group.ID)
	if err != nil {
		t.Fatalf("GetGroup(%d) error = %v, want nil", group.ID, err)
	}
	for _, item := range []GroupItem{groups[0], detail.Group} {
		if item.PrimaryNode.ID != 900 || item.ActiveNode.ID != 901 || item.BackupNode == nil || item.BackupNode.ID != 902 || len(item.Nodes) != 1 || item.Nodes[0].ID != 903 || item.Nodes[0].Priority != 2 {
			t.Errorf("group with missing nodes = %+v, want IDs 900/901/902/903 and priority 2", item)
		}
	}
}

func TestMissingGroupNodeDoesNotHideDatabaseFailure(t *testing.T) {
	ctx, _ := setupCloudflareLogicDB(t)
	wantErr := errors.New("node query failed")
	conn := repository.DB(ctx)
	if err := conn.Callback().Query().Before("gorm:query").Register("test:node_error", func(tx *gorm.DB) {
		if tx.Statement.Table == "of_nodes" {
			tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ListGroups(ctx); !errors.Is(err, wantErr) {
		t.Errorf("ListGroups(database failure) error = %v, want %v", err, wantErr)
	}
}

func TestBatchMoveDeduplicatesCleanupPersistenceAndDispatch(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	target := model.CFPointingGroup{Name: "target"}
	if err := repository.CreateCFPointingGroup(ctx, &target); err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveCFPointingManagedRecord(ctx, &model.CFPointingManagedRecord{MemberID: memberID, NodeID: 1, CFRecordID: "old"}); err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	calls := 0
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) {
		calls++
		return fmt.Sprintf("task-%d", calls), nil
	}))
	result, err := MoveMembers(ctx, member.GroupID, MemberBatchMoveInput{MemberIDs: []uint{memberID, memberID}, TargetGroupID: target.ID})
	if err != nil {
		t.Fatalf("MoveMembers(duplicate IDs) error = %v, want nil", err)
	}
	if len(result.MemberIDs) != 1 || len(result.TaskIDs) != 1 || calls != 1 || len(fake.deleted) != 1 {
		t.Errorf("MoveMembers(duplicate IDs) = %+v, dispatch=%d deleted=%v, want one each", result, calls, fake.deleted)
	}
	if _, err := MoveMembers(ctx, member.GroupID, MemberBatchMoveInput{MemberIDs: []uint{memberID}, TargetGroupID: target.ID}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("MoveMembers(replayed request) error = %v, want record not found", err)
	}
	if calls != 1 {
		t.Errorf("MoveMembers(replayed request) dispatch=%d, want 1", calls)
	}
}

func TestBatchRemoveDeduplicatesCleanupAndResult(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	member, err := repository.GetCFPointingMemberByID(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveCFPointingManagedRecord(ctx, &model.CFPointingManagedRecord{MemberID: memberID, NodeID: 1, CFRecordID: "old"}); err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	result, err := RemoveMembers(ctx, member.GroupID, []uint{memberID, memberID})
	if err != nil {
		t.Fatalf("RemoveMembers(duplicate IDs) error = %v, want nil", err)
	}
	if len(result.MemberIDs) != 1 || len(fake.deleted) != 1 {
		t.Errorf("RemoveMembers(duplicate IDs) = %+v deleted=%v, want one each", result, fake.deleted)
	}
}

func TestCustomTargetValidation(t *testing.T) {
	for _, tt := range []struct {
		kind, content string
		valid         bool
	}{
		{kind: "A", content: "192.0.2.1", valid: true},
		{kind: "AAAA", content: "2001:db8::1", valid: true},
		{kind: "CNAME", content: "Origin.Example.com.", valid: true},
		{kind: "CNAME", content: "https://example.com", valid: false},
		{kind: "CNAME", content: "192.0.2.1", valid: false},
		{kind: "CNAME", content: "*.example.com", valid: false},
		{kind: "CNAME", content: strings.Repeat("a", 64) + ".example.com", valid: false},
		{kind: "A", content: "2001:db8::1", valid: false},
		{kind: "AAAA", content: "192.0.2.1", valid: false},
		{kind: "TXT", content: "example.com", valid: false},
	} {
		t.Run(tt.kind+"_"+tt.content, func(t *testing.T) {
			if got := validCustomRecordContent(tt.kind, tt.content); got != tt.valid {
				t.Errorf("validCustomRecordContent(%q, %q) = %v, want %v", tt.kind, tt.content, got, tt.valid)
			}
		})
	}
}

func TestCustomTargetsReconcileWithoutNodes(t *testing.T) {
	for _, tt := range []struct{ kind, content string }{
		{kind: "A", content: "192.0.2.20"},
		{kind: "AAAA", content: "2001:db8::20"},
		{kind: "CNAME", content: strings.Repeat("a", 60) + ".origin.example.net"},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			ctx, memberID := setupCloudflareLogicDB(t)
			member, err := repository.GetCFPointingMemberByID(ctx, memberID)
			if err != nil {
				t.Fatal(err)
			}
			ttl := 600
			group, err := UpdateGroup(ctx, member.GroupID, GroupInput{Name: "custom", TargetMode: "custom", RecordType: tt.kind, RecordContent: tt.content, TTL: &ttl})
			if err != nil {
				t.Fatalf("UpdateGroup(custom) error = %v, want nil", err)
			}
			if len(group.Nodes) != 0 || group.PrimaryNode.ID != 0 {
				t.Errorf("UpdateGroup(custom) = %+v, want no nodes", group)
			}
			if err := repository.DB(ctx).Model(&model.CFPointingGroup{}).Where("id = ?", group.ID).Update("enabled", true).Error; err != nil {
				t.Fatal(err)
			}
			if err := repository.UpdateCFPointingMemberColumns(ctx, memberID, map[string]any{"proxied": false}); err != nil {
				t.Fatal(err)
			}
			fake := &fakeClient{}
			t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
			if err := ReconcileMember(ctx, memberID); err != nil {
				t.Fatalf("ReconcileMember(%s) error = %v, want nil", tt.kind, err)
			}
			if len(fake.created) != 1 || fake.created[0].Type != tt.kind || fake.created[0].Content != tt.content || fake.created[0].TTL != ttl {
				t.Fatalf("ReconcileMember(%s) created = %+v, want content %q TTL 600", tt.kind, fake.created, tt.content)
			}
			records, err := repository.ListCFPointingManagedRecords(ctx, memberID)
			if err != nil || len(records) != 1 || records[0].NodeID != 0 || records[0].DesiredIP != tt.content {
				t.Fatalf("managed custom records = %+v, %v, want node 0 content %q", records, err, tt.content)
			}
			if err := DeleteManagedRecord(ctx, memberID); err != nil {
				t.Fatalf("DeleteManagedRecord(%s) error = %v, want nil", tt.kind, err)
			}
			if len(fake.deleted) != 1 {
				t.Errorf("DeleteManagedRecord(%s) deleted = %v, want one", tt.kind, fake.deleted)
			}
		})
	}
}

func TestMissingPriorityNodeAllowsHealthyFallback(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.SaveCFPointingGroupWithNodes(ctx, &state.Group, []model.CFPointingGroupNode{{NodeID: 999, Priority: 0}, {NodeID: state.Nodes[0].Node.ID, Priority: 1}}); err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatalf("ReconcileMember(missing priority node) error = %v, want nil", err)
	}
	if len(fake.created) != 1 || fake.created[0].Content != state.Nodes[0].Node.IP {
		t.Errorf("fallback created = %+v, want healthy node %s", fake.created, state.Nodes[0].Node.IP)
	}
}

type existingRecordClient struct {
	fakeClient
	events []string
}

func (client *existingRecordClient) CreateARecord(ctx context.Context, zoneID string, input RecordInput) (*DNSRecord, error) {
	client.events = append(client.events, "create:"+input.Type)
	return client.fakeClient.CreateARecord(ctx, zoneID, input)
}

func (client *existingRecordClient) DeleteRecord(ctx context.Context, zoneID, id string) error {
	client.events = append(client.events, "delete:"+id)
	return client.fakeClient.DeleteRecord(ctx, zoneID, id)
}

func (client *existingRecordClient) GetRecord(_ context.Context, _, id string) (*DNSRecord, error) {
	return &DNSRecord{ID: id}, nil
}

func TestCustomTargetSwitchPreservesManagedOwnership(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	state, err := repository.GetCFPointingMemberContext(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	fake := &existingRecordClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	t.Cleanup(SetDispatchTaskForTest(func(context.Context, string, []byte, string) (string, error) { return "sync", nil }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateGroup(ctx, state.Group.ID, GroupInput{Name: "custom", TargetMode: "custom", RecordType: "CNAME", RecordContent: "origin.example.net", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatalf("node to CNAME reconcile error = %v, want nil", err)
	}
	if len(fake.deleted) != 1 || len(fake.created) != 2 || fake.created[1].Type != "CNAME" {
		t.Fatalf("node to CNAME deleted=%v created=%+v, want old node removed and CNAME created", fake.deleted, fake.created)
	}
	if strings.Join(fake.events, ",") != "create:A,delete:record-created-1,create:CNAME" {
		t.Errorf("node to CNAME events = %v, want deletion before CNAME creation", fake.events)
	}
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatalf("custom repeat reconcile error = %v, want nil", err)
	}
	if len(fake.created) != 2 || fake.updated == nil || fake.updated.Type != "CNAME" {
		t.Errorf("repeat custom reconcile created=%+v updated=%+v, want update only", fake.created, fake.updated)
	}
	if _, err := UpdateGroup(ctx, state.Group.ID, GroupInput{Name: "nodes", TargetMode: "node", Nodes: []GroupNodeInput{{NodeID: state.Nodes[0].Node.ID, Priority: 0}}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatalf("CNAME to node reconcile error = %v, want nil", err)
	}
	if len(fake.deleted) != 2 || len(fake.created) != 3 || fake.created[2].Type != "A" {
		t.Errorf("CNAME to node deleted=%v created=%+v, want custom removed and node A created", fake.deleted, fake.created)
	}
	records, err := repository.ListCFPointingManagedRecords(ctx, memberID)
	if err != nil || len(records) != 1 || records[0].NodeID != state.Nodes[0].Node.ID {
		t.Errorf("managed after mode switches = %+v, %v, want original node only", records, err)
	}
}

func TestNodeReplacementCreatesBeforeCleanup(t *testing.T) {
	ctx, memberID := setupCloudflareLogicDB(t)
	if err := repository.SaveCFPointingManagedRecord(ctx, &model.CFPointingManagedRecord{MemberID: memberID, NodeID: 999, CFRecordID: "old-node"}); err != nil {
		t.Fatal(err)
	}
	fake := &existingRecordClient{}
	t.Cleanup(SetClientFactoryForTest(func(string) Client { return fake }))
	if err := ReconcileMember(ctx, memberID); err != nil {
		t.Fatalf("ReconcileMember(node replacement) error = %v, want nil", err)
	}
	if strings.Join(fake.events, ",") != "create:A,delete:old-node" {
		t.Errorf("node replacement events = %v, want create before delete", fake.events)
	}
}
