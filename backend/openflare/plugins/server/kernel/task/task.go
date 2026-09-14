// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package task is a thin OpenFlare adapter over contracts.TaskService.
// Remaining OF handlers keep calling task.AppendLog / DispatchTask.
package task

import (
	"context"
	"errors"
	"strings"
	"sync"

	"Wavelet/core/contracts"

	"github.com/hibiken/asynq"
)

const (
	// QueueDefault is the default Asynq queue name.
	QueueDefault = "default"
	// DefaultMaxRetry is the default max retry count for OF tasks.
	DefaultMaxRetry              = 3
	defaultPermanentErrorMessage = "任务无法继续执行"
)

var (
	svcMu sync.RWMutex
	svc   contracts.TaskService
)

// SetService injects the platform TaskService used by DispatchTask/AppendLog.
func SetService(s contracts.TaskService) {
	svcMu.Lock()
	defer svcMu.Unlock()
	svc = s
}

func current() contracts.TaskService {
	svcMu.RLock()
	defer svcMu.RUnlock()
	return svc
}

// TaskParam is the OpenFlare task parameter descriptor.
type TaskParam = contracts.TaskParamDTO

// TaskMeta is the OpenFlare task metadata descriptor.
type TaskMeta struct {
	Type         string
	AsynqTask    string
	Name         string
	Description  string
	SupportsTime bool
	MaxRetry     int
	Queue        string
	Retryable    bool
	InternalOnly bool
	Params       []TaskParam
}

// ToDTO converts TaskMeta to the platform DTO.
func (m TaskMeta) ToDTO() contracts.TaskMetaDTO {
	queue := m.Queue
	if queue == "" {
		queue = QueueDefault
	}
	return contracts.TaskMetaDTO{
		Type:         m.Type,
		AsynqTask:    m.AsynqTask,
		Name:         m.Name,
		DisplayName:  m.Name,
		Description:  m.Description,
		SupportsTime: m.SupportsTime,
		Params:       m.Params,
		MaxRetry:     m.MaxRetry,
		Queue:        queue,
		Retryable:    m.Retryable,
	}
}

// TaskResult is the OpenFlare task execution result.
type TaskResult struct {
	Message string
	Detail  string
}

// PayloadValidator validates and normalizes a task payload.
type PayloadValidator interface {
	ValidatePayload(payload []byte) ([]byte, error)
}

// TaskHandler is the OpenFlare async task handler contract.
type TaskHandler interface {
	Execute(ctx context.Context, payload []byte) (*TaskResult, error)
}

// AppendLog appends a line to the current task execution log.
func AppendLog(ctx context.Context, format string, args ...any) {
	if s := current(); s != nil {
		s.AppendLog(ctx, format, args...)
	}
}

// DispatchTask enqueues a task by admin type.
func DispatchTask(ctx context.Context, taskType string, payload []byte, triggeredBy string) (string, error) {
	s := current()
	if s == nil {
		return "", errors.New("task service not initialized")
	}
	return s.Dispatch(ctx, taskType, payload, triggeredBy)
}

type permanentTaskError struct {
	message string
}

// PermanentError marks a safe domain message as a non-retryable task failure.
func PermanentError(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = defaultPermanentErrorMessage
	}
	return &permanentTaskError{message: message}
}

func (e *permanentTaskError) Error() string {
	return e.message
}

func (e *permanentTaskError) Unwrap() error {
	return asynq.SkipRetry
}
