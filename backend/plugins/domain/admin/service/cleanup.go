// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"Wavelet/core/contracts"
	pkgcache "Wavelet/pkg/cache/disk"
	"Wavelet/pkg/logger"
	"Wavelet/plugins/domain/admin/model"

	"Wavelet/plugins/domain/admin/repository"
)

const (
	// SystemCleanupTask 系统定期垃圾清理任务标识
	SystemCleanupTask   = "system:cleanup"
	ForceLogCleanupTask = "system:force-log-cleanup"
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

var ForceLogCleanupMeta = contracts.TaskMetaDTO{
	Type:         "force_log_cleanup",
	AsynqTask:    ForceLogCleanupTask,
	Name:         "清理 2026-10 節點存取日誌",
	DisplayName:  "清理 2026-10 節點存取日誌",
	Description:  "每批刪除最多 1,000 筆七天前資料，並記錄清理進度；不會直接刪除整個月份分區",
	Category:     "maintenance",
	SupportsTime: false,
	MaxRetry:     3,
	Queue:        taskQueueDefault,
	Retryable:    true,
	Params: []contracts.TaskParamDTO{{
		Name:        "confirmed",
		Label:       "確認清理十月分區",
		Type:        "boolean",
		Required:    true,
		Description: "確認每批最多刪除 1,000 筆七天以前的資料。",
	}},
}

type forceLogCleanupHandlerPayload struct {
	Confirmed bool `json:"confirmed"`
}

var forceLogCleanupPayload = []byte(`{"confirmed":true}`)

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
	Timeout:     30 * time.Minute,
	Queue:       taskQueueDefault,
	Retryable:   true,
}

// SystemCleanupHandler handles the system-wide garbage cleanup task.
type SystemCleanupHandler struct{}

type ForceLogCleanupHandler struct{}

// PreviewDatabaseMaintenance reports the pending cleanup and maintenance operations.
func PreviewDatabaseMaintenance(ctx context.Context) (model.DatabaseMaintenancePreview, error) {
	count, err := repository.CountExpiredTaskExecutions(ctx, time.Now())
	if err != nil {
		return model.DatabaseMaintenancePreview{}, err
	}
	status := pkgcache.Default().Status()
	postgres := GetDBConfig().Enabled
	operations := []string{"clear disk cache", "analyze", "vacuum"}
	if postgres {
		operations = []string{"clear disk cache", "reindex schema public", "analyze", "vacuum full"}
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
	if err := EmitEvent(ctx, contracts.EventTopicSystemCleanup, contracts.SystemCleanupEvent{
		TriggeredAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return model.DatabaseMaintenanceResult{}, err
	}
	stats, err := repository.CleanupTaskExecutionLogs(ctx, time.Now())
	if err != nil {
		return model.DatabaseMaintenanceResult{}, err
	}
	deleted := stats.HighFrequencyDeleted + stats.LowFrequencyDeleted
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

	// 1. 清理自身域（admin 域）的过期任务执行记录
	var deletedExecutions int64
	stats, cleanupErr := repository.CleanupTaskExecutionLogs(ctx, time.Now())
	if cleanupErr != nil {
		logger.WarnF(ctx, "清理过期任务执行日志失败: %v", cleanupErr)
	} else {
		deletedExecutions = stats.HighFrequencyDeleted + stats.LowFrequencyDeleted
		logger.InfoF(ctx, "已清理过期任务执行日志，共 %d 条", deletedExecutions)
	}
	// 2. 广播 EventTopicSystemCleanup 领域事件，由各业务域插件（upload, msg_gateway, user 等）自治执行各自的清理逻辑
	nowStr := time.Now().Format(time.RFC3339)
	if err := EmitEvent(ctx, contracts.EventTopicSystemCleanup, contracts.SystemCleanupEvent{
		TriggeredAt: nowStr,
	}); err != nil {
		return nil, fmt.Errorf("system cleanup listeners failed: %w", err)
	}
	if err := cleanupErr; err != nil {
		return nil, fmt.Errorf("task execution cleanup failed: %w", err)
	}

	msg := fmt.Sprintf("系统垃圾清理完成，已清理过期任务执行日志 %d 条，并已广播领域清理事件", deletedExecutions)
	logger.InfoF(ctx, "%s", msg)
	return &contracts.TaskResultDTO{Message: msg}, nil
}

func (h *ForceLogCleanupHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var request forceLogCleanupHandlerPayload
	if err := json.Unmarshal(payload, &request); err != nil || !request.Confirmed {
		return nil, errors.New("confirmed must be true")
	}
	return []byte(`{"confirmed":true}`), nil
}

func (h *ForceLogCleanupHandler) Execute(ctx context.Context, payload []byte) (*contracts.TaskResultDTO, error) {
	var request forceLogCleanupHandlerPayload
	if err := json.Unmarshal(payload, &request); err != nil || !request.Confirmed {
		return nil, errors.New("confirmed must be true")
	}
	storage := GetLogStorageService(ctx)
	if storage == nil {
		return nil, errLogStorageUnavailable
	}
	taskSvc := GetTaskService(ctx)
	if taskSvc == nil {
		return nil, errors.New("task service is not available for cleanup progress logging")
	}
	if !GetDBConfig().Enabled {
		return nil, errors.New("monthly 202610 cleanup requires PostgreSQL")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -7)
	month := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	estimatedInitialRows, estimatedInitialBytes, err := storage.NodeAccessLogPartitionStats(ctx, month)
	if err != nil {
		return nil, fmt.Errorf("read October partition statistics: %w", err)
	}
	taskSvc.AppendLog(ctx, "開始清理 of_node_access_logs_202610：目前統計估計約 %.2f GB、%s 筆；筆數為 PostgreSQL 統計估值。每批最多 1,000 筆，保留 cutoff 之後資料（cutoff=%s）。", float64(estimatedInitialBytes)/1e9, strconv.FormatInt(estimatedInitialRows, 10), cutoff.Format(time.RFC3339))
	var deletedTotal int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		deleted, err := storage.DeleteNodeAccessLogBatch(ctx, month, cutoff, 1000)
		if err != nil {
			return nil, fmt.Errorf("delete October 2026 access log batch: %w", err)
		}
		if deleted == 0 {
			break
		}
		deletedTotal += deleted
		taskSvc.AppendLog(ctx, "of_node_access_logs_202610 本批刪除 %d 筆，累計刪除 %s 筆；啟動時估計 %s 筆。", deleted, strconv.FormatInt(deletedTotal, 10), strconv.FormatInt(estimatedInitialRows, 10))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	taskSvc.AppendLog(ctx, "十月分區已無 cutoff 前的待刪記錄。本次刪除 %d 筆；PostgreSQL 檔案空間未必立即縮小，VACUUM FULL 需另行於維護窗執行。", deletedTotal)
	return &contracts.TaskResultDTO{Message: fmt.Sprintf("十月分區過期存取日誌清理完成，共刪除 %d 筆（開始時分區統計估值約 %.2f GB）", deletedTotal, float64(estimatedInitialBytes)/1e9), Detail: map[string]any{"partition": "of_node_access_logs_202610", "cutoff": cutoff, "deleted": deletedTotal, "initial_estimated_rows": estimatedInitialRows, "initial_estimated_bytes": estimatedInitialBytes}}, nil
}
