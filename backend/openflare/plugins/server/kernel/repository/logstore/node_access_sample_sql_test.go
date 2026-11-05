package logstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
)

func TestSQLSampleMatchesHashPrefixAndBoundsDeletes(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	rows := make([]analyticsmodel.NodeAccessLog, 1200)
	for i := range rows {
		rows[i] = analyticsmodel.NodeAccessLog{ID: uint64(i + 1), NodeID: "n", LoggedAt: now, Path: fmt.Sprint(i), SummaryTracked: true}
		prepareSample(&rows[i])
	}
	if err := s.db.CreateInBatches(rows, insertBatchSize).Error; err != nil {
		t.Fatal(err)
	}
	want, used := samplePrefix(rows, 1000)
	got, boundary, err := sqlSampleBoundary(s.db, "n", now.Format(time.DateOnly), 1000, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != used {
		t.Errorf("SQL used=%d, want %d", got, used)
	}
	more, err := trimSamplePage(ctx, s.db, "n", now.Format(time.DateOnly), boundary, false)
	if err != nil {
		t.Fatal(err)
	}
	if !more {
		t.Error("first trim page more=false, want true")
	}
	var count int64
	if err := s.db.Model(&analyticsmodel.NodeAccessLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 700 {
		t.Errorf("one delete page leaves=%d, want 700", count)
	}
	if err := trimSample(ctx, s.db, "n", now.Format(time.DateOnly), boundary, false); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Model(&analyticsmodel.NodeAccessLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(want)) {
		t.Errorf("retained=%d, want %d", count, len(want))
	}
}

func TestLegacyBaselineResumesThenAccumulates(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	base := analyticsmodel.NodeAccessDaily{Day: now.Format(time.DateOnly), NodeID: "n", StatusCode: 200, Source: "legacy", RequestCount: 10, BytesSent: 100}
	if err := s.db.Create(&base).Error; err != nil {
		t.Fatal(err)
	}
	row := analyticsmodel.NodeAccessLog{ID: 1, NodeID: "n", LoggedAt: now, StatusCode: 200, BytesSent: 10}
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err != nil {
		t.Fatal(err)
	}
	row.ID = 2
	row.SummaryTracked = false
	if err := s.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.archiveDaily(ctx); err != nil {
		t.Fatal(err)
	}
	var daily analyticsmodel.NodeAccessDaily
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.RequestCount != 11 || daily.BytesSent != 110 {
		t.Errorf("baseline+late=%+v, want 11/110", daily)
	}
}
