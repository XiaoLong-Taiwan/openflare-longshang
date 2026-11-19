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
	Pending            int64     `json:"pending"`
	OldestAt           time.Time `json:"oldest_at,omitempty"`
	PendingAgeSeconds  int64     `json:"pending_age_seconds,omitempty"`
	LastError          string    `json:"last_error,omitempty"`
	LastAttempt        time.Time `json:"last_attempt,omitempty"`
	LastAttemptOutcome string    `json:"last_attempt_outcome,omitempty"`
	LastSuccess        time.Time `json:"last_success,omitempty"`
	Attempts           int       `json:"attempts,omitempty"`
	NextAttemptAt      time.Time `json:"next_attempt_at,omitempty"`
	Backend            string    `json:"backend,omitempty"`
	WorkerRunning      bool      `json:"worker_running"`
	Healthy            bool      `json:"healthy"`
}

var nodeOutboxHealth struct {
	sync.RWMutex
	lastAttempt        time.Time
	lastAttemptOutcome string
	lastSuccess        time.Time
	lastError          string
	workerRunning      bool
}

// GetNodeAccessOutboxHealth returns durable queue depth and the latest retry failure.
func GetNodeAccessOutboxHealth(ctx context.Context) (NodeAccessOutboxHealth, error) {
	db := getGormDB(ctx)
	if db == nil {
		return NodeAccessOutboxHealth{}, fmt.Errorf("node outbox database is not initialized")
	}
	var health NodeAccessOutboxHealth
	var pending struct {
		Pending       int64
		CreatedAt     time.Time
		LastError     string
		Attempts      int
		NextAttemptAt time.Time
	}
	if err := db.WithContext(ctx).Model(&analyticsmodel.NodeAccessOutbox{}).
		Select("COUNT(*) OVER () AS pending, created_at, last_error, attempts, next_attempt_at").
		Order("created_at ASC, id ASC").Limit(1).Scan(&pending).Error; err != nil {
		return health, err
	}
	health.Pending = pending.Pending
	if pending.Pending > 0 {
		health.OldestAt = pending.CreatedAt
		health.PendingAgeSeconds = max(0, int64(time.Since(pending.CreatedAt).Seconds()))
		health.LastError = pending.LastError
		health.Attempts = pending.Attempts
		health.NextAttemptAt = pending.NextAttemptAt
	}
	nodeOutboxHealth.RLock()
	health.LastAttempt = nodeOutboxHealth.lastAttempt
	health.LastAttemptOutcome = nodeOutboxHealth.lastAttemptOutcome
	health.LastSuccess = nodeOutboxHealth.lastSuccess
	health.WorkerRunning = nodeOutboxHealth.workerRunning
	if health.LastError == "" {
		health.LastError = nodeOutboxHealth.lastError
	}
	nodeOutboxHealth.RUnlock()
	if backend, err := ActiveDatabase(ctx); err == nil {
		health.Backend = backend
	}
	health.Healthy = health.Pending == 0 && health.LastError == "" && health.WorkerRunning
	return health, nil
}

// RunNodeAccessOutboxWorker retries due ClickHouse deliveries until ctx is canceled.
func RunNodeAccessOutboxWorker(ctx context.Context, interval time.Duration) {
	nodeOutboxHealth.Lock()
	nodeOutboxHealth.workerRunning = true
	nodeOutboxHealth.Unlock()
	defer func() {
		nodeOutboxHealth.Lock()
		nodeOutboxHealth.workerRunning = false
		nodeOutboxHealth.Unlock()
	}()
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
			nodeOutboxHealth.lastAttemptOutcome = "error"
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
	var attemptedAt time.Time
	var deliveryErr error
	defer func() {
		if attemptedAt.IsZero() && result == nil {
			return
		}
		nodeOutboxHealth.Lock()
		defer nodeOutboxHealth.Unlock()
		if attemptedAt.IsZero() {
			attemptedAt = time.Now().UTC()
		}
		nodeOutboxHealth.lastAttempt = attemptedAt
		if err := errors.Join(result, deliveryErr); err != nil {
			nodeOutboxHealth.lastAttemptOutcome = "error"
			nodeOutboxHealth.lastError = err.Error()
		} else {
			nodeOutboxHealth.lastAttemptOutcome = "success"
			nodeOutboxHealth.lastError = ""
		}
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
		attemptedAt = time.Now().UTC()
		var delivery nodeDelivery
		if err := json.Unmarshal([]byte(page[0].Payload), &delivery); err != nil {
			deliveryErr = err
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
		deliveryErr = nil
		nodeOutboxHealth.Lock()
		nodeOutboxHealth.lastSuccess = time.Now().UTC()
		nodeOutboxHealth.Unlock()
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
