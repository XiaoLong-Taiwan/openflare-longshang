package clickhouse

import (
	"os"
	"strings"
	"testing"
)

func TestConfigurableRetentionRemovesConflictingTTL(t *testing.T) {
	data, err := os.ReadFile("202611020002_configurable_retention.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(data), "-- +goose Down")
	if !ok {
		t.Fatal("configurable retention migration has no Down section")
	}
	for _, table := range []string{
		"of_node_access_logs", "of_node_metric_snapshots", "of_node_edge_health",
		"of_node_obs_frps", "of_node_obs_frpc",
	} {
		if !strings.Contains(up, "ALTER TABLE "+table+" REMOVE TTL;") {
			t.Errorf("Up does not remove conflicting TTL from %s", table)
		}
	}
	if strings.Contains(up, "MODIFY TTL") || strings.Contains(down, "TTL") {
		t.Error("migration must not introduce or restore a fixed TTL that can delete retained rows")
	}
	if got := strings.Count(up, "ALTER TABLE"); got != 5 {
		t.Errorf("Up ALTER count = %d, want 5 configurable retention tables", got)
	}
}
