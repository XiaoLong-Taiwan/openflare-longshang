// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package logstore

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// listPartitionNames 列出 table 在当前 schema 下的全部直接分区表名（pg_inherits）。
func listPartitionNames(ctx context.Context, gdb *gorm.DB, table string) ([]string, error) {
	var names []string
	if err := gdb.WithContext(ctx).Raw(`
SELECT c.relname
FROM pg_inherits i
JOIN pg_class c ON c.oid = i.inhrelid
JOIN pg_class p ON p.oid = i.inhparent
JOIN pg_namespace n ON n.oid = p.relnamespace AND n.nspname = current_schema()
WHERE p.relname = ?`, table).Scan(&names).Error; err != nil {
		return nil, fmt.Errorf("list partitions of %s: %w", table, err)
	}
	return names, nil
}

// DropExpiredPartitions 直接删除完全过期的 PG 整月分区（避免 retention 清理逐行 DELETE）：
// 候选 = 月份早于 cutoff 月（按 cutoff 的 UTC 时刻取月，避免本地时区偏移超前误删）的分区，
// 且删除前校验分区内不存在保留数据；迁移冻结期间（ensureWritable）
// 直接返回 ErrMigrating，避免对冻结源库整月 DROP 丢数据；CH/SQLite 为 no-op。
func (s *gormLogStore) DropExpiredPartitions(ctx context.Context, cutoff time.Time) error {
	if !isPostgresDialect(s.db) {
		return nil
	}
	if err := s.ensureWritable(ctx); err != nil {
		return err
	}
	cu := cutoff.UTC()
	cutoffMonth := time.Date(cu.Year(), cu.Month(), 1, 0, 0, 0, 0, time.UTC)
	for _, table := range accessLogPartitionTables {
		column := "logged_at"
		if table == "w_user_access_logs" {
			column = "created_at"
		}
		names, err := listPartitionNames(ctx, s.db, table)
		if err != nil {
			return err
		}
		for _, name := range names {
			month, ok := partitionNameMonth(table, name)
			if !ok || !month.Before(cutoffMonth) {
				continue
			}
			var hasRetained int
			if err := s.db.WithContext(ctx).Raw("SELECT 1 FROM "+name+" WHERE "+column+" >= ? LIMIT 1", cu).Scan(&hasRetained).Error; err != nil {
				return fmt.Errorf("check partition %s retained rows: %w", name, err)
			}
			if hasRetained == 1 {
				continue
			}
			if err := s.db.WithContext(ctx).Exec("DROP TABLE IF EXISTS "+name).Error; err != nil {
				return fmt.Errorf("drop expired partition %s: %w", name, err)
			}
		}
	}
	return nil
}
