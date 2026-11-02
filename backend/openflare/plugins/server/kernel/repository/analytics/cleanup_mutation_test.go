package analytics

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type cleanupRow struct {
	count uint64
	err   error
}

func (r cleanupRow) Err() error { return r.err }
func (r cleanupRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*uint64) = r.count
	return nil
}
func (r cleanupRow) ScanStruct(any) error { return r.err }

type cleanupConn struct {
	mockConn
	counts    []cleanupRow
	execs     []string
	execArgs  [][]any
	execErr   error
	execFn    func(string) error
	countSQL  []string
	countArgs [][]any
}

func (c *cleanupConn) QueryRow(_ context.Context, query string, args ...any) driver.Row {
	c.countSQL = append(c.countSQL, query)
	c.countArgs = append(c.countArgs, args)
	row := c.counts[0]
	c.counts = c.counts[1:]
	return row
}

func (c *cleanupConn) Exec(_ context.Context, query string, args ...any) error {
	c.execs = append(c.execs, query)
	c.execArgs = append(c.execArgs, args)
	if c.execFn != nil {
		return c.execFn(query)
	}
	return c.execErr
}

func TestCleanupBeforeSynchronousMutation(t *testing.T) {
	cutoff := time.Date(2026, 11, 2, 1, 2, 3, 0, time.FixedZone("Taipei", 8*60*60))
	failure := errors.New("mutation failed")
	cleanups := []struct {
		name, table, column string
		fn                  func(context.Context, time.Time) (int64, error)
	}{
		{"node access", "of_node_access_logs", "logged_at", DeleteNodeAccessLogsBefore},
		{"metrics", "of_node_metric_snapshots", "captured_at", DeleteNodeMetricSnapshotsBefore},
		{"edge health", "of_node_edge_health", "captured_at", DeleteNodeEdgeHealthBefore},
		{"frps", "of_node_obs_frps", "captured_at", DeleteNodeObsFrpsBefore},
		{"frpc", "of_node_obs_frpc", "captured_at", DeleteNodeObsFrpcBefore},
		{"user access", "w_user_access_logs", "created_at", DeleteUserAccessLogsBefore},
	}
	for _, cleanup := range cleanups {
		t.Run(cleanup.name, func(t *testing.T) {
			for _, scenario := range []struct {
				name              string
				count             uint64
				countErr, execErr error
				want              int64
			}{
				{name: "success", count: 12, want: 12},
				{name: "empty"},
				{name: "mutation failure", count: 12, execErr: failure},
				{name: "count failure", countErr: failure},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					conn := &cleanupConn{counts: []cleanupRow{{count: scenario.count, err: scenario.countErr}}, execErr: scenario.execErr}
					SetChConnForTest(conn)
					t.Cleanup(func() { SetChConnForTest(nil) })
					got, err := cleanup.fn(context.Background(), cutoff)
					wantErr := scenario.countErr
					if wantErr == nil {
						wantErr = scenario.execErr
					}
					if got != scenario.want || !errors.Is(err, wantErr) {
						t.Errorf("cleanup(%v) = (%d, %v), want (%d, %v)", cutoff, got, err, scenario.want, wantErr)
					}
					wantSQL := "SELECT count() FROM " + cleanup.table + " WHERE " + cleanup.column + " < ?"
					if !reflect.DeepEqual(conn.countSQL, []string{wantSQL}) || !reflect.DeepEqual(conn.countArgs, [][]any{{cutoff.UTC()}}) {
						t.Errorf("count = (%v, %v), want (%s, %v)", conn.countSQL, conn.countArgs, wantSQL, cutoff.UTC())
					}
					var wantExecs []string
					if scenario.count > 0 && scenario.countErr == nil {
						wantExecs = []string{"ALTER TABLE " + cleanup.table + " DELETE WHERE " + cleanup.column + " < ? SETTINGS mutations_sync = 1"}
						if !reflect.DeepEqual(conn.execArgs, [][]any{{cutoff.UTC()}}) {
							t.Errorf("mutation args = %v, want %v", conn.execArgs, cutoff.UTC())
						}
					}
					if !reflect.DeepEqual(conn.execs, wantExecs) {
						t.Errorf("mutation SQL = %v, want %v", conn.execs, wantExecs)
					}
				})
			}
		})
	}
}

func TestCleanupBeforePreservesCompletedPartitionsOnFailure(t *testing.T) {
	failure := errors.New("boundary cleanup failed")
	for _, fn := range []func(context.Context, time.Time) (int64, error){DeleteNodeAccessLogsBefore, DeleteNodeMetricSnapshotsBefore} {
		for _, stage := range []string{"success", "count", "mutation", "partition"} {
			conn := &cleanupConn{counts: []cleanupRow{{count: 8}, {count: 5}}}
			conn.queryFn = func(context.Context, string, ...any) (driver.Rows, error) {
				return &mockRows{data: [][]any{{"202609"}, {"202611"}, {"202612"}, {"invalid"}}}, nil
			}
			if stage == "count" {
				conn.counts[1].err = failure
			}
			conn.execFn = func(query string) error {
				if stage == "mutation" && strings.Contains(query, " DELETE WHERE ") {
					return failure
				}
				if stage == "partition" {
					return failure
				}
				return nil
			}
			SetChConnForTest(conn)
			t.Cleanup(func() { SetChConnForTest(nil) })
			got, err := fn(context.Background(), time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC))
			want := int64(13)
			var wantErr error
			if stage == "count" || stage == "mutation" {
				want, wantErr = 8, failure
			}
			if stage == "partition" {
				want, wantErr = 0, failure
			}
			if got != want || !errors.Is(err, wantErr) {
				t.Errorf("cleanup(stage=%s) = (%d, %v), want (%d, %v)", stage, got, err, want, wantErr)
			}
		}
	}
}

func TestCleanupByNodeUsesCutoffAndSynchronousMutation(t *testing.T) {
	cutoff := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	failure := errors.New("mutation failed")
	for _, execErr := range []error{nil, failure} {
		conn := &cleanupConn{counts: []cleanupRow{{count: 9}}, execErr: execErr}
		SetChConnForTest(conn)
		t.Cleanup(func() { SetChConnForTest(nil) })
		got, err := DeleteNodeAccessLogsByNodeBefore(context.Background(), "node-1", cutoff)
		want := int64(9)
		if execErr != nil {
			want = 0
		}
		if got != want || !errors.Is(err, execErr) {
			t.Errorf("DeleteNodeAccessLogsByNodeBefore() = (%d, %v), want (%d, %v)", got, err, want, execErr)
		}
		wantSQL := "ALTER TABLE of_node_access_logs DELETE WHERE node_id = ? AND logged_at < ? SETTINGS mutations_sync = 1"
		if !reflect.DeepEqual(conn.execs, []string{wantSQL}) || !reflect.DeepEqual(conn.execArgs, [][]any{{"node-1", cutoff}}) {
			t.Errorf("node mutation = (%v, %v), want (%s, node-1/%v)", conn.execs, conn.execArgs, wantSQL, cutoff)
		}
	}
}
