package logstore

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"Wavelet/openflare/plugins/server/kernel/runtimeconfig"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type storageStatsConn struct {
	driver.Conn
	query string
	args  []any
	err   error
}

func (c *storageStatsConn) QueryRow(_ context.Context, query string, args ...any) driver.Row {
	c.query, c.args = query, args
	return storageStatsRow{err: c.err}
}

type storageStatsRow struct {
	driver.Row
	err error
}

func (r storageStatsRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*uint64) = 7
	*dest[1].(*uint64) = 1024
	return nil
}

func TestLogStorageTableStatsScopesConfiguredClickHouseDatabase(t *testing.T) {
	original := runtimeconfig.Get()
	cfg := original
	cfg.ClickHouse.Database = "logs_stats_test"
	runtimeconfig.Set(cfg)
	t.Cleanup(func() { runtimeconfig.Set(original) })
	conn := &storageStatsConn{}
	analyticsrepo.SetChConnForTest(conn)
	t.Cleanup(func() { analyticsrepo.SetChConnForTest(nil) })
	rows, bytes, err := logStorageTableStats(t.Context(), dbNameClickHouse, "of_node_access_logs")
	if err != nil || rows != 7 || bytes != 1024 {
		t.Errorf("logStorageTableStats(clickhouse) = %d, %d, %v, want 7, 1024, nil", rows, bytes, err)
	}
	if !strings.Contains(conn.query, "WHERE active AND database = ? AND table = ?") || !slices.Equal(conn.args, []any{"logs_stats_test", "of_node_access_logs"}) {
		t.Errorf("logStorageTableStats query = %q, args = %v, want configured database and table predicates", conn.query, conn.args)
	}
	conn.err = errors.New("scan failed")
	if _, _, err := logStorageTableStats(t.Context(), dbNameClickHouse, "of_node_access_logs"); !errors.Is(err, conn.err) {
		t.Errorf("logStorageTableStats(scan failure) error = %v, want %v", err, conn.err)
	}
}
