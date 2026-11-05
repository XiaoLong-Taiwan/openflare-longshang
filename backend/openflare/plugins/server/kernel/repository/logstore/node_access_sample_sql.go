package logstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	"Wavelet/pkg/idgen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type nodeDelivery struct {
	Rows      []analyticsmodel.NodeAccessLog
	DeleteIDs []uint64
	MarkIDs   []uint64
}

func enqueueNodeDelivery(tx *gorm.DB, delivery nodeDelivery) error {
	if len(delivery.Rows)+len(delivery.DeleteIDs)+len(delivery.MarkIDs) == 0 {
		return nil
	}
	payload, err := json.Marshal(delivery)
	if err != nil {
		return err
	}
	return tx.Create(&analyticsmodel.NodeAccessOutbox{ID: idgen.NextUint64ID(), Payload: string(payload), CreatedAt: time.Now().UTC()}).Error
}

func prepareSample(row *analyticsmodel.NodeAccessLog) {
	row.SampleKey = fmt.Sprintf("%s%020d", sampleHash(*row), row.ID)
	row.DetailSize = detailBytes(*row)
}

func sqlSampleBoundary(tx *gorm.DB, node, day string, budget uint64, previous string) (uint64, string, error) {
	start, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return 0, "", err
	}
	var result struct {
		Used     uint64
		Boundary string
	}
	query := `WITH ranked AS (SELECT sample_key, detail_size, SUM(detail_size) OVER (ORDER BY sample_key ROWS UNBOUNDED PRECEDING) AS running FROM of_node_access_logs WHERE node_id = ? AND logged_at >= ? AND logged_at < ? AND (? = '' OR sample_key < ?)) SELECT COALESCE(MAX(CASE WHEN running <= ? THEN running ELSE 0 END),0) AS used, COALESCE(MIN(CASE WHEN running > ? THEN sample_key ELSE NULL END),'') AS boundary FROM ranked`
	if err := tx.Raw(query, node, start, start.AddDate(0, 0, 1), previous, previous, budget, budget).Scan(&result).Error; err != nil {
		return 0, "", err
	}
	if previous != "" && (result.Boundary == "" || previous < result.Boundary) {
		result.Boundary = previous
	}
	return result.Used, result.Boundary, nil
}

func trimSamplePage(ctx context.Context, db *gorm.DB, node, day, boundary string, ch bool) (bool, error) {
	if boundary == "" {
		return false, nil
	}
	start, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return false, err
	}
	count := 0
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := accountingLock(tx); err != nil {
			return err
		}
		var rows []analyticsmodel.NodeAccessLog
		if err := tx.Select("id,logged_at").Where("node_id = ? AND logged_at >= ? AND logged_at < ? AND sample_key >= ? AND summary_tracked = true", node, start, start.AddDate(0, 0, 1), boundary).Order("sample_key").Limit(insertBatchSize).Find(&rows).Error; err != nil {
			return err
		}
		count = len(rows)
		if count == 0 {
			return nil
		}
		ids := make([]uint64, count)
		for i, row := range rows {
			ids[i] = row.ID
		}
		if ch {
			if err := enqueueNodeDelivery(tx, nodeDelivery{DeleteIDs: ids}); err != nil {
				return err
			}
		}
		selection := tx.Model(&analyticsmodel.NodeAccessLog{}).Select("rowid").Where("node_id = ? AND logged_at >= ? AND logged_at < ? AND sample_key >= ? AND summary_tracked = true", node, start, start.AddDate(0, 0, 1), boundary).Order("sample_key").Limit(insertBatchSize)
		physical := "rowid"
		if isPostgresDialect(tx) {
			physical = "(tableoid,ctid)"
			selection = selection.Select("tableoid,ctid")
		}
		return tx.Where(physical+" IN (?)", selection).Delete(&analyticsmodel.NodeAccessLog{}).Error
	})
	return count == insertBatchSize, err
}

func trimSample(ctx context.Context, db *gorm.DB, node, day, boundary string, ch bool) error {
	for {
		more, err := trimSamplePage(ctx, db, node, day, boundary, ch)
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
	}
}

func updateSampleQuota(tx *gorm.DB, quota analyticsmodel.NodeAccessQuota) error {
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "day"}, {Name: "node_id"}}, DoUpdates: clause.AssignmentColumns([]string{"used_bytes", "sample_boundary"})}).Create(&quota).Error
}

func finishNodeSamples(ctx context.Context, db *gorm.DB, rows []analyticsmodel.NodeAccessLog, ch bool) error {
	type key struct{ day, node string }
	groups := map[key]bool{}
	for _, row := range rows {
		groups[key{row.LoggedAt.UTC().Format(time.DateOnly), row.NodeID}] = true
	}
	for k := range groups {
		var quota analyticsmodel.NodeAccessQuota
		if err := db.WithContext(ctx).Where("day = ? AND node_id = ?", k.day, k.node).First(&quota).Error; err != nil {
			return err
		}
		if err := trimSample(ctx, db, k.node, k.day, quota.SampleBoundary, ch); err != nil {
			return err
		}
	}
	return nil
}

func backfillSampleKeys(ctx context.Context, db *gorm.DB) error {
	for {
		var page []analyticsmodel.NodeAccessLog
		if err := db.WithContext(ctx).Where("sample_key = '' OR sample_key IS NULL").Order("logged_at,id").Limit(insertBatchSize).Find(&page).Error; err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := accountingLock(tx); err != nil {
				return err
			}
			for i := range page {
				prepareSample(&page[i])
				if err := tx.Model(&analyticsmodel.NodeAccessLog{}).Where("id = ? AND logged_at = ? AND (sample_key = '' OR sample_key IS NULL)", page[i].ID, page[i].LoggedAt).Updates(map[string]any{"sample_key": page[i].SampleKey, "detail_size": page[i].DetailSize}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
}

func pruneNodeLedger(ctx context.Context, db *gorm.DB) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -8)
	for {
		var rows []analyticsmodel.NodeAccessBatch
		if err := db.WithContext(ctx).Where("created_at IS NULL OR created_at < ?", cutoff).Limit(insertBatchSize).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		keys := make([]string, len(rows))
		for i, row := range rows {
			keys[i] = row.BatchKey
		}
		if err := db.WithContext(ctx).Where("batch_key IN ?", keys).Delete(&analyticsmodel.NodeAccessBatch{}).Error; err != nil {
			return err
		}
	}
	for {
		var rows []analyticsmodel.NodeAccessQuota
		if err := db.WithContext(ctx).Where("day < ?", cutoff.Format(time.DateOnly)).Limit(insertBatchSize).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			if err := db.WithContext(ctx).Where("day = ? AND node_id = ?", row.Day, row.NodeID).Delete(&analyticsmodel.NodeAccessQuota{}).Error; err != nil {
				return err
			}
		}
	}
}
