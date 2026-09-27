// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package contracts

import (
	"context"
	"time"
)

// LogStorageCategory describes one persisted or in-memory log category.
type LogStorageCategory struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Table       string `json:"table"`
	Rows        int64  `json:"rows"`
	Bytes       int64  `json:"bytes"`
	Clearable   bool   `json:"clearable"`
	Description string `json:"description"`
}

// LogStorageOverview contains the current log database and category sizes.
type LogStorageOverview struct {
	Database   string               `json:"database"`
	Categories []LogStorageCategory `json:"categories"`
	TotalRows  int64                `json:"total_rows"`
	TotalBytes int64                `json:"total_bytes"`
}

// LogStorageCleanupRequest describes a category cleanup request.
type LogStorageCleanupRequest struct {
	Category string `json:"category"`
	Days     *int   `json:"days"`
}

// LogStorageCleanupResult reports the number of rows removed from one category.
type LogStorageCleanupResult struct {
	Category string `json:"category"`
	Deleted  int64  `json:"deleted"`
}

// LogStorageService exposes log capacity and cleanup operations to admin tooling.
type LogStorageService interface {
	Overview(ctx context.Context) (LogStorageOverview, error)
	Cleanup(ctx context.Context, category string, before *time.Time) ([]LogStorageCleanupResult, error)
}
