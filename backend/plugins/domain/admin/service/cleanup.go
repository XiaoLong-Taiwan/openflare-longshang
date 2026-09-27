// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"Wavelet/core/contracts"
	"Wavelet/pkg/logger"
	"Wavelet/plugins/domain/admin/model"
	"Wavelet/plugins/domain/admin/repository"
	pkgcache "Wavelet/pkg/cache/disk"
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// SystemCleanupTask 系统定期垃圾清理任务标识
	SystemCleanupTask = "system:cleanup"
	// TaskTypeSystemCleanup 系统定期垃圾清理管理类型
	TaskTypeSystemCleanup = "system_cleanup"
	taskQueueDefault      = "default"
)

// SystemCleanupMeta describes the system-wide cleanup task metadata.
var SystemCleanupMeta = contracts.TaskMetaDTO{
	Type:         TaskTypeSystemCleanup,
	AsynqTask:    SystemCleanupTask,
	Name:         "系统垃圾清理",
	DisplayName:  "系统垃圾清理",
	Description:  "定期清理过期任务执行记录，并通过领域事件广播触发各业务域自治清理（临时文件、历史推送等）",
	Category:     "maintenance",
	SupportsTime: false,
	MaxRetry:     3,
	Queue:        taskQueueDefault,
	Retryable:    true,
}

// DatabaseMaintenanceTask identifies the full database maintenance task.
const DatabaseMaintenanceTask = "database:maintenance"

// TaskTypeDatabaseMaintenance identifies the admin task metadata type.
const TaskTypeDatabaseMaintenance = "database_maintenance"

// DatabaseMaintenanceMeta describes the full database maintenance task.
var DatabaseMaintenanceMeta = contracts.TaskMetaDTO{
	Type:        TaskTypeDatabaseMaintenance,
	AsynqTask:   DatabaseMaintenanceTask,
	Name:        "資料庫完整維護",
	DisplayName: "資料庫完整維護",
	Description: "清理過期任務記錄、清空磁碟快取並更新資料庫統計與空間",
	Category:    "maintenance",
	MaxRetry:    1,
	Queue:       taskQueueDefault,
	Retryable:   true,
}

// SystemCleanupHandler handles the system-wide garbage cleanup task.
type SystemCleanupHandler struct{}

// PreviewDatabaseMaintenance reports the pending cleanup and maintenance operations.
func PreviewDatabaseMaintenance(ctx context.Context) (model.DatabaseMaintenancePreview, error) {
	before := time.Now().Add(-7 * 24 * time.Hour)
	count, err := repository.CountExpiredTaskExecutions(ctx, before)
	if err != nil {
		return model.DatabaseMaintenancePreview{}, err
	}
	status := pkgcache.Default().Status()
	postgres := GetDBConfig().Enabled
	operations := []string{"clear disk cache", "analyze", "vacuum"}
	if postgres {
		operations = []string{"clear disk cache", "reindex schema public", "analyze", "vacuum"}
	}
	return model.DatabaseMaintenancePreview{
		DatabaseType:          map[bool]string{true: "postgres", false: "sqlite"}[postgres],
		ExpiredTaskExecutions: count,
		CacheKeys:             status.KeysCount,
		CacheBytes:            status.TotalSize,
		Operations:            operations,
	}, nil
}

// RunDatabaseMaintenance performs the confirmed cleanup and database maintenance.
func RunDatabaseMaintenance(ctx context.Context) (model.DatabaseMaintenanceResult, error) {
	before := time.Now().Add(-7 * 24 * time.Hour)
	if err := EmitEvent(ctx, contracts.EventTopicSystemCleanup, contracts.SystemCleanupEvent{
		TriggeredAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return model.DatabaseMaintenanceResult{}, err
	}
	deleted, err := repository.DeleteExpiredTaskExecutions(ctx, before)
	if err != nil {
		return model.DatabaseMaintenanceResult{}, err
	}
	if err := pkgcache.Default().Clear(); err != nil {
		return model.DatabaseMaintenanceResult{ExpiredTaskExecutions: deleted}, err
	}
	operations, err := repository.OptimizeDatabase(ctx, GetDBConfig().Enabled)
	if err != nil {
		return model.DatabaseMaintenanceResult{ExpiredTaskExecutions: deleted, CacheCleared: true, Operations: operations}, err
	}
	logger.InfoF(ctx, "資料庫完整維護完成，清理任務記錄 %d 條", deleted)
	return model.DatabaseMaintenanceResult{
		ExpiredTaskExecutions: deleted,
		CacheCleared:          true,
		DatabaseOptimized:     true,
		Operations:            operations,
	}, nil
}

// DatabaseMaintenanceHandler handles the full database maintenance task.
type DatabaseMaintenanceHandler struct{}

// Execute runs the full database maintenance task.
func (h *DatabaseMaintenanceHandler) Execute(ctx context.Context, _ []byte) (*contracts.TaskResultDTO, error) {
	result, err := RunDatabaseMaintenance(ctx)
	if err != nil {
		return nil, err
	}
	return &contracts.TaskResultDTO{Message: "資料庫完整維護完成", Detail: result}, nil
}

// Execute executes system cleanup: clears old task executions and emits EventTopicSystemCleanup.
func (h *SystemCleanupHandler) Execute(ctx context.Context, _ []byte) (*contracts.TaskResultDTO, error) {
	db := GetDB(ctx)
	if db == nil {
		return nil, errors.New("database service not available")
	}

	// 1. 清理自身域（admin 域）的过期任务执行记录（7 天前）
	var deletedExecutions int64
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	res := db.Where("created_at < ?", sevenDaysAgo).Delete(&model.TaskExecution{})
	if err := res.Error; err != nil {
		logger.WarnF(ctx, "清理过期任务执行日志失败: %v", err)
	} else {
		deletedExecutions = res.RowsAffected
		logger.InfoF(ctx, "已清理 7 天前任务执行日志，共 %d 条", deletedExecutions)
	}

	// 2. 广播 EventTopicSystemCleanup 领域事件，由各业务域插件（upload, msg_gateway, user 等）自治执行各自的清理逻辑
	nowStr := time.Now().Format(time.RFC3339)
	if err := EmitEvent(ctx, contracts.EventTopicSystemCleanup, contracts.SystemCleanupEvent{
		TriggeredAt: nowStr,
	}); err != nil {
		logger.WarnF(ctx, "广播系统清理领域事件失败: %v", err)
	}

	msg := fmt.Sprintf("系统垃圾清理完成，已清理过期任务执行日志 %d 条，并已广播领域清理事件", deletedExecutions)
	logger.InfoF(ctx, "%s", msg)
	return &contracts.TaskResultDTO{Message: msg}, nil
}
