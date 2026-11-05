// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package service_test

import (
	"Wavelet/core/contracts"
	"Wavelet/plugins/domain/admin/model"
	"Wavelet/plugins/domain/admin/service"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type forceCleanupLogStorage struct {
	month         time.Time
	before        time.Time
	calls         int
	deleted       int64
	rows          int64
	bytes         int64
	hasMore       bool
	progressCalls int
}

func (s *forceCleanupLogStorage) Overview(context.Context) (contracts.LogStorageOverview, error) {
	return contracts.LogStorageOverview{}, nil
}

func (s *forceCleanupLogStorage) Cleanup(_ context.Context, category string, before *time.Time) ([]contracts.LogStorageCleanupResult, error) {
	return nil, errors.New("unexpected generic cleanup")
}

func (s *forceCleanupLogStorage) DeleteNodeAccessLogBatch(_ context.Context, month, before time.Time, limit int) (int64, error) {
	s.calls++
	s.month = month
	s.before = before
	if limit != 1000 {
		return 0, errors.New("unexpected batch size")
	}
	if s.deleted > 0 {
		deleted := min(s.deleted, int64(limit))
		s.deleted -= deleted
		return deleted, nil
	}
	return 0, nil
}

func (s *forceCleanupLogStorage) HasNodeAccessLogsBefore(_ context.Context, month, before time.Time) (bool, error) {
	s.progressCalls++
	if !month.Equal(s.month) || !before.Equal(s.before) {
		return false, errors.New("progress scope differs from deletion")
	}
	return s.hasMore, nil
}

func (s *forceCleanupLogStorage) NodeAccessLogPartitionStats(context.Context, time.Time) (int64, int64, error) {
	return 117712744, 45600849920, nil
}

func TestForceLogCleanupHandler_Execute(t *testing.T) {
	previousDBConfig := service.GetDBConfig()
	service.SetDBConfig(model.DatabaseConfig{Enabled: true})
	t.Cleanup(func() { service.SetDBConfig(previousDBConfig) })
	storage := &forceCleanupLogStorage{deleted: 1000000, hasMore: true}
	service.SetLogStorageService(storage)
	t.Cleanup(service.ResetServices)
	taskSvc := &cleanupTaskService{}
	service.SetTaskService(taskSvc)

	payload, err := (&service.ForceLogCleanupHandler{}).ValidatePayload([]byte(`{"confirmed":true,"month":"2026-09"}`))
	require.NoError(t, err)
	_, err = (&service.ForceLogCleanupHandler{}).ValidatePayload([]byte(`{"confirmed":false}`))
	require.Error(t, err)
	result, err := (&service.ForceLogCleanupHandler{}).Execute(context.Background(), payload)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), storage.month)
	assert.Equal(t, 1000, storage.calls)
	assert.WithinDuration(t, time.Now().UTC().AddDate(0, 0, -7), storage.before, time.Second)
	assert.Contains(t, result.Message, "2026-09")
	assert.Less(t, len(taskSvc.logs), 1000)
	detail := result.Detail.(map[string]any)
	assert.Equal(t, int64(1000000), detail["deleted"])
	assert.Equal(t, true, detail["has_more"])
	assert.Equal(t, 1, storage.progressCalls)
	storage.calls = 0
	storage.deleted = 1000000
	storage.hasMore = false
	result, err = (&service.ForceLogCleanupHandler{}).Execute(context.Background(), payload)
	require.NoError(t, err)
	assert.Equal(t, false, result.Detail.(map[string]any)["has_more"])
	storage.calls = 0
	storage.deleted = 12
	storage.hasMore = true
	result, err = (&service.ForceLogCleanupHandler{}).Execute(context.Background(), payload)
	require.NoError(t, err)
	assert.Equal(t, 1, storage.calls)
	assert.Equal(t, int64(12), result.Detail.(map[string]any)["deleted"])
	assert.Equal(t, true, result.Detail.(map[string]any)["has_more"])
}

type cleanupTaskService struct{ logs []string }

func TestForceLogCleanupHandlerRejectsInvalidPayload(t *testing.T) {
	handler := &service.ForceLogCleanupHandler{}
	for _, payload := range []string{
		`{"confirmed":true}`,
		`{"confirmed":false,"month":"2020-01"}`,
		`{"confirmed":true,"month":"2020-13"}`,
		`{"confirmed":true,"month":"2020-1"}`,
		`{"confirmed":true,"month":"0000-01"}`,
		`{"confirmed":true,"month":"2020-01; DROP TABLE logs"}`,
		`{`,
	} {
		if _, err := handler.ValidatePayload([]byte(payload)); err == nil {
			t.Errorf("ValidatePayload(%q) error = nil, want error", payload)
		}
		if _, err := handler.Execute(context.Background(), []byte(payload)); err == nil {
			t.Errorf("Execute(%q) error = nil, want error", payload)
		}
	}
}

func (s *cleanupTaskService) Dispatch(context.Context, string, []byte, string) (string, error) {
	return "", nil
}
func (s *cleanupTaskService) Retry(context.Context, uint64) (string, error) { return "", nil }
func (s *cleanupTaskService) ListTasks() []contracts.TaskMetaDTO            { return nil }
func (s *cleanupTaskService) GetTaskMeta(string) (contracts.TaskMetaDTO, bool) {
	return contracts.TaskMetaDTO{}, false
}
func (s *cleanupTaskService) ValidatePayload(_ string, payload []byte) ([]byte, error) {
	return payload, nil
}
func (s *cleanupTaskService) ReloadScheduler() error { return nil }
func (s *cleanupTaskService) AppendLog(_ context.Context, format string, args ...any) {
	s.logs = append(s.logs, fmt.Sprintf(format, args...))
}
func (s *cleanupTaskService) ListExecutions(context.Context, string, string, int, int) ([]contracts.TaskExecutionDTO, int64, error) {
	return nil, 0, nil
}
func (s *cleanupTaskService) GetExecution(context.Context, uint64) (*contracts.TaskExecutionDTO, error) {
	return nil, nil
}
func (s *cleanupTaskService) GetExecutionByTaskID(context.Context, string) (*contracts.TaskExecutionDTO, error) {
	return nil, nil
}

func TestForceLogCleanupHandler_RequiresLogStorage(t *testing.T) {
	service.ResetServices()
	_, err := (&service.ForceLogCleanupHandler{}).Execute(context.Background(), []byte(`{"confirmed":true,"month":"2026-10"}`))
	assert.Error(t, err)
}

func TestSystemCleanupHandler_Execute(t *testing.T) {
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, sqliteDB.AutoMigrate(&model.TaskExecution{}))

	service.SetDBService(&testDBService{db: sqliteDB})
	defer service.ResetServices()

	now := time.Now()
	oldTime := now.Add(-4 * 24 * time.Hour)
	recentTime := now.Add(-1 * time.Hour)

	// Seed old execution (should be cleaned)
	oldExec := model.TaskExecution{
		ID:        1,
		TaskID:    "task-old",
		TaskType:  "sample_task",
		Status:    model.TaskExecutionStatusSucceeded,
		CreatedAt: oldTime,
	}
	require.NoError(t, sqliteDB.Create(&oldExec).Error)

	// Seed recent execution (should remain)
	recentExec := model.TaskExecution{
		ID:        2,
		TaskID:    "task-recent",
		TaskType:  "sample_task",
		Status:    model.TaskExecutionStatusSucceeded,
		CreatedAt: recentTime,
	}
	require.NoError(t, sqliteDB.Create(&recentExec).Error)

	var eventFired atomic.Bool
	service.SetEventEmitter(func(ctx context.Context, topic string, payload any) error {
		if topic == contracts.EventTopicSystemCleanup {
			eventFired.Store(true)
		}
		return nil
	})

	handler := &service.SystemCleanupHandler{}
	res, err := handler.Execute(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Contains(t, res.Message, "系统垃圾清理完成")

	// Verify old execution is deleted and recent remains
	var count int64
	sqliteDB.Model(&model.TaskExecution{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var remaining model.TaskExecution
	sqliteDB.First(&remaining)
	assert.Equal(t, uint64(2), remaining.ID)

	assert.True(t, eventFired.Load(), "EventTopicSystemCleanup must be emitted")
}
