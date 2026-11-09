// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"

	"gorm.io/gorm"
)

// ListOpenFlareNodePage returns a bounded page and the count for the same filters.
func ListOpenFlareNodePage(ctx context.Context, query model.OpenFlareNodeListQuery) ([]model.OpenFlareNode, int64, error) {
	conn := DB(ctx)
	if conn == nil {
		return nil, 0, errors.New(errDatabaseNotInitialized)
	}
	filtered := filterOpenFlareNodes(conn.Model(&model.OpenFlareNode{}), query)
	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	nodes := make([]model.OpenFlareNode, 0)
	err := filtered.Order("id DESC").Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).Find(&nodes).Error
	return nodes, total, err
}

func filterOpenFlareNodes(query *gorm.DB, filters model.OpenFlareNodeListQuery) *gorm.DB {
	if filters.Search != "" {
		escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(filters.Search)
		pattern := "%" + escaped + "%"
		query = query.Where("(LOWER(name) LIKE LOWER(?) ESCAPE '!' OR LOWER(node_id) LIKE LOWER(?) ESCAPE '!' OR LOWER(ip) LIKE LOWER(?) ESCAPE '!')", pattern, pattern, pattern)
	}
	if filters.NodeType == "edge_node" {
		query = query.Where("(node_type = ? OR node_type = '')", filters.NodeType)
	} else if filters.NodeType != "" {
		query = query.Where("node_type = ?", filters.NodeType)
	}
	cutoff := filters.Now.Add(-time.Minute)
	connected := filters.ConnectedNodeIDs
	switch filters.Status {
	case "pending":
		query = query.Where("(last_seen_at IS NULL OR last_seen_at = ?)", time.Time{})
		if len(connected) > 0 {
			query = query.Where("node_id NOT IN ?", connected)
		}
	case "online":
		if len(connected) > 0 {
			query = query.Where("((last_seen_at >= ? AND last_seen_at <> ?) OR node_id IN ?)", cutoff, time.Time{}, connected)
		} else {
			query = query.Where("last_seen_at >= ? AND last_seen_at <> ?", cutoff, time.Time{})
		}
	case "offline":
		query = query.Where("last_seen_at < ? AND last_seen_at <> ?", cutoff, time.Time{})
		if len(connected) > 0 {
			query = query.Where("node_id NOT IN ?", connected)
		}
	}
	return query
}
