package state

import (
	"path/filepath"
	"testing"

	"Wavelet/openflare/plugins/agent/protocol"
)

func TestAckRecordsPreservesNewDataInSameWindow(t *testing.T) {
	store := NewObservabilityBufferStore(filepath.Join(t.TempDir(), "buffer.json"))
	sent := ObservabilityBufferRecord{WindowStartedAtUnix: 60, HostMetrics: &protocol.NodeMetricSnapshot{CapturedAtUnix: 65, CPUUsagePercent: 1}, EdgeHealth: &protocol.NodeEdgeHealth{CapturedAtUnix: 65, Connections: 1}, AccessLogs: []protocol.NodeAccessLog{{LoggedAtUnix: 65, Path: "/old"}}}
	if err := store.Upsert(sent, 0); err != nil {
		t.Fatal(err)
	}
	newData := ObservabilityBufferRecord{WindowStartedAtUnix: 60, HostMetrics: &protocol.NodeMetricSnapshot{CapturedAtUnix: 65, CPUUsagePercent: 2}, EdgeHealth: &protocol.NodeEdgeHealth{CapturedAtUnix: 65, Connections: 2}, AccessLogs: []protocol.NodeAccessLog{{LoggedAtUnix: 65, Path: "/old", BytesSent: 2}}}
	if err := store.Upsert(newData, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.AckRecords([]ObservabilityBufferRecord{sent}, 0); err != nil {
		t.Fatal(err)
	}
	got, err := store.Replayable(0, 0)
	if err != nil || len(got) != 1 {
		t.Fatalf("AckRecords(old snapshot) = %v, %v, want one updated window", got, err)
	}
	if got[0].HostMetrics == nil || *got[0].HostMetrics != *newData.HostMetrics || got[0].EdgeHealth == nil || *got[0].EdgeHealth != *newData.EdgeHealth || len(got[0].AccessLogs) != 1 || got[0].AccessLogs[0] != newData.AccessLogs[0] {
		t.Fatalf("AckRecords(old snapshot) = %+v, want new metrics, health and log retained", got[0])
	}
}

func TestAckRecordsMatchesFullLogAndMultiplicity(t *testing.T) {
	store := NewObservabilityBufferStore(filepath.Join(t.TempDir(), "buffer.json"))
	old := protocol.NodeAccessLog{LoggedAtUnix: 65, Path: "/same", BytesSent: 1}
	updated := old
	updated.BytesSent = 2
	if err := store.Upsert(ObservabilityBufferRecord{WindowStartedAtUnix: 60, AccessLogs: []protocol.NodeAccessLog{old, updated, old}}, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.AckRecords([]ObservabilityBufferRecord{{WindowStartedAtUnix: 60, AccessLogs: []protocol.NodeAccessLog{old}}}, 0); err != nil {
		t.Fatal(err)
	}
	got, err := store.Replayable(0, 0)
	if err != nil || len(got) != 1 || len(got[0].AccessLogs) != 2 || got[0].AccessLogs[0] != updated || got[0].AccessLogs[1] != old {
		t.Fatalf("AckRecords(one old log) = %v, %v, want changed log and one duplicate retained", got, err)
	}
}
