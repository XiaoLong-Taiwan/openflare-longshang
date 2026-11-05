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
	TaskTypeSystemCleanup     = "system_cleanup"
	taskQueueDefault          = "default"
	forceLogCleanupBatchLimit = 1000000
	forceLogCleanupBatchSize  = 1000
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
	Name:         "月份日誌清理與彙總壓縮",
	DisplayName:  "月份日誌清理與彙總壓縮",
	Description:  "指定 YYYY-MM 月份，先保留節點存取日誌摘要，再分批刪除七天前明細；每批最多 1,000 筆，每工作最多 1,000,000 筆，並回報實際進度。彙總壓縮不代表 VACUUM 縮檔",
	Category:     "maintenance",
	SupportsTime: false,
	MaxRetry:     3,
	Queue:        taskQueueDefault,
	Retryable:    true,
	Timeout:      24 * time.Hour,
	Params: []contracts.TaskParamDTO{{
		Name:        "month",
		Label:       "清理與彙總壓縮月份（YYYY-MM）",
		Type:        "string",
		Required:    true,
		Description: "輸入要清理與彙總壓縮的節點存取日誌 UTC 月份，格式為 YYYY-MM。",
	}, {
		Name:        "confirmed",
		Label:       "確認清理與彙總壓縮指定月份",
		Type:        "boolean",
		Required:    true,
		Description: "確認先保留摘要，再以每批最多 1,000 筆刪除指定月份七天前明細；不執行 VACUUM 縮檔。",
	}},
}

type forceLogCleanupHandlerPayload struct {
	Confirmed bool   `json:"confirmed"`
	Month     string `json:"month"`
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
	month, err := time.Parse("2006-01", request.Month)
	if err != nil || month.Year() < 1 || month.Year() > 9999 {
		return nil, errors.New("month must be YYYY-MM")
	}
	return json.Marshal(request)
}

func (h *ForceLogCleanupHandler) Execute(ctx context.Context, payload []byte) (*contracts.TaskResultDTO, error) {
	validated, err := h.ValidatePayload(payload)
	if err != nil {
		return nil, err
	}
	var request forceLogCleanupHandlerPayload
	if err := json.Unmarshal(validated, &request); err != nil {
		return nil, err
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
		return nil, errors.New("monthly cleanup requires PostgreSQL")
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -7)
	month, err := time.Parse("2006-01", request.Month)
	if err != nil {
		return nil, err
	}
	partition := "of_node_access_logs_" + month.Format("200601")
	estimatedInitialRows, estimatedInitialBytes, err := storage.NodeAccessLogPartitionStats(ctx, month)
	if err != nil {
		return nil, fmt.Errorf("read partition statistics: %w", err)
	}
	taskSvc.AppendLog(ctx, "開始 %s 月份日誌清理與彙總壓縮：%s 估計約 %.2f GB、%s 筆（PostgreSQL 統計估值）。先保留摘要再刪除過期明細，每批最多 1,000 筆，每工作最多 1,000,000 筆；cutoff=%s。", request.Month, partition, float64(estimatedInitialBytes)/1e9, strconv.FormatInt(estimatedInitialRows, 10), cutoff.Format(time.RFC3339))
	var deletedTotal int64
	for batch := 0; batch < forceLogCleanupBatchLimit/forceLogCleanupBatchSize; batch++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		deleted, err := storage.DeleteNodeAccessLogBatch(ctx, month, cutoff, forceLogCleanupBatchSize)
		if err != nil {
			return nil, fmt.Errorf("delete access log batch after %d rows: %w", deletedTotal, err)
		}
		if deleted < 0 || deleted > forceLogCleanupBatchSize {
			return nil, fmt.Errorf("invalid deleted batch count: %d", deleted)
		}
		deletedTotal += deleted
		if batch%10 == 0 || deleted < forceLogCleanupBatchSize {
			taskSvc.AppendLog(ctx, "%s 本批實際刪除 %d 筆，累計刪除 %d 筆。", partition, deleted, deletedTotal)
		}
		if deleted < forceLogCleanupBatchSize {
			break
		}
	}
	hasMore, err := storage.HasNodeAccessLogsBefore(ctx, month, cutoff)
	if err != nil {
		return nil, fmt.Errorf("check remaining access logs: %w", err)
	}
	if hasMore {
		taskSvc.AppendLog(ctx, "指定月份仍有 cutoff 前明細，重新執行此工作可繼續清理與彙總壓縮。")
	} else {
		taskSvc.AppendLog(ctx, "查詢確認指定月份已無 cutoff 前的待刪記錄。")
	}
	taskSvc.AppendLog(ctx, "彙總壓縮是保留摘要後刪除過期明細，不執行 VACUUM 縮檔；PostgreSQL 檔案空間未必立即縮小，VACUUM FULL 需另行於維護窗執行。")
	return &contracts.TaskResultDTO{Message: fmt.Sprintf("%s 本次月份日誌清理與彙總壓縮結束，已保留摘要並實際刪除 %d 筆過期明細，has_more=%t", request.Month, deletedTotal, hasMore), Detail: map[string]any{"partition": partition, "month": request.Month, "cutoff": cutoff, "deleted": deletedTotal, "has_more": hasMore, "initial_estimated_rows": estimatedInitialRows, "initial_estimated_bytes": estimatedInitialBytes}}, nil
}
