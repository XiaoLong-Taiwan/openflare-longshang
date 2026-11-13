package repository

import (
	"fmt"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
)

func TestLatestMetricSnapshotsWithoutGlobalLimit(t *testing.T) {
	ctx, cleanup := setupOpenFlareAccessLogTestEnvironment(t)
	t.Cleanup(cleanup)
	if err := DB(ctx).AutoMigrate(&analyticsmodel.NodeMetricSnapshot{}); err != nil {
		t.Fatal(err)
	}
	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rows := []analyticsmodel.NodeMetricSnapshot{
		{ID: 1, NodeID: "quiet", CapturedAt: since, StorageUsedBytes: 42},
		{ID: 2, NodeID: "expired", CapturedAt: since.Add(-time.Nanosecond)},
		{ID: 3, NodeID: "", CapturedAt: since.Add(time.Hour)},
		{ID: 999, NodeID: "busy", CapturedAt: since},
	}
	for i := range 150 {
		rows = append(rows, analyticsmodel.NodeMetricSnapshot{ID: uint64(i + 10), NodeID: "busy", CapturedAt: since.Add(time.Minute), StorageUsedBytes: int64(i)})
	}
	if err := DB(ctx).CreateInBatches(rows, 50).Error; err != nil {
		t.Fatal(err)
	}
	got, err := ListOpenFlareLatestMetricSnapshotsSince(ctx, "", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("ListOpenFlareLatestMetricSnapshotsSince(all) = %d rows, want 2", len(got))
	}
	if got[0].NodeID != "busy" || got[0].ID != 159 || got[0].StorageUsedBytes != 149 || got[1].NodeID != "quiet" || got[1].StorageUsedBytes != 42 {
		t.Errorf("ListOpenFlareLatestMetricSnapshotsSince(all) = %+v, %+v, want busy:159/149 and quiet:1/42", got[0], got[1])
	}
	filtered, err := ListOpenFlareLatestMetricSnapshotsSince(ctx, "quiet", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].ID != 1 {
		t.Errorf("ListOpenFlareLatestMetricSnapshotsSince(quiet) = %+v, want ID 1", filtered)
	}
	empty, err := ListOpenFlareLatestMetricSnapshotsSince(ctx, "", since.Add(2*time.Hour))
	if err != nil || len(empty) != 0 {
		t.Errorf("ListOpenFlareLatestMetricSnapshotsSince(after all) = %v, %v, want empty, nil", empty, err)
	}
	var nodes []analyticsmodel.NodeMetricSnapshot
	for i := range 150 {
		nodes = append(nodes, analyticsmodel.NodeMetricSnapshot{ID: uint64(i + 1000), NodeID: fmt.Sprintf("node-%03d", i), CapturedAt: since})
	}
	if err := DB(ctx).CreateInBatches(nodes, 50).Error; err != nil {
		t.Fatal(err)
	}
	got, err = ListOpenFlareLatestMetricSnapshotsSince(ctx, "", since)
	if err != nil || len(got) != 152 {
		t.Errorf("ListOpenFlareLatestMetricSnapshotsSince(152 nodes) = %d rows, %v, want 152, nil", len(got), err)
	}
}
