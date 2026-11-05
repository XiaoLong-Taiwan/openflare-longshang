package logstore

import (
	"context"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
)

func TestMonthlyDeleteDoesNotTrimSamples(t *testing.T) {
	s := newTestGormStore(t)
	month := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]analyticsmodel.NodeAccessLog, 1005)
	for i := range rows {
		rows[i] = analyticsmodel.NodeAccessLog{ID: uint64(i + 1), NodeID: "n", LoggedAt: month, BytesSent: 10, SampleKey: "z", DetailSize: nodeDailyQuotaBytes}
	}
	if err := s.db.CreateInBatches(rows, insertBatchSize).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Create(&analyticsmodel.NodeAccessQuota{Day: month.Format(time.DateOnly), NodeID: "n", SampleBoundary: "a"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, want := range []int64{1000, 5, 0} {
		got, err := s.DeleteMonthBeforeBatch(context.Background(), month, month.AddDate(0, 1, 0), 1000)
		if err != nil || got != want {
			t.Fatalf("DeleteMonthBeforeBatch(over-quota)=%d/%v, want %d/nil", got, err, want)
		}
	}
	var count uint64
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count)").Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1005 {
		t.Errorf("monthly delete summary=%d, want 1005", count)
	}
}

func TestClearMigrationTargetResetsIndexAndPreservesSummary(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: now, SummaryTracked: true}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Create(&analyticsmodel.NodeAccessCheckpoint{Name: "ch-detail-index", ThroughID: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if err := addDaily(s.db, dailyGroups([]analyticsmodel.NodeAccessLog{row}, "ingest")); err != nil {
		t.Fatal(err)
	}
	s.skipFreeze = true
	if _, err := s.DeleteAll(ctx); err != nil {
		t.Fatal(err)
	}
	analyticsrepo.SetChConnForTest(&outboxCH{legacy: []analyticsmodel.NodeAccessLog{row}})
	t.Cleanup(func() { analyticsrepo.SetChConnForTest(nil) })
	for range 2 {
		if err := importClickHouseIndex(ctx, s.db); err != nil {
			t.Fatal(err)
		}
	}
	var details, requests int64
	if err := s.db.Model(&analyticsmodel.NodeAccessLog{}).Count(&details).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count)").Scan(&requests).Error; err != nil {
		t.Fatal(err)
	}
	if details != 1 || requests != 1 {
		t.Errorf("recover migration shadow details/summary=%d/%d, want 1/1", details, requests)
	}
}

func TestLegacySnapshotCheckpointFailureAndRetry(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	base := analyticsmodel.NodeAccessDaily{Day: now.Format(time.DateOnly), NodeID: "n", StatusCode: 200, Source: "legacy", RequestCount: 1, BytesSent: 1}
	if err := s.db.Create(&base).Error; err != nil {
		t.Fatal(err)
	}
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: now, StatusCode: 200, BytesSent: 10}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec("CREATE TRIGGER fail_checkpoint BEFORE INSERT ON of_node_access_checkpoints BEGIN SELECT RAISE(ABORT,'checkpoint failed'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err == nil {
		t.Fatal("archiveDaily(checkpoint failure)=nil, want error")
	}
	var daily analyticsmodel.NodeAccessDaily
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.BytesSent != 1 {
		t.Errorf("failed snapshot bytes=%d, want rollback to 1", daily.BytesSent)
	}
	if err := s.db.Exec("DROP TRIGGER fail_checkpoint").Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.archiveDaily(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.RequestCount != 1 || daily.BytesSent != 10 {
		t.Errorf("retried snapshot=%+v, want 1 request/10 bytes", daily)
	}
}
