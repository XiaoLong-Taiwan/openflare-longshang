// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"time"

	"Wavelet/openflare/plugins/server/kernel/repository/logstore"
	"Wavelet/pkg/logger"
)

// StorageOperationSource reports the state and retained-data range of a log source.
type StorageOperationSource struct {
	Status string     `json:"status" enums:"available,empty,unavailable,error"`
	Oldest *time.Time `json:"oldest,omitempty"`
	Newest *time.Time `json:"newest,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// StorageOperationCapacity reports the scope and result of log-store capacity measurement.
type StorageOperationCapacity struct {
	Status      string `json:"status" enums:"available,shared,error"`
	Scope       string `json:"scope,omitempty" enums:"clickhouse_active_parts,primary_database"`
	BytesOnDisk *int64 `json:"bytes_on_disk,omitempty"`
	ActiveParts *int64 `json:"active_parts,omitempty"`
	TotalRows   *int64 `json:"total_rows,omitempty"`
}

// StorageOperations summarizes read-only database and log-store operational state.
type StorageOperations struct {
	Backend       string                           `json:"backend,omitempty"`
	BackendStatus string                           `json:"backend_status" enums:"available,error"`
	BackendError  string                           `json:"backend_error,omitempty"`
	Capacity      StorageOperationCapacity         `json:"capacity"`
	NodeLogs      StorageOperationSource           `json:"node_logs"`
	UserLogs      StorageOperationSource           `json:"user_logs"`
	Outbox        *logstore.NodeAccessOutboxHealth `json:"outbox,omitempty"`
	OutboxStatus  string                           `json:"outbox_status" enums:"available,error,unsupported"`
	OutboxError   string                           `json:"outbox_error,omitempty"`
}

// GetStorageOperations returns read-only status for the active log store and its Outbox.
func GetStorageOperations(ctx context.Context) StorageOperations {
	result := StorageOperations{
		BackendStatus: "error",
		Capacity:      StorageOperationCapacity{Status: "error"},
		NodeLogs:      StorageOperationSource{Status: "unavailable"},
		UserLogs:      StorageOperationSource{Status: "unavailable"},
		OutboxStatus:  "error",
	}

	backend, backendErr := logstore.ActiveDatabase(ctx)
	if backendErr != nil {
		logger.ErrorF(ctx, "resolve storage operations log backend: %v", backendErr)
		result.BackendError = "Could not resolve the log backend"
		result.OutboxError = result.BackendError
	} else {
		result.Backend = backend
		result.BackendStatus = "available"
		switch backend {
		case "clickhouse":
			result.Capacity.Scope = "clickhouse_active_parts"
			capacity, err := logstore.GetClickHouseCapacity(ctx)
			if err != nil {
				logger.ErrorF(ctx, "query storage operations ClickHouse capacity: %v", err)
			} else {
				result.Capacity.Status = "available"
				result.Capacity.BytesOnDisk = &capacity.BytesOnDisk
				result.Capacity.ActiveParts = &capacity.ActiveParts
				result.Capacity.TotalRows = &capacity.TotalRows
			}

			health, err := logstore.GetNodeAccessOutboxHealth(ctx)
			if err != nil {
				logger.ErrorF(ctx, "query storage operations Outbox health: %v", err)
				result.OutboxError = "Could not query Outbox health"
			} else {
				if health.LastError != "" {
					health.LastError = "Outbox processing encountered an error"
				}
				result.Outbox = &health
				result.OutboxStatus = "available"
			}
		case "postgres", "sqlite":
			result.Capacity = StorageOperationCapacity{Status: "shared", Scope: "primary_database"}
			result.OutboxStatus = "unsupported"
		default:
			result.OutboxStatus = "unsupported"
		}
	}

	store, storeErr := logstore.Active(ctx)
	if storeErr != nil {
		logger.ErrorF(ctx, "open storage operations log store: %v", storeErr)
		result.NodeLogs = StorageOperationSource{Status: "error", Error: "Could not open the log store"}
		result.UserLogs = result.NodeLogs
	} else {
		result.NodeLogs = migrationRangeStatus(ctx, "node", store.AccessLogs.MigrationRange)
		result.UserLogs = migrationRangeStatus(ctx, "user", store.UserAccessLogs.MigrationRange)
	}
	return result
}

func migrationRangeStatus(ctx context.Context, source string, query func(context.Context) (time.Time, time.Time, error)) StorageOperationSource {
	from, to, err := query(ctx)
	if err != nil {
		logger.ErrorF(ctx, "query storage operations %s log retention range: %v", source, err)
		return StorageOperationSource{Status: "error", Error: "Could not query the retained log range"}
	}
	if from.IsZero() || to.IsZero() {
		return StorageOperationSource{Status: "empty"}
	}
	return StorageOperationSource{Status: "available", Oldest: &from, Newest: &to}
}
