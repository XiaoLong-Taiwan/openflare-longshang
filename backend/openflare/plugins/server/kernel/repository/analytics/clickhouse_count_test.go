// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package analytics

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/runtimeconfig"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type capacityTestConn struct {
	mockConn
	row   driver.Row
	query string
	args  []any
}

func (c *capacityTestConn) QueryRow(_ context.Context, query string, args ...any) driver.Row {
	c.query, c.args = query, args
	return c.row
}

type capacityTestRow struct {
	driver.Row
	values [3]uint64
	err    error
}

func (r capacityTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		*dest[i].(*uint64) = value
	}
	return nil
}

func TestGetClickHouseCapacity(t *testing.T) {
	original := runtimeconfig.Get()
	config := original
	config.ClickHouse.Database = "capacity_test"
	runtimeconfig.Set(config)
	t.Cleanup(func() { runtimeconfig.Set(original) })
	scanErr := errors.New("capacity scan failed")
	for _, tt := range []struct {
		name string
		row  capacityTestRow
		want ClickHouseCapacity
	}{
		{name: "measured", row: capacityTestRow{values: [3]uint64{3, 2500, 1048576}}, want: ClickHouseCapacity{ActiveParts: 3, TotalRows: 2500, BytesOnDisk: 1048576}},
		{name: "confirmed zero"},
		{name: "scan error", row: capacityTestRow{err: scanErr}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conn := &capacityTestConn{row: tt.row}
			SetChConnForTest(conn)
			t.Cleanup(func() { SetChConnForTest(nil) })

			got, err := GetClickHouseCapacity(t.Context())
			if !strings.Contains(conn.query, "FROM system.parts\nWHERE active AND database = ?") || !slices.Equal(conn.args, []any{"capacity_test"}) {
				t.Errorf("GetClickHouseCapacity query = %q, args = %v, want active parts scoped to capacity_test", conn.query, conn.args)
			}
			if !errors.Is(err, tt.row.err) {
				t.Errorf("GetClickHouseCapacity error = %v, want %v", err, tt.row.err)
			}
			if tt.row.err != nil {
				if got != nil {
					t.Errorf("GetClickHouseCapacity on scan failure = %+v, want nil", got)
				}
			} else if got == nil || *got != tt.want {
				t.Errorf("GetClickHouseCapacity = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSafeInt64Count(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count uint64
		want  int64
	}{
		{name: "zero", count: 0, want: 0},
		{name: "small", count: 42, want: 42},
		{name: "max int64", count: math.MaxInt64, want: math.MaxInt64},
		{name: "overflow clamps", count: math.MaxUint64, want: math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := safeInt64Count(tt.count); got != tt.want {
				t.Fatalf("safeInt64Count(%d) = %d, want %d", tt.count, got, tt.want)
			}
		})
	}
}
