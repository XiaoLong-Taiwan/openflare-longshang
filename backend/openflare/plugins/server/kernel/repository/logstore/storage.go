// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package logstore

import (
	"Wavelet/core/contracts"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type logStorageService struct{}

// NewLogStorageService returns the log storage service implementation.
func NewLogStorageService() contracts.LogStorageService {
	return logStorageService{}
}

var logStorageCategories = []struct {
	key         string
	name        string
	table       string
	description string
	clearable   bool
}{
	{key: "node_access", name: "node_access", table: analyticsmodel.NodeAccessLog{}.TableName(), description: "Node access and request records", clearable: true},
	{key: "user_access", name: "user_access", table: analyticsmodel.UserAccessLog{}.TableName(), description: "User access and API request records", clearable: true},
	{key: "metric", name: "metric", table: analyticsmodel.NodeMetricSnapshot{}.TableName(), description: "Node CPU, memory, disk and network metrics", clearable: true},
	{key: "edge_health", name: "edge_health", table: analyticsmodel.NodeEdgeHealth{}.TableName(), description: "Node edge health records", clearable: true},
	{key: "frps", name: "frps", table: analyticsmodel.NodeObsFrps{}.TableName(), description: "FRPS observability records", clearable: true},
	{key: "frpc", name: "frpc", table: analyticsmodel.NodeObsFrpc{}.TableName(), description: "FRPC observability records", clearable: true},
	{key: "system", name: "system", table: "", description: "Process ring-buffer logs are memory-only", clearable: false},
}

func (logStorageService) Overview(ctx context.Context) (contracts.LogStorageOverview, error) {
	database, err := ActiveDatabase(ctx)
	if err != nil {
		return contracts.LogStorageOverview{}, err
	}
	result := contracts.LogStorageOverview{Database: database, Categories: make([]contracts.LogStorageCategory, 0, len(logStorageCategories))}
	for _, category := range logStorageCategories {
		item := contracts.LogStorageCategory{Key: category.key, Name: category.name, Table: category.table, Description: category.description, Clearable: category.clearable}
		if category.table != "" {
			item.Rows, item.Bytes, err = logStorageTableStats(ctx, database, category.table)
			if err != nil {
				return contracts.LogStorageOverview{}, err
			}
			result.TotalRows += item.Rows
			result.TotalBytes += item.Bytes
		}
		result.Categories = append(result.Categories, item)
	}
	return result, nil
}

func (logStorageService) Cleanup(ctx context.Context, category string, before *time.Time) ([]contracts.LogStorageCleanupResult, error) {
	store, err := Active(ctx)
	if err != nil {
		return nil, err
	}
	if category == "" {
		category = "all"
	}
	keys := make([]string, 0, len(logStorageCategories))
	if category == "all" {
		for _, item := range logStorageCategories {
			if item.clearable {
				keys = append(keys, item.key)
			}
		}
	} else {
		keys = append(keys, category)
	}
	results := make([]contracts.LogStorageCleanupResult, 0, len(keys))
	for _, key := range keys {
		var deleted int64
		switch key {
		case "node_access":
			if before == nil {
				deleted, err = store.AccessLogs.DeleteAll(ctx)
			} else {
				deleted, err = store.AccessLogs.DeleteBefore(ctx, *before)
			}
		case "user_access":
			if before == nil {
				deleted, err = store.UserAccessLogs.DeleteAll(ctx)
			} else {
				deleted, err = store.UserAccessLogs.DeleteBefore(ctx, *before)
			}
		case "metric":
			if before == nil {
				deleted, err = store.Observability.DeleteAllMetricSnapshots(ctx)
			} else {
				deleted, err = store.Observability.DeleteMetricSnapshotsBefore(ctx, *before)
			}
		case "edge_health":
			if before == nil {
				deleted, err = store.Observability.DeleteAllEdgeHealth(ctx)
			} else {
				deleted, err = store.Observability.DeleteEdgeHealthBefore(ctx, *before)
			}
		case "frps":
			if before == nil {
				deleted, err = store.Observability.DeleteAllNodeObservationFrps(ctx)
			} else {
				deleted, err = store.Observability.DeleteNodeObservationFrpsBefore(ctx, *before)
			}
		case "frpc":
			if before == nil {
				deleted, err = store.Observability.DeleteAllNodeObservationFrpc(ctx)
			} else {
				deleted, err = store.Observability.DeleteNodeObservationFrpcBefore(ctx, *before)
			}
		default:
			return nil, fmt.Errorf("unsupported log storage category: %s", key)
		}
		if err != nil {
			return nil, fmt.Errorf("cleanup %s: %w", key, err)
		}
		results = append(results, contracts.LogStorageCleanupResult{Category: key, Deleted: deleted})
	}
	return results, nil
}

func logStorageTableStats(ctx context.Context, database, table string) (int64, int64, error) {
	if database == dbNameClickHouse {
		conn, err := chConn(ctx)
		if err != nil {
			return 0, 0, err
		}
		var rows, bytes uint64
		err = conn.QueryRow(ctx, "SELECT ifNull(sum(rows), 0), ifNull(sum(bytes_on_disk), 0) FROM system.parts WHERE active AND table = ?", table).Scan(&rows, &bytes)
		return int64(rows), int64(bytes), err
	}
	db := getGormDB(ctx)
	if db == nil {
		return 0, 0, fmt.Errorf("database is not initialized")
	}
	var rows int64
	if err := db.Table(table).Count(&rows).Error; err != nil {
		return 0, 0, err
	}
	if database == dbNamePostgres {
		var bytes int64
		if err := db.Raw("SELECT pg_total_relation_size(?)", table).Scan(&bytes).Error; err != nil {
			return 0, 0, err
		}
		return rows, bytes, nil
	}
	bytes, err := sqliteTableBytes(ctx, db, table)
	if err != nil {
		return rows, 0, nil
	}
	return rows, bytes, nil
}

func sqliteTableBytes(_ context.Context, db *gorm.DB, table string) (int64, error) {
	var bytes int64
	if err := db.Raw("SELECT COALESCE(sum(pgsize), 0) FROM dbstat WHERE name = ?", table).Scan(&bytes).Error; err != nil {
		return 0, fmt.Errorf("query sqlite table size: %w", err)
	}
	return bytes, nil
}
