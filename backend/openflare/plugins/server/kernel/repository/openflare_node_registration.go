package repository

import (
	"context"
	"errors"

	"Wavelet/openflare/plugins/server/kernel/model"

	"gorm.io/gorm/clause"
)

// CreateOrGetOpenFlareDiscoveryNode inserts a node or returns its existing identity.
func CreateOrGetOpenFlareDiscoveryNode(ctx context.Context, node *model.OpenFlareNode) (*model.OpenFlareNode, error) {
	conn := DB(ctx)
	if conn == nil {
		return nil, errors.New(errDatabaseNotInitialized)
	}
	if err := conn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_id"}},
		DoNothing: true,
	}).Create(node).Error; err != nil {
		return nil, err
	}
	return GetOpenFlareNodeByNodeID(ctx, node.NodeID)
}
