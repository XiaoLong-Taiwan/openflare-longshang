// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// DeleteAllNodeMetricSnapshots hard-deletes all node metric snapshots via TRUNCATE.
func DeleteAllNodeMetricSnapshots(ctx context.Context) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	outcome, err := truncateClickHouseTable(ctx, conn, nodeMetricSnapshotTableName())
	if err != nil {
		return 0, err
	}
	return outcome.DeletedCount, nil
}

// DeleteNodeMetricSnapshotsBefore removes expired metric partitions and cutoff rows.
func DeleteNodeMetricSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	return deleteWithExpiredPartitions(ctx, conn, nodeMetricSnapshotTableName(), "captured_at", cutoff)
}

// MaterializeNodeMetricSnapshotsTTL force-materializes table TTL and reports an honest outcome.
func MaterializeNodeMetricSnapshotsTTL(ctx context.Context) (CleanupOutcome, error) {
	conn, err := observabilityConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeMetricSnapshotTableName()
	ttlDays := TableTTLDaysNodeMetricSnapshots
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE captured_at < ?", tableName),
		[]any{cutoff},
	)
}

// DeleteAllNodeEdgeHealth truncates of_node_edge_health.
func DeleteAllNodeEdgeHealth(ctx context.Context) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	outcome, err := truncateClickHouseTable(ctx, conn, nodeEdgeHealthTableName())
	if err != nil {
		return 0, err
	}
	return outcome.DeletedCount, nil
}

// DeleteNodeEdgeHealthBefore removes expired edge-health partitions and cutoff rows.
func DeleteNodeEdgeHealthBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	return deleteWithExpiredPartitions(ctx, conn, nodeEdgeHealthTableName(), "captured_at", cutoff)
}

// MaterializeNodeEdgeHealthTTL force-materializes of_node_edge_health table TTL.
func MaterializeNodeEdgeHealthTTL(ctx context.Context) (CleanupOutcome, error) {
	conn, err := observabilityConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeEdgeHealthTableName()
	ttlDays := TableTTLDaysNodeObs
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE captured_at < ?", tableName),
		[]any{cutoff},
	)
}

// DeleteAllNodeObsFrps hard-deletes all FRPS observations via TRUNCATE.
func DeleteAllNodeObsFrps(ctx context.Context) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	outcome, err := truncateClickHouseTable(ctx, conn, nodeObsFrpsTableName())
	if err != nil {
		return 0, err
	}
	return outcome.DeletedCount, nil
}

// DeleteNodeObsFrpsBefore removes expired FRPS partitions and cutoff rows.
func DeleteNodeObsFrpsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	return deleteWithExpiredPartitions(ctx, conn, nodeObsFrpsTableName(), "captured_at", cutoff)
}

// MaterializeNodeObsFrpsTTL force-materializes table TTL and reports an honest outcome.
func MaterializeNodeObsFrpsTTL(ctx context.Context) (CleanupOutcome, error) {
	conn, err := observabilityConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeObsFrpsTableName()
	ttlDays := TableTTLDaysNodeObs
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE captured_at < ?", tableName),
		[]any{cutoff},
	)
}

// DeleteAllNodeObsFrpc hard-deletes all FRPC observations via TRUNCATE.
func DeleteAllNodeObsFrpc(ctx context.Context) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	outcome, err := truncateClickHouseTable(ctx, conn, nodeObsFrpcTableName())
	if err != nil {
		return 0, err
	}
	return outcome.DeletedCount, nil
}

// DeleteNodeObsFrpcBefore removes expired FRPC partitions and cutoff rows.
func DeleteNodeObsFrpcBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	conn, err := observabilityConn()
	if err != nil {
		return 0, err
	}
	return deleteWithExpiredPartitions(ctx, conn, nodeObsFrpcTableName(), "captured_at", cutoff)
}

func deleteWithExpiredPartitions(ctx context.Context, conn driver.Conn, table, column string, cutoff time.Time) (int64, error) {
	cutoff = cutoff.UTC()
	cutoffMonth := time.Date(cutoff.Year(), cutoff.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := conn.Query(ctx, "SELECT DISTINCT partition FROM system.parts WHERE database = currentDatabase() AND table = ? AND active", table)
	if err != nil {
		return 0, err
	}
	var partitions []string
	for rows.Next() {
		var partition string
		if err := rows.Scan(&partition); err != nil {
			rows.Close()
			return 0, err
		}
		partitions = append(partitions, partition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	var deleted int64
	for _, partition := range partitions {
		month, err := time.Parse("200601", partition)
		if err != nil || !month.Before(cutoffMonth) {
			continue
		}
		var count uint64
		if err := conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s WHERE %s >= ? AND %s < ?", table, column, column), month, month.AddDate(0, 1, 0)).Scan(&count); err != nil {
			return deleted, err
		}
		if err := conn.Exec(ctx, "ALTER TABLE "+table+" DROP PARTITION ?", partition); err != nil {
			return deleted, fmt.Errorf("drop expired partition %s.%s: %w", table, partition, err)
		}
		deleted += safeInt64Count(count)
	}
	var count uint64
	if err := conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s WHERE %s < ?", table, column), cutoff).Scan(&count); err != nil {
		return deleted, err
	}
	if count > 0 {
		if err := conn.Exec(ctx, fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s < ? SETTINGS mutations_sync = 1", table, column), cutoff); err != nil {
			return deleted, fmt.Errorf("delete %s before cutoff: %w", table, err)
		}
	}
	return deleted + safeInt64Count(count), nil
}

// MaterializeNodeObsFrpcTTL force-materializes table TTL and reports an honest outcome.
func MaterializeNodeObsFrpcTTL(ctx context.Context) (CleanupOutcome, error) {
	conn, err := observabilityConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeObsFrpcTableName()
	ttlDays := TableTTLDaysNodeObs
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE captured_at < ?", tableName),
		[]any{cutoff},
	)
}
