package integration

import (
	"context"
	"net/http"
	"testing"

	ofnode "Wavelet/openflare/plugins/server/domain/fleet/node"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestHeartbeatV0024Compatibility(t *testing.T) {
	engine, cleanup := setupProtocolTestEnv(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	edge, err := ofnode.CreateNode(ctx, ofnode.Input{Name: "edge-v0024", IP: "203.0.113.24"})
	if err != nil {
		t.Fatalf("CreateNode(edge-v0024) = %v, want nil", err)
	}
	for _, version := range []string{"v0.0.24", "dev"} {
		for _, schema := range []int{0, 2} {
			rec := performJSONRequest(t, engine, http.MethodPost, "/api/v1/agent/nodes/heartbeat", map[string]any{
				"node_id": "untrusted-node-id", "ip": "203.0.113.24", "version": version, "schema_version": schema,
			}, map[string]string{"X-Agent-Token": edge.AccessToken})
			if rec.Code != http.StatusOK {
				t.Fatalf("Heartbeat(version=%s, schema=%d) = %d %s, want 200", version, schema, rec.Code, rec.Body.String())
			}
			stored, err := repository.GetOpenFlareNodeByNodeID(ctx, edge.NodeID)
			if err != nil {
				t.Fatalf("GetOpenFlareNodeByNodeID(%s) = %v, want nil", edge.NodeID, err)
			}
			wantVersion := version
			if version == "dev" {
				wantVersion = "v0.0.24"
			}
			if stored.Version != wantVersion || stored.Status != "online" || stored.LastSeenAt == nil {
				t.Errorf("Heartbeat(version=%s, schema=%d) stored = %s/%s/%v, want %s/online/non-nil heartbeat", version, schema, stored.Version, stored.Status, stored.LastSeenAt, wantVersion)
			}
		}
	}
	rec := performJSONRequest(t, engine, http.MethodPost, "/api/v1/agent/nodes/heartbeat", map[string]any{
		"ip": "203.0.113.24", "version": "v0.0.24",
	}, map[string]string{"X-Agent-Token": "invalid-regression-token"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Heartbeat(invalid token) = %d, want 401", rec.Code)
	}
}
