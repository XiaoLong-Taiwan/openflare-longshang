package logstore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresLatestMetricSnapshotsSQL(t *testing.T) {
	var output bytes.Buffer
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=test dbname=test", PreferSimpleProtocol: true}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true,
		Logger: logger.New(testLogCaptureWriter{&output}, logger.Config{LogLevel: logger.Info}),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = newGormStore(db).ListLatestMetricSnapshots(context.Background(), "node-a", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err != nil && !errors.Is(err, gorm.ErrDryRunModeUnsupported) {
		t.Fatalf("ListLatestMetricSnapshots(PostgreSQL dry run) error = %v, want dry-run unsupported or nil", err)
	}
	sql := output.String()
	for _, fragment := range []string{"ROW_NUMBER() OVER (PARTITION BY node_id ORDER BY captured_at DESC, id DESC)", "node_id = 'node-a'", "captured_at >=", "node_id <> ''", "snapshot_rank = 1"} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("ListLatestMetricSnapshots(PostgreSQL) SQL = %s, want fragment %q", sql, fragment)
		}
	}
	if strings.Contains(sql, "LIMIT") {
		t.Errorf("ListLatestMetricSnapshots(PostgreSQL) SQL = %s, want no global LIMIT", sql)
	}
}

func TestSummarizeNodeAccessDailyAcrossDaysAndSources(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)
	s := newTestGormStore(t)
	SetDBResolver(func(context.Context) *gorm.DB { return s.db })
	t.Cleanup(func() { SetDBResolver(nil) })
	SetConfigReader(func(context.Context, string) (string, error) { return "", nil })
	ctx := context.Background()
	day := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rows := []analyticsmodel.NodeAccessDaily{
		{Day: "2026-11-01", NodeID: "a", StatusCode: 503, Source: "ingest", RequestCount: 2, BytesSent: 20, RequestLength: 4},
		{Day: "2026-11-01", NodeID: "a", StatusCode: 503, Source: "legacy", RequestCount: 3, BytesSent: 30, RequestLength: 6},
		{Day: "2026-11-02", NodeID: "a", StatusCode: 503, Source: "ingest", RequestCount: 4, BytesSent: 40, RequestLength: 8},
		{Day: "2026-11-02", NodeID: "a", StatusCode: 200, Source: "ingest", RequestCount: 7},
		{Day: "2026-11-02", NodeID: "b", StatusCode: 503, Source: "ingest", RequestCount: 8},
		{Day: "2026-11-03", NodeID: "a", StatusCode: 503, Source: "ingest", RequestCount: 16},
	}
	if err := s.db.Create(rows).Error; err != nil {
		t.Fatal(err)
	}
	query := model.OpenFlareAccessLogQuery{NodeID: "a", Since: day, Until: day.Add(48*time.Hour - time.Nanosecond), StatusCode: 503}
	for range 2 {
		got, err := SummarizeNodeAccessDaily(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("SummarizeNodeAccessDaily(%+v) = %+v, want 1 group", query, got)
		}
		if got[0].RequestCount != 9 || got[0].BytesSent != 90 || got[0].RequestLength != 18 || got[0].NodeID != "a" || got[0].StatusCode != 503 {
			t.Errorf("SummarizeNodeAccessDaily(%+v) = %+v, want a/503: 9 requests, 90 sent, 18 received", query, got[0])
		}
	}
	daily, err := ListNodeAccessDaily(ctx, "a", query.Since, query.Until)
	if err != nil {
		t.Fatal(err)
	}
	if len(daily) != 3 || daily[0].Day != "2026-11-01" || daily[0].RequestCount != 5 {
		t.Errorf("ListNodeAccessDaily(a, two days) = %+v, want 3 dated groups with first count 5", daily)
	}
}
