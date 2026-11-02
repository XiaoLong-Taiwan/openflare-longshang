package repository_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"Wavelet/plugins/domain/admin/errs"
	"Wavelet/plugins/domain/admin/model"
	"Wavelet/plugins/domain/admin/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTaskCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(:memory:) error = %v, want nil", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		repository.SetDBService(nil)
		if err := sqlDB.Close(); err != nil {
			t.Errorf("sqlDB.Close() error = %v, want nil", err)
		}
	})
	if err := db.AutoMigrate(&model.TaskExecution{}); err != nil {
		t.Fatalf("AutoMigrate(TaskExecution) error = %v, want nil", err)
	}
	repository.SetDBService(stubDBService{db: db})
	return db
}

func checkTaskCleanupBatches(t *testing.T, succeededCutoffDays, failedCutoffDays int, cleanup func(context.Context, time.Time) (int64, error)) {
	t.Helper()
	db := newTaskCleanupTestDB(t)
	cutoff := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for statusIndex, status := range []model.TaskExecutionStatus{model.TaskExecutionStatusSucceeded, model.TaskExecutionStatusFailed} {
		rows := make([]map[string]any, 2005)
		for i := range rows {
			createdAt := cutoff.AddDate(0, 0, -failedCutoffDays-1)
			if status == model.TaskExecutionStatusSucceeded {
				createdAt = cutoff.AddDate(0, 0, -succeededCutoffDays-1)
			}
			rows[i] = map[string]any{"task_id": fmt.Sprintf("expired-%d-%d", statusIndex, i), "task_type": "test", "task_name": "test", "status": status, "created_at": createdAt}
		}
		if err := db.Table("w_task_executions").CreateInBatches(rows, 100).Error; err != nil {
			t.Fatalf("seed expired %s tasks error = %v, want nil", status, err)
		}
	}
	retained := []map[string]any{
		{"task_id": "pending", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusPending, "created_at": cutoff.AddDate(0, 0, -90)},
		{"task_id": "running", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusRunning, "created_at": cutoff.AddDate(0, 0, -90)},
		{"task_id": "succeeded-boundary", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusSucceeded, "created_at": cutoff.AddDate(0, 0, -succeededCutoffDays)},
		{"task_id": "failed-boundary", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusFailed, "created_at": cutoff.AddDate(0, 0, -failedCutoffDays)},
		{"task_id": "succeeded-recent", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusSucceeded, "created_at": cutoff.AddDate(0, 0, 1-succeededCutoffDays)},
		{"task_id": "failed-recent", "task_type": "test", "task_name": "test", "status": model.TaskExecutionStatusFailed, "created_at": cutoff.AddDate(0, 0, 1-failedCutoffDays)},
	}
	if err := db.Table("w_task_executions").Create(&retained).Error; err != nil {
		t.Fatalf("seed retained tasks error = %v, want nil", err)
	}
	var batches []int64
	if err := db.Callback().Delete().After("gorm:delete").Register("test:task_cleanup_batches", func(tx *gorm.DB) {
		if tx.Error == nil {
			batches = append(batches, tx.RowsAffected)
		}
	}); err != nil {
		t.Fatalf("register delete callback error = %v, want nil", err)
	}
	total, err := cleanup(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("cleanup(%v) error = %v, want nil", cutoff, err)
	}
	if total != 4010 {
		t.Errorf("cleanup(%v) deleted = %d, want 4010", cutoff, total)
	}
	if want := []int64{1000, 1000, 5, 1000, 1000, 5}; !reflect.DeepEqual(batches, want) {
		t.Errorf("cleanup(%v) batches = %v, want %v", cutoff, batches, want)
	}
	var remaining []model.TaskExecution
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("query retained tasks error = %v, want nil", err)
	}
	if len(remaining) != len(retained) {
		t.Fatalf("cleanup(%v) remaining count = %d, want %d", cutoff, len(remaining), len(retained))
	}
	for i, row := range remaining {
		if row.Status != retained[i]["status"] || !row.CreatedAt.Equal(retained[i]["created_at"].(time.Time)) {
			t.Errorf("cleanup(%v) retained row %d = (%s, %v), want (%s, %v)", cutoff, row.ID, row.Status, row.CreatedAt, retained[i]["status"], retained[i]["created_at"])
		}
	}
	total, err = cleanup(context.Background(), cutoff)
	if err != nil || total != 0 {
		t.Errorf("cleanup(%v) repeated = (%d, %v), want (0, nil)", cutoff, total, err)
	}
}

func TestCleanupTaskExecutionLogsBatches(t *testing.T) {
	checkTaskCleanupBatches(t, 3, 30, func(ctx context.Context, cutoff time.Time) (int64, error) {
		stats, err := repository.CleanupTaskExecutionLogs(ctx, cutoff)
		if err == nil && (stats.HighFrequencyDeleted != stats.LowFrequencyDeleted) {
			return 0, fmt.Errorf("cleanup stats = %+v, want equal succeeded and failed counts", stats)
		}
		return stats.HighFrequencyDeleted + stats.LowFrequencyDeleted, err
	})
}

func TestDeleteExpiredTaskExecutionsBatches(t *testing.T) {
	checkTaskCleanupBatches(t, 3, 7, func(ctx context.Context, cutoff time.Time) (int64, error) {
		return repository.DeleteExpiredTaskExecutions(ctx, cutoff.AddDate(0, 0, -7))
	})
}

func TestTaskCleanupNilDB(t *testing.T) {
	repository.SetDBService(nil)
	t.Cleanup(func() { repository.SetDBService(nil) })
	now := time.Now().UTC()
	stats, err := repository.CleanupTaskExecutionLogs(context.Background(), now)
	if !errors.Is(err, errs.ErrDatabaseUninitialized) || stats != (model.TaskExecutionCleanupStats{}) {
		t.Errorf("CleanupTaskExecutionLogs(nil DB) = (%+v, %v), want (zero stats, %v)", stats, err, errs.ErrDatabaseUninitialized)
	}
	total, err := repository.DeleteExpiredTaskExecutions(context.Background(), now)
	if !errors.Is(err, errs.ErrDatabaseUninitialized) || total != 0 {
		t.Errorf("DeleteExpiredTaskExecutions(nil DB) = (%d, %v), want (0, %v)", total, err, errs.ErrDatabaseUninitialized)
	}
}

func TestTaskCleanupPreviewMatchesRetention(t *testing.T) {
	db := newTaskCleanupTestDB(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	rows := []model.TaskExecution{
		{TaskID: "succeeded-expired", Status: model.TaskExecutionStatusSucceeded, CreatedAt: now.AddDate(0, 0, -4)},
		{TaskID: "succeeded-boundary", Status: model.TaskExecutionStatusSucceeded, CreatedAt: now.AddDate(0, 0, -3)},
		{TaskID: "failed-expired", Status: model.TaskExecutionStatusFailed, CreatedAt: now.AddDate(0, 0, -31)},
		{TaskID: "failed-boundary", Status: model.TaskExecutionStatusFailed, CreatedAt: now.AddDate(0, 0, -30)},
		{TaskID: "failed-recent", Status: model.TaskExecutionStatusFailed, CreatedAt: now.AddDate(0, 0, -10)},
		{TaskID: "pending", Status: model.TaskExecutionStatusPending, CreatedAt: now.AddDate(0, 0, -90)},
		{TaskID: "running", Status: model.TaskExecutionStatusRunning, CreatedAt: now.AddDate(0, 0, -90)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed task preview rows: %v", err)
	}
	count, err := repository.CountExpiredTaskExecutions(ctx, now)
	if err != nil || count != 2 {
		t.Fatalf("CountExpiredTaskExecutions(%v) = (%d, %v), want (2, nil)", now, count, err)
	}
	var before int64
	if err := db.Model(&model.TaskExecution{}).Count(&before).Error; err != nil || before != int64(len(rows)) {
		t.Fatalf("preview remaining rows = (%d, %v), want (%d, nil)", before, err, len(rows))
	}
	stats, err := repository.CleanupTaskExecutionLogs(ctx, now)
	if err != nil || stats.HighFrequencyDeleted+stats.LowFrequencyDeleted != count {
		t.Errorf("CleanupTaskExecutionLogs(%v) = (%+v, %v), want %d deleted and nil error", now, stats, err, count)
	}
	count, err = repository.CountExpiredTaskExecutions(ctx, now)
	if err != nil || count != 0 {
		t.Errorf("CountExpiredTaskExecutions(%v) after cleanup = (%d, %v), want (0, nil)", now, count, err)
	}
}
