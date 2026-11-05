package logstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"gorm.io/gorm"
)

var nodeOutboxGate = make(chan struct{}, 1)

func PrepareNodeAccessSwitch(ctx context.Context) error {
	store, err := Active(ctx)
	if err != nil {
		return err
	}
	switch access := store.AccessLogs.(type) {
	case *clickhouseLogStore:
		return access.archiveDaily(ctx)
	case *gormLogStore:
		if err := access.archiveDaily(ctx); err != nil {
			return err
		}
		return backfillSampleKeys(ctx, access.db)
	}
	return nil
}

func deliverNodeChanges(ctx context.Context, delivery nodeDelivery) error {
	ids := append([]uint64(nil), delivery.DeleteIDs...)
	for _, row := range delivery.Rows {
		ids = append(ids, row.ID)
	}
	if len(ids) > 0 {
		if err := analyticsrepo.DeleteNodeAccessIDs(ctx, ids); err != nil {
			return err
		}
	}
	if len(delivery.Rows) > 0 {
		if err := analyticsrepo.BatchInsertNodeAccessLogs(ctx, delivery.Rows); err != nil {
			return err
		}
	}
	if len(delivery.MarkIDs) > 0 {
		return analyticsrepo.MarkNodeAccessIDs(ctx, delivery.MarkIDs)
	}
	return nil
}

func flushNodeOutbox(ctx context.Context, db *gorm.DB, deliver func(context.Context, nodeDelivery) error) (result error) {
	select {
	case nodeOutboxGate <- struct{}{}:
		defer func() { <-nodeOutboxGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if db == nil {
		return fmt.Errorf("node outbox database is not initialized")
	}
	if isPostgresDialect(db) {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			return err
		}
		locked := false
		defer func() {
			if locked {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				var unlocked bool
				err := conn.QueryRowContext(cleanupCtx, "SELECT pg_advisory_unlock(7130261104)").Scan(&unlocked)
				if err == nil && !unlocked {
					err = fmt.Errorf("node outbox session lock was not held")
				}
				if err != nil {
					result = errors.Join(result, fmt.Errorf("unlock node outbox: %w", err))
					locked = false
				}
			}
			if !locked {
				if err := conn.Raw(func(any) error { return driver.ErrBadConn }); err != nil && !errors.Is(err, driver.ErrBadConn) && !errors.Is(err, sql.ErrConnDone) {
					result = errors.Join(result, err)
				}
			}
			if err := conn.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
				result = errors.Join(result, err)
			}
		}()
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(7130261104)"); err != nil {
			return err
		}
		locked = true
		db = db.Session(&gorm.Session{NewDB: true, SkipDefaultTransaction: true})
		db.Statement.ConnPool = conn
	}
	for batch := 0; batch < cleanupBatchesPerRun; batch++ {
		var page []analyticsmodel.NodeAccessOutbox
		if err := db.WithContext(ctx).Order("id").Limit(1).Find(&page).Error; err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		var delivery nodeDelivery
		if err := json.Unmarshal([]byte(page[0].Payload), &delivery); err != nil {
			return err
		}
		if err := deliver(ctx, delivery); err != nil {
			return err
		}
		if err := db.WithContext(ctx).Where("id = ?", page[0].ID).Delete(&analyticsmodel.NodeAccessOutbox{}).Error; err != nil {
			return err
		}
	}
	return fmt.Errorf("node outbox drain exceeded per-run page budget at %s", time.Now().UTC().Format(time.RFC3339))
}
