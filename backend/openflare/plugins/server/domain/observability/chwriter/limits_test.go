package chwriter

import (
	"context"
	"fmt"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"Wavelet/pkg/batchwriter"
)

func TestDedupSetBound(t *testing.T) {
	set := newDedupSet()
	for i := range dedupMaxKeys {
		if !set.markIfNew(fmt.Sprint(i)) {
			t.Fatalf("markIfNew(%d) = false, want true before capacity", i)
		}
	}
	if set.markIfNew("overflow") || len(set.keys) != dedupMaxKeys {
		t.Errorf("markIfNew(overflow) keys = %d, want %d and rejection", len(set.keys), dedupMaxKeys)
	}
	set.keys["0"] = time.Now().Add(-time.Second)
	set.lastCleanup = time.Now().Add(-time.Minute)
	if !set.markIfNew("after-expiry") {
		t.Error("markIfNew(after expiry) = false, want true")
	}
}

func TestMetricSnapshotKeyMatchesStoragePrecision(t *testing.T) {
	row := analyticsmodel.NodeMetricSnapshot{NodeID: "n1", CapturedAt: time.Unix(1, 100)}
	other := row
	other.CapturedAt = other.CapturedAt.Add(time.Nanosecond)
	if metricSnapshotKey(row) != metricSnapshotKey(other) {
		t.Error("metricSnapshotKey(same millisecond) differs, want equal")
	}
	other.CapturedAt = other.CapturedAt.Add(time.Millisecond)
	if metricSnapshotKey(row) == metricSnapshotKey(other) {
		t.Error("metricSnapshotKey(next millisecond) equal, want distinct")
	}
}

func TestQueueNodeAccessLogsKeepsIdenticalBatchesAndBounds(t *testing.T) {
	oldWriter := nodeAccessLogWriter
	defer func() { nodeAccessLogWriter = oldWriter }()
	var stored []analyticsmodel.NodeAccessLog
	cfg := batchwriter.DefaultConfig()
	cfg.MaxBatchSize = 4
	cfg.MinBatchSize = 0
	writer, err := batchwriter.New(cfg, func(_ context.Context, rows []analyticsmodel.NodeAccessLog) error {
		stored = append(stored, rows...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	writer.Start(context.Background())
	nodeAccessLogWriter = writer
	row := analyticsmodel.NodeAccessLog{NodeID: "n1", LoggedAt: time.Unix(1, 0), Path: "/"}
	if !QueueNodeAccessLogs([]analyticsmodel.NodeAccessLog{row, row}) || !QueueNodeAccessLogs([]analyticsmodel.NodeAccessLog{row, row}) {
		t.Fatal("QueueNodeAccessLogs(identical batches) = false, want both accepted")
	}
	if QueueNodeAccessLogs(make([]analyticsmodel.NodeAccessLog, nodeAccessLogMaxBatchSize+1)) {
		t.Error("QueueNodeAccessLogs(oversized batch) = true, want false")
	}
	if err := writer.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 4 {
		t.Errorf("QueueNodeAccessLogs(two identical batches) stored = %d, want 4", len(stored))
	}
}

func TestQueueNodeAccessLogsRejectsWholeBatchOnBackpressure(t *testing.T) {
	oldWriter := nodeAccessLogWriter
	defer func() { nodeAccessLogWriter = oldWriter }()
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	cfg := batchwriter.DefaultConfig()
	cfg.QueueSize = 2
	cfg.MaxBatchSize = 1
	writer, err := batchwriter.New(cfg, func(context.Context, []analyticsmodel.NodeAccessLog) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	writer.Start(context.Background())
	t.Cleanup(func() {
		close(release)
		if err := writer.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	nodeAccessLogWriter = writer
	row := analyticsmodel.NodeAccessLog{NodeID: "busy", LoggedAt: time.Unix(1, 0)}
	writer.TryEnqueue(row)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("flush(first row) did not start within one second")
	}
	writer.TryEnqueue(row)
	batch := []analyticsmodel.NodeAccessLog{row, row}
	QueueNodeAccessLogs(batch)
	if writer.Len() != 1 {
		t.Errorf("QueueNodeAccessLogs(2 rows with 1 slot) depth = %d, want 1", writer.Len())
	}
	if QueueNodeAccessLogs(batch) {
		t.Error("QueueNodeAccessLogs(batch without capacity) = true, want false")
	}
	if writer.Len() != 1 {
		t.Errorf("QueueNodeAccessLogs(rejected batch) depth = %d, want 1", writer.Len())
	}
}
