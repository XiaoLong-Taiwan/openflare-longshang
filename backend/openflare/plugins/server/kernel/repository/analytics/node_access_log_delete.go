// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package analytics

import (
	"context"
	"fmt"
	"time"
)

// DeleteAllNodeAccessLogs hard-deletes all node access logs via TRUNCATE.
func DeleteAllNodeAccessLogs(ctx context.Context) (int64, error) {
	conn, err := nodeAccessLogConn()
	if err != nil {
		return 0, err
	}
	outcome, err := truncateClickHouseTable(ctx, conn, nodeAccessLogTableName())
	if err != nil {
		return 0, err
	}
	return outcome.DeletedCount, nil
}

// DeleteNodeAccessLogsBefore drops fully expired monthly partitions and mutates the cutoff month.
func DeleteNodeAccessLogsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	conn, err := nodeAccessLogConn()
	if err != nil {
		return 0, err
	}
	var deleted int64
	cutoffUTC := cutoff.UTC()
	cutoffMonth := time.Date(cutoffUTC.Year(), cutoffUTC.Month(), 1, 0, 0, 0, 0, time.UTC)
	var partitions []string
	rows, err := conn.Query(ctx, "SELECT DISTINCT partition FROM system.parts WHERE database = currentDatabase() AND table = ? AND active", nodeAccessLogTableName())
	if err != nil {
		return 0, err
	}
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
	for _, partition := range partitions {
		month, err := time.Parse("200601", partition)
		if err != nil || !month.Before(cutoffMonth) {
			continue
		}
		var count uint64
		if err := conn.QueryRow(ctx, "SELECT count() FROM "+nodeAccessLogTableName()+" WHERE logged_at >= ? AND logged_at < ?", month, month.AddDate(0, 1, 0)).Scan(&count); err != nil {
			return deleted, err
		}
		if err := conn.Exec(ctx, "ALTER TABLE "+nodeAccessLogTableName()+" DROP PARTITION ?", partition); err != nil {
			return deleted, fmt.Errorf("drop expired node access log partition %s: %w", partition, err)
		}
		deleted += safeInt64Count(count)
	}
	var count uint64
	if err := conn.QueryRow(ctx, "SELECT count() FROM "+nodeAccessLogTableName()+" WHERE logged_at < ?", cutoffUTC).Scan(&count); err != nil {
		return deleted, err
	}
	if count > 0 {
		if err := conn.Exec(ctx, "ALTER TABLE "+nodeAccessLogTableName()+" DELETE WHERE logged_at < ? SETTINGS mutations_sync = 1", cutoffUTC); err != nil {
			return deleted, fmt.Errorf("delete node access logs before cutoff: %w", err)
		}
	}
	return deleted + safeInt64Count(count), nil
}

// MaterializeNodeAccessLogsTTL force-materializes table TTL and reports an honest outcome.
func MaterializeNodeAccessLogsTTL(ctx context.Context) (CleanupOutcome, error) {
	conn, err := nodeAccessLogConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeAccessLogTableName()
	ttlDays := TableTTLDaysNodeAccessLogs
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE logged_at < ?", tableName),
		[]any{cutoff},
	)
}

func DeleteNodeAccessLogsByNodeBefore(ctx context.Context, nodeID string, cutoff time.Time) (int64, error) {
	conn, err := nodeAccessLogConn()
	if err != nil {
		return 0, err
	}
	cutoff = cutoff.UTC()
	count, err := countClickHouseRows(ctx, conn, "SELECT count() FROM "+nodeAccessLogTableName()+" WHERE node_id = ? AND logged_at < ?", []any{nodeID, cutoff})
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}
	if err := conn.Exec(ctx, "ALTER TABLE "+nodeAccessLogTableName()+" DELETE WHERE node_id = ? AND logged_at < ? SETTINGS mutations_sync = 1", nodeID, cutoff); err != nil {
		return 0, fmt.Errorf("delete node access logs before cutoff: %w", err)
	}
	return count, nil
}

// MaterializeNodeAccessLogsTTLByNode materializes table-global TTL and estimates node-scoped rows past TTL.
func MaterializeNodeAccessLogsTTLByNode(ctx context.Context, nodeID string) (CleanupOutcome, error) {
	conn, err := nodeAccessLogConn()
	if err != nil {
		return CleanupOutcome{}, err
	}
	tableName := nodeAccessLogTableName()
	ttlDays := TableTTLDaysNodeAccessLogs
	cutoff := tableTTLCutoff(ttlDays, time.Now())
	return materializeExpiredByTableTTL(
		ctx,
		conn,
		tableName,
		ttlDays,
		fmt.Sprintf("SELECT count() FROM %s WHERE node_id = ? AND logged_at < ?", tableName),
		[]any{nodeID, cutoff},
	)
}
