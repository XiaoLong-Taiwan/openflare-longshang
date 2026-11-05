package logstore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"gorm.io/gorm"
)

func TestNodeDailyQuotaAndRetry(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	day := now.Format(time.DateOnly)
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n1", LoggedAt: now, StatusCode: 200, BytesSent: 123, RequestLength: 45}
	size := detailBytes(row)
	if err := s.db.Create(&analyticsmodel.NodeAccessQuota{Day: day, NodeID: "n1", UsedBytes: nodeDailyQuotaBytes - size}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.BatchInsertNodeAccessLogs(ctx, []analyticsmodel.NodeAccessLog{row}); err != nil {
		t.Fatal(err)
	}
	row.ID = 2
	row.StatusCode = 503
	if err := s.BatchInsertNodeAccessLogs(ctx, []analyticsmodel.NodeAccessLog{row}); err != nil {
		t.Fatal(err)
	}
	if err := newGormStore(s.db).BatchInsertNodeAccessLogs(ctx, []analyticsmodel.NodeAccessLog{row}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := s.db.Model(&analyticsmodel.NodeAccessLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("detail count = %d, want 2 after repairing stale quota", count)
	}
	var totals struct {
		Requests uint64
		Sent     uint64
		Received uint64
	}
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count) AS requests, SUM(bytes_sent) AS sent, SUM(request_length) AS received").Scan(&totals).Error; err != nil {
		t.Fatal(err)
	}
	if totals.Requests != 2 || totals.Sent != 246 || totals.Received != 90 {
		t.Errorf("daily totals = %+v, want requests=2 sent=246 received=90", totals)
	}
	var quota analyticsmodel.NodeAccessQuota
	if err := s.db.First(&quota).Error; err != nil {
		t.Fatal(err)
	}
	if quota.UsedBytes != 2*size {
		t.Errorf("used bytes = %d, want %d from actual details", quota.UsedBytes, 2*size)
	}
}

func TestSamplePrefixOrderAndBatchIndependence(t *testing.T) {
	var rows []analyticsmodel.NodeAccessLog
	for i := 1; i <= 40; i++ {
		rows = append(rows, analyticsmodel.NodeAccessLog{ID: uint64(i), NodeID: "n", Path: strings.Repeat("x", i), LoggedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)})
	}
	want, used := samplePrefix(append([]analyticsmodel.NodeAccessLog(nil), rows...), 600)
	var incremental []analyticsmodel.NodeAccessLog
	boundary := ""
	for i := len(rows) - 1; i >= 0; i-- {
		incremental, _, boundary = sampleWithBoundary(append(incremental, rows[i]), 600, boundary)
	}
	if !reflect.DeepEqual(want, incremental) {
		t.Errorf("samplePrefix reversed single batches = %v, want %v", incremental, want)
	}
	if used > 600 || len(want) == 0 || len(want) == len(rows) {
		t.Errorf("samplePrefix used=%d count=%d, want nonempty partial sample <=600", used, len(want))
	}
	changed := rows[0]
	changed.ID = 999
	changed.CreatedAt = time.Now()
	changed.SummaryTracked = true
	if sampleHash(changed) != sampleHash(rows[0]) {
		t.Error("sampleHash changed after persistence metadata, want stable hash")
	}
}

func TestGroupedAccountingSQLCount(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	var statements int
	if err := s.db.Callback().Create().Before("gorm:create").Register("count_accounting", func(tx *gorm.DB) {
		if tx.Statement.Table == "of_node_access_daily" || tx.Statement.Table == "of_node_access_quota" {
			statements++
		}
	}); err != nil {
		t.Fatal(err)
	}
	rows := make([]analyticsmodel.NodeAccessLog, 400)
	for i := range rows {
		rows[i] = analyticsmodel.NodeAccessLog{ID: uint64(i + 1), NodeID: "n", LoggedAt: time.Now().UTC(), StatusCode: int32(200 + i%2)}
	}
	if err := s.BatchInsertNodeAccessLogs(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if statements != 2 {
		t.Errorf("quota/summary create SQL=%d for 400 rows, want 2", statements)
	}
}

func TestExistingOverQuotaRepairPreservesSummary(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	rows := make([]analyticsmodel.NodeAccessLog, 20)
	for i := range rows {
		rows[i] = analyticsmodel.NodeAccessLog{ID: uint64(i + 1), NodeID: "n", LoggedAt: time.Now().UTC(), BytesSent: 10, Path: strings.Repeat("x", i)}
	}
	if err := s.db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.maintainNodeAccountingWithBudget(ctx, 300); err != nil {
			t.Fatal(err)
		}
	}
	var quota analyticsmodel.NodeAccessQuota
	if err := s.db.First(&quota).Error; err != nil {
		t.Fatal(err)
	}
	if quota.UsedBytes > 300 || quota.SampleBoundary == "" {
		t.Errorf("repaired quota=%+v, want <=300 and persistent boundary", quota)
	}
	var daily analyticsmodel.NodeAccessDaily
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.RequestCount != 20 || daily.BytesSent != 200 {
		t.Errorf("repaired summary=%+v, want 20 requests/200 bytes", daily)
	}
}

func TestArchiveUsesPhysicalRowsForDuplicateIDs(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	if err := s.db.Exec("CREATE TABLE duplicate_access (id BIGINT,node_id TEXT,logged_at DATETIME,bytes_sent BIGINT,request_length BIGINT,status_code INTEGER,summary_tracked BOOLEAN,created_at DATETIME,remote_addr TEXT,region TEXT,host TEXT,path TEXT,user_agent TEXT,cache_status TEXT,request_time_ms BIGINT,sample_key TEXT,detail_size BIGINT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec("DROP TABLE of_node_access_logs").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec("ALTER TABLE duplicate_access RENAME TO of_node_access_logs").Error; err != nil {
		t.Fatal(err)
	}
	rows := make([]analyticsmodel.NodeAccessLog, cleanupBatchSize+1)
	for i := range rows {
		rows[i] = analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: time.Now().UTC(), BytesSent: 1}
	}
	if err := s.db.CreateInBatches(rows, insertBatchSize).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err != nil {
		t.Fatal(err)
	}
	var count uint64
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count)").Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != uint64(len(rows)) {
		t.Errorf("duplicate ID archived count=%d, want %d", count, len(rows))
	}
}

func TestArchiveCheckpointAccumulatesLateRows(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), BytesSent: 10}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteBefore(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	row.ID = 2
	row.BytesSent = 20
	row.SummaryTracked = false
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.archiveDaily(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var daily analyticsmodel.NodeAccessDaily
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.RequestCount != 2 || daily.BytesSent != 30 {
		t.Errorf("late checkpoint summary=%+v, want requests=2 bytes=30", daily)
	}
}

func TestArchiveCheckpointRollback(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: time.Now()}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec("CREATE TRIGGER fail_checkpoint BEFORE UPDATE OF summary_tracked ON of_node_access_logs BEGIN SELECT RAISE(ABORT, 'checkpoint failed'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err == nil {
		t.Fatal("archive checkpoint failure=nil, want error")
	}
	var count int64
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("rolled back daily count=%d, want 0", count)
	}
	if err := s.db.Exec("DROP TRIGGER fail_checkpoint").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAccountingLedgerRetention(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	rows := []analyticsmodel.NodeAccessBatch{{BatchKey: "old", CreatedAt: now.AddDate(0, 0, -10)}, {BatchKey: "current", CreatedAt: now}}
	if err := s.db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.maintainNodeAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var remaining []analyticsmodel.NodeAccessBatch
	if err := s.db.Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].BatchKey != "current" {
		t.Errorf("retained ledger=%+v, want current only", remaining)
	}
}

func TestNodeDailyFailureRollsBack(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: time.Now().UTC()}
	err := persistNodeBatch(ctx, s.db, []analyticsmodel.NodeAccessLog{row}, func(*gorm.DB, []analyticsmodel.NodeAccessLog) error { return errors.New("write failed") })
	if err == nil {
		t.Fatal("persistNodeBatch(write failure) = nil, want error")
	}
	for _, entity := range []any{&analyticsmodel.NodeAccessBatch{}, &analyticsmodel.NodeAccessDaily{}, &analyticsmodel.NodeAccessQuota{}} {
		var count int64
		if err := s.db.Model(entity).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("rollback count(%T) = %d, want 0", entity, count)
		}
	}
}

func TestLegacyDailyBeforeBatchedDeletion(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	month := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := []analyticsmodel.NodeAccessLog{
		{ID: 1, NodeID: "n", LoggedAt: month, StatusCode: 200, BytesSent: 100},
		{ID: 2, NodeID: "n", LoggedAt: month.Add(time.Hour), StatusCode: 200, BytesSent: 200},
		{ID: 3, NodeID: "n", LoggedAt: month.Add(2 * time.Hour), StatusCode: 500, BytesSent: 300},
	}
	if err := s.db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if _, err := s.DeleteMonthBeforeBatch(ctx, month, month.AddDate(0, 1, 0), 1); err != nil {
			t.Fatal(err)
		}
	}
	var daily []analyticsmodel.NodeAccessDaily
	if err := s.db.Order("status_code").Find(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if len(daily) != 2 {
		t.Fatalf("daily groups = %d, want 2", len(daily))
	}
	if daily[0].RequestCount != 2 || daily[0].BytesSent != 300 || daily[1].RequestCount != 1 || daily[1].BytesSent != 300 {
		t.Errorf("daily after repeated deletes = %+v, want 200:2/300 500:1/300", daily)
	}
}

func TestNodeDailyIsolationAndExpiredInput(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := s.db.Create(&analyticsmodel.NodeAccessQuota{Day: now.Format(time.DateOnly), NodeID: "full", UsedBytes: nodeDailyQuotaBytes}).Error; err != nil {
		t.Fatal(err)
	}
	rows := []analyticsmodel.NodeAccessLog{
		{ID: 1, NodeID: "full", LoggedAt: now},
		{ID: 2, NodeID: "other", LoggedAt: now},
		{ID: 3, NodeID: "full", LoggedAt: now.AddDate(0, 0, -1)},
		{ID: 4, NodeID: "old", LoggedAt: now.AddDate(0, 0, -8)},
	}
	if err := s.BatchInsertNodeAccessLogs(ctx, rows); err != nil {
		t.Fatal(err)
	}
	var details []analyticsmodel.NodeAccessLog
	if err := s.db.Order("id").Find(&details).Error; err != nil {
		t.Fatal(err)
	}
	if len(details) != 3 || details[0].ID != 1 || details[1].ID != 2 || details[2].ID != 3 {
		t.Errorf("isolated detail selection = %+v, want IDs 1, 2 and 3 after quota repair", details)
	}
	var requests uint64
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count)").Scan(&requests).Error; err != nil {
		t.Fatal(err)
	}
	if requests != 4 {
		t.Errorf("daily requests = %d, want 4 including expired and over-quota records", requests)
	}
}
