// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package driver_asynq_worker

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// TaskExecutionStatus 任务执行状态
type TaskExecutionStatus string

// Task execution status constants.
const (
	TaskExecutionStatusPending   TaskExecutionStatus = "pending"
	TaskExecutionStatusRunning   TaskExecutionStatus = "running"
	TaskExecutionStatusSucceeded TaskExecutionStatus = "succeeded"
	TaskExecutionStatusFailed    TaskExecutionStatus = "failed"
)

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID           uint64              `json:"id,string" gorm:"primaryKey"`
	TaskID       string              `json:"task_id" gorm:"size:128;uniqueIndex;not null"`
	TaskType     string              `json:"task_type" gorm:"size:64;index;not null"`
	TaskName     string              `json:"task_name" gorm:"size:128"`
	Status       TaskExecutionStatus `json:"status" gorm:"size:32;index;not null"`
	Retryable    bool                `json:"retryable" gorm:"not null;default:false"`
	MaxRetry     int                 `json:"max_retry" gorm:"not null;default:0"`
	RetryCount   int                 `json:"retry_count" gorm:"not null;default:0"`
	Log          string              `json:"log" gorm:"type:text"`
	ErrorMessage string              `json:"error_message" gorm:"type:text"`
	Result       string              `json:"result" gorm:"type:text"`
	StartedAt    *time.Time          `json:"started_at" gorm:"index"`
	FinishedAt   *time.Time          `json:"finished_at"`
	Duration     int64               `json:"duration" gorm:"comment:耗时毫秒"`
	Payload      string              `json:"payload" gorm:"type:text"`
	TriggeredBy  string              `json:"triggered_by" gorm:"size:32;not null;default:system"`
	CreatedAt    time.Time           `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (TaskExecution) TableName() string {
	return "w_task_executions"
}

// CreateTaskExecution 创建任务执行记录
func CreateTaskExecution(ctx context.Context, exec *TaskExecution) error {
	return getDB(ctx).Create(exec).Error
}

// GetTaskExecutionByTaskID 根据 TaskID 查询执行记录
func GetTaskExecutionByTaskID(ctx context.Context, taskID string) (*TaskExecution, error) {
	var exec TaskExecution
	if err := getDB(ctx).Where("task_id = ?", taskID).First(&exec).Error; err != nil {
		return nil, err
	}
	_ = loadTaskExecutionLog(ctx, &exec)
	return &exec, nil
}

// GetTaskExecutionByID 根据主键 ID 查询执行记录
func GetTaskExecutionByID(ctx context.Context, id uint64) (*TaskExecution, error) {
	var exec TaskExecution
	if err := getDB(ctx).Where("id = ?", id).First(&exec).Error; err != nil {
		return nil, err
	}
	_ = loadTaskExecutionLog(ctx, &exec)
	return &exec, nil
}

// GetLatestTaskExecutionByTaskType 获取指定任务类型的最新执行记录
func GetLatestTaskExecutionByTaskType(ctx context.Context, taskType string) (*TaskExecution, bool, error) {
	var exec TaskExecution
	err := getDB(ctx).Where("task_type = ?", taskType).Order("id DESC").First(&exec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &exec, true, nil
}

// TaskExecutionCleanupStats describes task execution log cleanup results.
type TaskExecutionCleanupStats struct {
	HighFrequencyDeleted int64
	LowFrequencyDeleted  int64
}

// CleanupTaskExecutionLogs removes finished task execution logs according to frequency-based retention.
func CleanupTaskExecutionLogs(ctx context.Context, now time.Time) (TaskExecutionCleanupStats, error) {
	highFrequencyCutoff := now.AddDate(0, 0, -3)
	lowFrequencyCutoff := now.AddDate(0, 0, -30)
	var highFrequencyDeleted, lowFrequencyDeleted int64
	if err := deleteExpiredTaskExecutions(ctx, TaskExecutionStatusSucceeded, highFrequencyCutoff, &highFrequencyDeleted); err != nil {
		return TaskExecutionCleanupStats{}, err
	}
	if err := deleteExpiredTaskExecutions(ctx, TaskExecutionStatusFailed, lowFrequencyCutoff, &lowFrequencyDeleted); err != nil {
		return TaskExecutionCleanupStats{}, err
	}

	return TaskExecutionCleanupStats{
		HighFrequencyDeleted: highFrequencyDeleted,
		LowFrequencyDeleted:  lowFrequencyDeleted,
	}, nil
}

func deleteExpiredTaskExecutions(ctx context.Context, status TaskExecutionStatus, cutoff time.Time, total *int64) error {
	for {
		ids := make([]uint64, 0, 1000)
		if err := getDB(ctx).Model(&TaskExecution{}).Where("status = ? AND created_at < ?", status, cutoff).Order("id ASC").Limit(1000).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		result := getDB(ctx).Where("id IN ? AND status = ?", ids, status).Delete(&TaskExecution{})
		if result.Error != nil {
			return result.Error
		}
		*total += result.RowsAffected
		if len(ids) < 1000 {
			return nil
		}
	}
}
