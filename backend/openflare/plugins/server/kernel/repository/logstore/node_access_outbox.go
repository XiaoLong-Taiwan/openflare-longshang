package logstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	analyticsmodel "Wavelet/openflare/plugins/server/kernel/model/analytics"
	analyticsrepo "Wavelet/openflare/plugins/server/kernel/repository/analytics"
	"gorm.io/gorm"
)

var nodeOutboxGate = make(chan struct{}, 1)

// NodeAccessOutboxHealth summarizes pending ClickHouse delivery and recent worker status.
type NodeAccessOutboxHealth struct {
	Pending     int64     `json:"pending"`
	OldestAt    time.Time `json:"oldest_at,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
	LastAttempt time.Time `json:"last_attempt,omitempty"`
	Healthy     bool      `json:"healthy"`
}

type nodeOutboxDBHealth struct {
	Pending int64
	Oldest  *time.Time
	Error   string
}

var nodeOutboxHealth struct {
	sync.RWMutex
	lastAttempt time.Time
	lastError   string
}

// GetNodeAccessOutboxHealth returns durable queue depth and the latest retry failure.
func GetNodeAccessOutboxHealth(ctx context.Context) (NodeAccessOutboxHealth, error) {
	db := getGormDB(ctx)
	if db == nil {
		return NodeAccessOutboxHealth{}, fmt.Errorf("node outbox database is not initialized")
	}
	var health NodeAccessOutboxHealth
	var row nodeOutboxDBHealth
	var oldest sql.NullString
	var lastError sql.NullString
	if err := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessOutbox{}).
		Select("COUNT(*) AS pending, MIN(created_at) AS oldest, COALESCE(MAX(last_error), '') AS error").
		Row().Scan(&row.Pending, &oldest, &lastError); err != nil {
		return health, err
	}
	health.Pending = row.Pending
	if oldest.Valid {
		parsed, err := parseOutboxTime(oldest.String)
		if err != nil {
			return health, err
		}
		health.OldestAt = parsed
	}
	health.LastError = lastError.String
	nodeOutboxHealth.RLock()
	health.LastAttempt = nodeOutboxHealth.lastAttempt
	if health.LastError == "" {
		health.LastError = nodeOutboxHealth.lastError
	}
	nodeOutboxHealth.RUnlock()
	health.Healthy = health.LastError == ""
	return health, nil
}

// RunNodeAccessOutboxWorker retries due ClickHouse deliveries until ctx is canceled.
func RunNodeAccessOutboxWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second * 30
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if store, err := Active(ctx); err == nil {
			if _, ok := store.AccessLogs.(*clickhouseLogStore); ok {
				_ = flushNodeOutbox(ctx, getGormDB(ctx), deliverNodeChanges)
			}
		} else {
			nodeOutboxHealth.Lock()
			nodeOutboxHealth.lastAttempt = time.Now().UTC()
			nodeOutboxHealth.lastError = err.Error()
			nodeOutboxHealth.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

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
	nodeOutboxHealth.Lock()
	nodeOutboxHealth.lastAttempt = time.Now().UTC()
	nodeOutboxHealth.Unlock()
	defer func() {
		nodeOutboxHealth.Lock()
		if result != nil {
			nodeOutboxHealth.lastError = result.Error()
		} else {
			nodeOutboxHealth.lastError = ""
		}
		nodeOutboxHealth.Unlock()
	}()
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
		if err := db.WithContext(ctx).Where("next_attempt_at <= ?", time.Now().UTC()).Order("id").Limit(1).Find(&page).Error; err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		var delivery nodeDelivery
		if err := json.Unmarshal([]byte(page[0].Payload), &delivery); err != nil {
			attempts := page[0].Attempts + 1
			next := time.Now().UTC().Add(outboxRetryDelay(attempts))
			if updateErr := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessOutbox{}).Where("id = ?", page[0].ID).Updates(map[string]any{
				"attempts": attempts, "next_attempt_at": next, "last_error": "invalid outbox payload",
			}).Error; updateErr != nil {
				return errors.Join(err, updateErr)
			}
			continue
		}
		if err := deliver(ctx, delivery); err != nil {
			attempts := page[0].Attempts + 1
			next := time.Now().UTC().Add(outboxRetryDelay(attempts))
			if updateErr := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessOutbox{}).Where("id = ?", page[0].ID).Updates(map[string]any{
				"attempts": attempts, "next_attempt_at": next, "last_error": err.Error(),
			}).Error; updateErr != nil {
				return errors.Join(err, updateErr)
			}
			return err
		}
		if err := db.WithContext(ctx).Where("id = ?", page[0].ID).Delete(&analyticsmodel.NodeAccessOutbox{}).Error; err != nil {
			return err
		}
	}
	return fmt.Errorf("node outbox drain exceeded per-run page budget at %s", time.Now().UTC().Format(time.RFC3339))
}

func outboxRetryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 8 {
		attempts = 8
	}
	return time.Second * time.Duration(1<<(attempts-1))
}

func parseOutboxTime(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse outbox time %q", value)
}
