package logstore

import (
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func importClickHouseIndex(ctx context.Context, db *gorm.DB) error {
	for {
		checkpoint := analyticsmodel.NodeAccessCheckpoint{Name: "ch-detail-index"}
		if err := db.WithContext(ctx).Where("name = ?", checkpoint.Name).Limit(1).Find(&checkpoint).Error; err != nil {
			return err
		}
		rows, err := analyticsrepo.NodeAccessIndexPage(ctx, checkpoint.ThroughID, insertBatchSize)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			prepareSample(&rows[i])
		}
		checkpoint.ThroughID = rows[len(rows)-1].ID
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := accountingLock(tx); err != nil {
				return err
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, insertBatchSize).Error; err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoUpdates: clause.AssignmentColumns([]string{"through_id"})}).Create(&checkpoint).Error
		}); err != nil {
			return err
		}
	}
}
