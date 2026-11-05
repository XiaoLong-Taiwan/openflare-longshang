package logstore

import (
	"context"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
)

func TestNodeQuotaDecimalGBBoundary(t *testing.T) {
	if nodeDailyQuotaBytes != 1_000_000_000 {
		t.Fatalf("nodeDailyQuotaBytes=%d, want 1000000000", nodeDailyQuotaBytes)
	}
	s := newTestGormStore(t)
	now := time.Now().UTC()
	rows := []analyticsmodel.NodeAccessLog{
		{ID: 1, NodeID: "n", LoggedAt: now, SampleKey: "a", DetailSize: 999_999_999, SummaryTracked: true},
		{ID: 2, NodeID: "n", LoggedAt: now, SampleKey: "b", DetailSize: 1, SummaryTracked: true},
		{ID: 3, NodeID: "n", LoggedAt: now, SampleKey: "c", DetailSize: 1, SummaryTracked: true},
	}
	if err := s.db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	used, boundary, err := sqlSampleBoundary(s.db, "n", now.Format(time.DateOnly), nodeDailyQuotaBytes, "")
	if err != nil {
		t.Fatal(err)
	}
	if used != 1_000_000_000 || boundary != "c" {
		t.Errorf("sqlSampleBoundary(1000000001 bytes)=%d/%q, want 1000000000/c", used, boundary)
	}
	if err := trimSample(context.Background(), s.db, "n", now.Format(time.DateOnly), boundary, false); err != nil {
		t.Fatal(err)
	}
	var retained []uint64
	if err := s.db.Model(&analyticsmodel.NodeAccessLog{}).Order("id").Pluck("id", &retained).Error; err != nil {
		t.Fatal(err)
	}
	if len(retained) != 2 || retained[0] != 1 || retained[1] != 2 {
		t.Errorf("trimSample(decimal GB) IDs=%v, want [1 2]", retained)
	}
}
