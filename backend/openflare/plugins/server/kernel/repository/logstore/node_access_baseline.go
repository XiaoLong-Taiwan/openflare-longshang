package logstore

import (
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
)

func prepareLegacyBaseline(ctx context.Context, db *gorm.DB) error {
	var complete int64
	if err := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessCheckpoint{}).Where("name = ?", "legacy-v2").Count(&complete).Error; err != nil {
		return err
	}
	if complete > 0 {
		return nil
	}
	for {
		var groups []analyticsmodel.NodeAccessDaily
		day := "strftime('%Y-%m-%d', logged_at)"
		if isPostgresDialect(db) {
			day = "to_char(logged_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')"
		}
		if err := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessDaily{}).Where("source = ?", "legacy").Where("NOT EXISTS (SELECT 1 FROM of_node_access_checkpoints WHERE name = 'baseline:' || of_node_access_daily.day || ':' || of_node_access_daily.node_id || ':' || CAST(of_node_access_daily.status_code AS TEXT))").Order("day,node_id,status_code").Limit(1).Find(&groups).Error; err != nil {
			return err
		}
		if len(groups) == 0 {
			break
		}
		group := groups[0]
		name := "baseline:" + group.Day + ":" + group.NodeID + ":" + strconv.Itoa(int(group.StatusCode))
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := accountingLock(tx); err != nil {
				return err
			}
			var existing int64
			if err := tx.Model(&analyticsmodel.NodeAccessCheckpoint{}).Where("name IN ?", []string{name, "legacy-v2"}).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return nil
			}
			var snapshot struct {
				analyticsmodel.NodeAccessDaily `gorm:"embedded"`
				ThroughID                      uint64
			}
			if err := tx.Model(&analyticsmodel.NodeAccessLog{}).Select("COUNT(*) AS request_count,COALESCE(SUM(bytes_sent),0) AS bytes_sent,COALESCE(SUM(request_length),0) AS request_length,COALESCE(MAX(id),0) AS through_id").Where("summary_tracked = false AND "+day+" = ? AND node_id = ? AND status_code = ?", group.Day, group.NodeID, group.StatusCode).Scan(&snapshot).Error; err != nil {
				return err
			}
			aggregate := snapshot.NodeAccessDaily
			aggregate.Day, aggregate.NodeID, aggregate.StatusCode = group.Day, group.NodeID, group.StatusCode
			if err := saveLegacyDaily(ctx, tx, []analyticsmodel.NodeAccessDaily{aggregate}); err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&analyticsmodel.NodeAccessCheckpoint{Name: name, ThroughID: snapshot.ThroughID}).Error
		})
		if err != nil {
			return err
		}
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&analyticsmodel.NodeAccessCheckpoint{Name: "legacy-v2"}).Error
}
