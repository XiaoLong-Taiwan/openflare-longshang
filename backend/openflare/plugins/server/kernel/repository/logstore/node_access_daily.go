package logstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"Wavelet/openflare/plugins/server/kernel/model"
	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"Wavelet/pkg/idgen"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const nodeDailyQuotaBytes uint64 = 1_000_000_000

func detailBytes(row analyticsmodel.NodeAccessLog) uint64 {
	return 64 + uint64(len(row.NodeID)+len(row.RemoteAddr)+len(row.Region)+len(row.Host)+len(row.Path)+len(row.UserAgent)+len(row.CacheStatus))
}

func accountingLock(tx *gorm.DB) error {
	if isPostgresDialect(tx) {
		return tx.Exec("SELECT pg_advisory_xact_lock(7130261103)").Error
	}
	return nil
}

func sampleHash(row analyticsmodel.NodeAccessLog) string {
	row.ID = 0
	row.CreatedAt = time.Time{}
	row.SummaryTracked = false
	row.SampleKey = ""
	row.DetailSize = 0
	row.LoggedAt = row.LoggedAt.UTC()
	data, _ := json.Marshal(row)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func samplePrefix(rows []analyticsmodel.NodeAccessLog, budget uint64) ([]analyticsmodel.NodeAccessLog, uint64) {
	selected, used, _ := sampleWithBoundary(rows, budget, "")
	return selected, used
}

func sampleWithBoundary(rows []analyticsmodel.NodeAccessLog, budget uint64, boundary string) ([]analyticsmodel.NodeAccessLog, uint64, string) {
	type ranked struct {
		row  analyticsmodel.NodeAccessLog
		hash string
	}
	ranks := make([]ranked, len(rows))
	for i, row := range rows {
		ranks[i] = ranked{row, sampleHash(row)}
	}
	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].hash == ranks[j].hash {
			return ranks[i].row.ID < ranks[j].row.ID
		}
		return ranks[i].hash < ranks[j].hash
	})
	var used uint64
	selected := make([]analyticsmodel.NodeAccessLog, 0, len(rows))
	for _, item := range ranks {
		size := detailBytes(item.row)
		key := fmt.Sprintf("%s%020d", item.hash, item.row.ID)
		if boundary != "" && key >= boundary {
			break
		}
		if size > budget-used {
			boundary = key
			break
		}
		used += size
		selected = append(selected, item.row)
	}
	return selected, used, boundary
}

func dailyGroups(rows []analyticsmodel.NodeAccessLog, source string) []analyticsmodel.NodeAccessDaily {
	type key struct {
		day, node string
		status    int32
	}
	groups := map[key]analyticsmodel.NodeAccessDaily{}
	for _, row := range rows {
		k := key{row.LoggedAt.UTC().Format(time.DateOnly), row.NodeID, row.StatusCode}
		v := groups[k]
		v.Day, v.NodeID, v.StatusCode, v.Source = k.day, k.node, k.status, source
		v.RequestCount++
		v.BytesSent += row.BytesSent
		v.RequestLength += row.RequestLength
		groups[k] = v
	}
	result := make([]analyticsmodel.NodeAccessDaily, 0, len(groups))
	for _, row := range groups {
		result = append(result, row)
	}
	return result
}

func addDaily(tx *gorm.DB, rows []analyticsmodel.NodeAccessDaily) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "day"}, {Name: "node_id"}, {Name: "status_code"}, {Name: "source"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":  gorm.Expr("of_node_access_daily.request_count + excluded.request_count"),
			"bytes_sent":     gorm.Expr("of_node_access_daily.bytes_sent + excluded.bytes_sent"),
			"request_length": gorm.Expr("of_node_access_daily.request_length + excluded.request_length"),
		}),
	}).CreateInBatches(rows, insertBatchSize).Error
}

func persistNodeBatch(ctx context.Context, db *gorm.DB, rows []analyticsmodel.NodeAccessLog, write func(*gorm.DB, []analyticsmodel.NodeAccessLog) error) error {
	if len(rows) == 0 {
		return nil
	}
	if db == nil {
		return errors.New("node access accounting database is not initialized")
	}
	for i := range rows {
		if rows[i].ID == 0 {
			rows[i].ID = idgen.NextUint64ID()
		}
		prepareSample(&rows[i])
	}
	ids := make([]uint64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	payload, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payload)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := accountingLock(tx); err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&analyticsmodel.NodeAccessBatch{BatchKey: hex.EncodeToString(hash[:]), CreatedAt: time.Now().UTC()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := addDaily(tx, dailyGroups(rows, "ingest")); err != nil {
			return err
		}
		type key struct{ day, node string }
		groups := map[key][]analyticsmodel.NodeAccessLog{}
		for _, row := range rows {
			k := key{row.LoggedAt.UTC().Format(time.DateOnly), row.NodeID}
			groups[k] = append(groups[k], row)
		}
		var candidates []analyticsmodel.NodeAccessLog
		for _, row := range rows {
			if !row.LoggedAt.Before(time.Now().UTC().AddDate(0, 0, -7)) {
				row.SummaryTracked = true
				candidates = append(candidates, row)
			}
		}
		if len(candidates) > 0 {
			if err := tx.CreateInBatches(candidates, insertBatchSize).Error; err != nil {
				return err
			}
		}
		quotas := make([]analyticsmodel.NodeAccessQuota, 0, len(groups))
		var selected []analyticsmodel.NodeAccessLog
		for k, incoming := range groups {
			var quota analyticsmodel.NodeAccessQuota
			if err := tx.Where("day = ? AND node_id = ?", k.day, k.node).Limit(1).Find(&quota).Error; err != nil {
				return err
			}
			used, boundary, err := sqlSampleBoundary(tx, k.node, k.day, nodeDailyQuotaBytes, quota.SampleBoundary)
			if err != nil {
				return err
			}
			var seen uint64
			for _, row := range incoming {
				seen += detailBytes(row)
				if !row.LoggedAt.Before(time.Now().UTC().AddDate(0, 0, -7)) && (boundary == "" || row.SampleKey < boundary) {
					row.SummaryTracked = true
					selected = append(selected, row)
				}
			}
			quotas = append(quotas, analyticsmodel.NodeAccessQuota{Day: k.day, NodeID: k.node, UsedBytes: used, SeenBytes: seen, SampleBoundary: boundary})
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "day"}, {Name: "node_id"}}, DoUpdates: clause.Assignments(map[string]any{"used_bytes": gorm.Expr("excluded.used_bytes"), "seen_bytes": gorm.Expr("of_node_access_quota.seen_bytes + excluded.seen_bytes"), "sample_boundary": gorm.Expr("excluded.sample_boundary")})}).CreateInBatches(quotas, insertBatchSize).Error; err != nil {
			return err
		}
		if len(selected) == 0 {
			return nil
		}
		return write(tx, selected)
	})
}

func saveLegacyDaily(ctx context.Context, db *gorm.DB, rows []analyticsmodel.NodeAccessDaily) error {
	if db == nil {
		return errors.New("node access accounting database is not initialized")
	}
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		rows[i].Source = "legacy"
	}
	updates := map[string]any{}
	for _, col := range []string{"request_count", "bytes_sent", "request_length"} {
		updates[col] = gorm.Expr("CASE WHEN of_node_access_daily." + col + " < excluded." + col + " THEN excluded." + col + " ELSE of_node_access_daily." + col + " END")
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "day"}, {Name: "node_id"}, {Name: "status_code"}, {Name: "source"}}, DoUpdates: clause.Assignments(updates)}).CreateInBatches(rows, insertBatchSize).Error
}

func (s *gormLogStore) maintainNodeAccounting(ctx context.Context) error {
	return s.maintainNodeAccountingWithBudget(ctx, nodeDailyQuotaBytes)
}

func (s *gormLogStore) maintainNodeAccountingWithBudget(ctx context.Context, budget uint64) error {
	if err := s.archiveDaily(ctx); err != nil {
		return err
	}
	if err := backfillSampleKeys(ctx, s.db); err != nil {
		return err
	}
	afterDay, afterNode := "", ""
	for {
		var groups []analyticsmodel.NodeAccessDaily
		day := "strftime('%Y-%m-%d', logged_at)"
		if isPostgresDialect(s.db) {
			day = "to_char(logged_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')"
		}
		q := s.db.WithContext(ctx).Model(&analyticsmodel.NodeAccessLog{}).Select(day+" AS day,node_id").Where("("+day+" > ?) OR ("+day+" = ? AND node_id > ?)", afterDay, afterDay, afterNode).Group(day + ",node_id").Order("day,node_id").Limit(100)
		if err := q.Scan(&groups).Error; err != nil {
			return err
		}
		if len(groups) == 0 {
			break
		}
		for _, group := range groups {
			boundary := ""
			err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := accountingLock(tx); err != nil {
					return err
				}
				var quota analyticsmodel.NodeAccessQuota
				if err := tx.Where("day = ? AND node_id = ?", group.Day, group.NodeID).Limit(1).Find(&quota).Error; err != nil {
					return err
				}
				used, next, err := sqlSampleBoundary(tx, group.NodeID, group.Day, budget, quota.SampleBoundary)
				if err != nil {
					return err
				}
				boundary = next
				return updateSampleQuota(tx, analyticsmodel.NodeAccessQuota{Day: group.Day, NodeID: group.NodeID, UsedBytes: used, SampleBoundary: next})
			})
			if err != nil {
				return err
			}
			ch, err := resolveDatabase(ctx)
			if err != nil {
				return err
			}
			if err := trimSample(ctx, s.db, group.NodeID, group.Day, boundary, ch == "clickhouse"); err != nil {
				return err
			}
			afterDay, afterNode = group.Day, group.NodeID
		}
	}
	return pruneNodeLedger(ctx, s.db)
}

func (s *gormLogStore) archiveDaily(ctx context.Context) error {
	return s.archiveScope(ctx, "1 = 1")
}

func (s *gormLogStore) archiveScope(ctx context.Context, condition string, args ...any) error {
	if err := prepareLegacyBaseline(ctx, s.db); err != nil {
		return err
	}
	for {
		processed := 0
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := accountingLock(tx); err != nil {
				return err
			}
			type pendingRow struct {
				analyticsmodel.NodeAccessLog `gorm:"embedded"`
				ArchiveKey                   string
			}
			var pending []pendingRow
			physical := "CAST(rowid AS TEXT)"
			if isPostgresDialect(tx) {
				physical = "tableoid::text || ':' || ctid::text"
			}
			q := tx.Model(&analyticsmodel.NodeAccessLog{}).Select("*, "+physical+" AS archive_key").Where("summary_tracked = ?", false).Where(condition, args...).Order("logged_at,id").Limit(cleanupBatchSize)
			if err := q.Find(&pending).Error; err != nil {
				return err
			}
			processed = len(pending)
			if processed == 0 {
				return nil
			}
			rows := make([]analyticsmodel.NodeAccessLog, len(pending))
			keys := make([]string, len(pending))
			for i, row := range pending {
				rows[i] = row.NodeAccessLog
				keys[i] = row.ArchiveKey
			}
			var delta []analyticsmodel.NodeAccessLog
			names := make([]string, 0, len(rows))
			for _, row := range rows {
				names = append(names, "baseline:"+row.LoggedAt.UTC().Format(time.DateOnly)+":"+row.NodeID+":"+fmt.Sprint(row.StatusCode))
			}
			var checkpoints []analyticsmodel.NodeAccessCheckpoint
			if err := tx.Where("name IN ?", names).Find(&checkpoints).Error; err != nil {
				return err
			}
			through := map[string]uint64{}
			for _, checkpoint := range checkpoints {
				through[checkpoint.Name] = checkpoint.ThroughID
			}
			for _, row := range rows {
				name := "baseline:" + row.LoggedAt.UTC().Format(time.DateOnly) + ":" + row.NodeID + ":" + fmt.Sprint(row.StatusCode)
				limit, ok := through[name]
				if !ok || row.ID > limit {
					delta = append(delta, row)
				}
			}
			if err := addDaily(tx, dailyGroups(delta, "legacy")); err != nil {
				return err
			}
			return tx.Model(&analyticsmodel.NodeAccessLog{}).Where("summary_tracked = ? AND "+physical+" IN ?", false, keys).Where(condition, args...).Update("summary_tracked", true).Error
		})
		if err != nil {
			return err
		}
		if processed < cleanupBatchSize {
			return nil
		}
	}
}

func (s *clickhouseLogStore) archiveDaily(ctx context.Context) error {
	db := getGormDB(ctx)
	if db == nil {
		return errors.New("node access accounting database is not initialized")
	}
	if err := flushNodeOutbox(ctx, db, deliverNodeChanges); err != nil {
		return err
	}
	for {
		rows, err := analyticsrepo.LegacyNodeAccessPage(ctx, insertBatchSize)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return importClickHouseIndex(ctx, db)
		}
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := accountingLock(tx); err != nil {
				return err
			}
			ids := make([]uint64, len(rows))
			for i := range rows {
				ids[i] = rows[i].ID
				prepareSample(&rows[i])
			}
			data, err := json.Marshal(rows)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			batch := analyticsmodel.NodeAccessBatch{BatchKey: "legacy-ch-" + hex.EncodeToString(sum[:]), CreatedAt: time.Now().UTC()}
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				if err := addDaily(tx, dailyGroups(rows, "legacy-ch")); err != nil {
					return err
				}
				for i := range rows {
					rows[i].SummaryTracked = true
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, insertBatchSize).Error; err != nil {
					return err
				}
			}
			return enqueueNodeDelivery(tx, nodeDelivery{MarkIDs: ids})
		})
		if err != nil {
			return err
		}
		if err := flushNodeOutbox(ctx, db, deliverNodeChanges); err != nil {
			return err
		}
	}
}

func ListNodeAccessDaily(ctx context.Context, nodeID string, since, until time.Time) ([]analyticsmodel.NodeAccessDaily, error) {
	return listNodeAccessDaily(ctx, nodeID, since, until, 0, true)
}

func SummarizeNodeAccessDaily(ctx context.Context, query model.OpenFlareAccessLogQuery) ([]analyticsmodel.NodeAccessDaily, error) {
	return listNodeAccessDaily(ctx, query.NodeID, query.Since, query.Until, query.StatusCode, false)
}

func listNodeAccessDaily(ctx context.Context, nodeID string, since, until time.Time, statusCode int, byDay bool) ([]analyticsmodel.NodeAccessDaily, error) {
	store, err := Active(ctx)
	if err != nil {
		return nil, err
	}
	switch access := store.AccessLogs.(type) {
	case *gormLogStore:
		if err := access.archiveDaily(ctx); err != nil {
			return nil, err
		}
	case *clickhouseLogStore:
		if err := access.archiveDaily(ctx); err != nil {
			return nil, err
		}
	}
	db := getGormDB(ctx)
	if db == nil {
		return nil, errors.New("node access accounting database is not initialized")
	}
	q := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessDaily{})
	if nodeID != "" {
		q = q.Where("node_id = ?", nodeID)
	}
	if !since.IsZero() {
		q = q.Where("day >= ?", since.UTC().Format(time.DateOnly))
	}
	if !until.IsZero() {
		q = q.Where("day <= ?", until.UTC().Format(time.DateOnly))
	}
	if statusCode != 0 {
		q = q.Where("status_code = ?", statusCode)
	}
	group := "node_id, status_code"
	if byDay {
		group = "day, " + group
	}
	var rows []analyticsmodel.NodeAccessDaily
	err = q.Select(group + ", SUM(request_count) AS request_count, SUM(bytes_sent) AS bytes_sent, SUM(request_length) AS request_length").Group(group).Order(group).Scan(&rows).Error
	return rows, err
}
