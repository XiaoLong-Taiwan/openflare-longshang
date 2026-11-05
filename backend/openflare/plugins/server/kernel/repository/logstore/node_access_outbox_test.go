package logstore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"gorm.io/gorm"
)

type outboxCH struct {
	driver.Conn
	deleted int
	batch   outboxBatch
	fail    bool
	legacy  []analyticsmodel.NodeAccessLog
}

type legacyMockRows struct {
	driver.Rows
	rows  []analyticsmodel.NodeAccessLog
	index int
}

func (r *legacyMockRows) Next() bool   { return r.index < len(r.rows) }
func (r *legacyMockRows) Close() error { return nil }
func (r *legacyMockRows) Err() error   { return nil }
func (r *legacyMockRows) Scan(dest ...any) error {
	row := r.rows[r.index]
	r.index++
	*dest[0].(*uint64) = row.ID
	*dest[1].(*string) = row.NodeID
	*dest[2].(*time.Time) = row.LoggedAt
	for _, i := range []int{3, 4, 5, 6, 7, 8} {
		*dest[i].(*string) = ""
	}
	*dest[9].(*int32) = row.StatusCode
	*dest[10].(*uint64) = row.BytesSent
	*dest[11].(*uint64) = row.RequestLength
	*dest[12].(*uint32) = 0
	*dest[13].(*time.Time) = row.CreatedAt
	*dest[14].(*bool) = row.SummaryTracked
	return nil
}
func (c *outboxCH) Query(_ context.Context, query string, args ...any) (driver.Rows, error) {
	var rows []analyticsmodel.NodeAccessLog
	after := args[0].(uint64)
	for _, row := range c.legacy {
		if row.ID > after && (!strings.Contains(query, "summary_tracked = false") || !row.SummaryTracked) {
			rows = append(rows, row)
		}
	}
	return &legacyMockRows{rows: rows}, nil
}

type outboxBatch struct {
	driver.Batch
	rows int
}

func (b *outboxBatch) Append(...any) error { b.rows++; return nil }
func (b *outboxBatch) Send() error         { return nil }
func (c *outboxCH) Exec(_ context.Context, query string, _ ...any) error {
	c.deleted++
	if strings.Contains(query, "UPDATE summary_tracked") {
		for i := range c.legacy {
			c.legacy[i].SummaryTracked = true
		}
	}
	return nil
}
func (c *outboxCH) PrepareBatch(context.Context, string, ...driver.PrepareBatchOption) (driver.Batch, error) {
	if c.fail {
		return nil, errors.New("offline")
	}
	return &c.batch, nil
}

func TestClickHouseWriteRecoversOutboxAfterFailure(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)
	s := newTestGormStore(t)
	ctx := context.Background()
	SetDBResolver(func(context.Context) *gorm.DB { return s.db })
	t.Cleanup(func() { SetDBResolver(nil) })
	mock := &outboxCH{fail: true}
	analyticsrepo.SetChConnForTest(mock)
	t.Cleanup(func() { analyticsrepo.SetChConnForTest(nil) })
	ch := newClickHouseStore()
	rows := []analyticsmodel.NodeAccessLog{{ID: 100, NodeID: "n", LoggedAt: time.Now().UTC(), BytesSent: 50}}
	if err := ch.BatchInsertNodeAccessLogs(ctx, rows); err == nil {
		t.Fatal("offline CH write=nil, want error with retained outbox")
	}
	var pending int64
	if err := s.db.Model(&analyticsmodel.NodeAccessOutbox{}).Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if pending != 1 {
		t.Fatalf("offline pending=%d, want 1", pending)
	}
	mock.fail = false
	if err := ch.BatchInsertNodeAccessLogs(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if mock.batch.rows != 1 {
		t.Errorf("CH appended=%d, want 1", mock.batch.rows)
	}
	var requests uint64
	if err := s.db.Model(&analyticsmodel.NodeAccessDaily{}).Select("SUM(request_count)").Scan(&requests).Error; err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Errorf("retry requests=%d, want 1", requests)
	}
	if err := s.db.Model(&analyticsmodel.NodeAccessOutbox{}).Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Errorf("drained pending=%d, want 0", pending)
	}
}

func TestOutboxRetryAfterDeliveryBeforeAcknowledge(t *testing.T) {
	s := newTestGormStore(t)
	ctx := context.Background()
	if err := enqueueNodeDelivery(s.db, nodeDelivery{DeleteIDs: []uint64{1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec("CREATE TRIGGER fail_ack BEFORE DELETE ON of_node_access_outbox BEGIN SELECT RAISE(ABORT,'ack failed'); END").Error; err != nil {
		t.Fatal(err)
	}
	calls := 0
	deliver := func(context.Context, nodeDelivery) error { calls++; return nil }
	if err := flushNodeOutbox(ctx, s.db, deliver); err == nil {
		t.Fatal("ack failure=nil, want error")
	}
	if err := s.db.Exec("DROP TRIGGER fail_ack").Error; err != nil {
		t.Fatal(err)
	}
	if err := flushNodeOutbox(ctx, s.db, deliver); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("delivery attempts=%d, want 2", calls)
	}
}

func TestClickHouseHistoricalCheckpointAndRepeat(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)
	s := newTestGormStore(t)
	ctx := context.Background()
	SetDBResolver(func(context.Context) *gorm.DB { return s.db })
	t.Cleanup(func() { SetDBResolver(nil) })
	mock := &outboxCH{legacy: []analyticsmodel.NodeAccessLog{{ID: 1, NodeID: "n", LoggedAt: time.Now().UTC(), StatusCode: 200, BytesSent: 77}}}
	analyticsrepo.SetChConnForTest(mock)
	t.Cleanup(func() { analyticsrepo.SetChConnForTest(nil) })
	for range 2 {
		if err := newClickHouseStore().archiveDaily(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var daily analyticsmodel.NodeAccessDaily
	if err := s.db.First(&daily).Error; err != nil {
		t.Fatal(err)
	}
	if daily.RequestCount != 1 || daily.BytesSent != 77 {
		t.Errorf("CH historical daily=%+v, want 1/77", daily)
	}
	if !mock.legacy[0].SummaryTracked {
		t.Error("CH checkpoint flag=false, want true")
	}
}
