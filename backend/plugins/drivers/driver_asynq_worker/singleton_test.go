package driver_asynq_worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSingletonDispatchAcrossClientsAndWorkerLifecycle(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()
	const taskType = "test:singleton_health"
	RegisterTaskMeta(TaskMeta{Type: taskType, AsynqTask: taskType, Name: "health", Singleton: true, Queue: "default", Retryable: true, MaxRetry: 1})
	ctx := context.Background()
	started := make(chan struct{})
	release := make(chan struct{})
	RegisterHandler(taskType, &mockHandler{executeFunc: func(_ context.Context, _ []byte) (*TaskResult, error) {
		close(started)
		<-release
		return &TaskResult{Message: "done"}, nil
	}})
	id, err := DispatchTask(ctx, taskType, []byte("{}"), "controller-a")
	require.NoError(t, err)
	_, err = DispatchTask(context.Background(), taskType, []byte("null"), "controller-b")
	require.ErrorIs(t, err, asynq.ErrDuplicateTask)
	_, err = DispatchTask(ctx, taskType, []byte(`{"unexpected":true}`), "invalid")
	require.Error(t, err)
	var count int64
	require.NoError(t, getDB(ctx).Model(&TaskExecution{}).Where("task_type = ?", taskType).Count(&count).Error)
	require.EqualValues(t, 1, count)
	addr := getRedisClient().(*redis.Client).Options().Addr
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: addr})
	defer inspector.Close()
	info, err := inspector.GetTaskInfo("default", id)
	require.NoError(t, err)
	require.Empty(t, info.Payload)
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
	defer client.Close()
	_, err = client.Enqueue(asynq.NewTask(taskType, nil), asynq.Unique(24*time.Hour))
	require.ErrorIs(t, err, asynq.ErrDuplicateTask)
	_, err = client.Enqueue(asynq.NewTask(taskType, nil), asynq.TaskID(id), asynq.Retention(time.Hour))
	require.ErrorIs(t, err, asynq.ErrTaskIDConflict)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: addr}, asynq.Config{
		Concurrency: 1, Queues: map[string]int{"default": 1}, TaskCheckInterval: 10 * time.Millisecond,
		ShutdownTimeout: time.Second,
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		return ProcessTask(ctx, task)
	})
	require.NoError(t, server.Start(mux))
	defer server.Shutdown()
	defer close(release)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not start")
	}
	_, err = client.Enqueue(asynq.NewTask(taskType, nil), asynq.Unique(24*time.Hour))
	require.ErrorIs(t, err, asynq.ErrDuplicateTask)
	release <- struct{}{}
	require.Eventually(t, func() bool {
		_, err := inspector.GetTaskInfo("default", id)
		return errors.Is(err, asynq.ErrTaskNotFound)
	}, 5*time.Second, 10*time.Millisecond)
	server.Shutdown()
	firstExecution, err := getTaskExecutionByTaskID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, TaskExecutionStatusSucceeded, firstExecution.Status)
	secondID, err := DispatchTask(context.Background(), taskType, nil, "next-cycle")
	require.NoError(t, err)
	require.Equal(t, id, secondID)
	secondExecution, err := getTaskExecutionByTaskID(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, TaskExecutionStatusPending, secondExecution.Status)
}

func TestSingletonPendingDeletionAndManualRetry(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()
	const taskType = "test:singleton_retry"
	RegisterTaskMeta(TaskMeta{Type: taskType, AsynqTask: taskType, Name: "retry", Singleton: true, Queue: "default", Retryable: true, MaxRetry: 0})
	ctx := context.Background()
	id, err := DispatchTask(ctx, taskType, nil, "first")
	require.NoError(t, err)
	addr := getRedisClient().(*redis.Client).Options().Addr
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: addr})
	defer inspector.Close()
	RegisterHandler(taskType, &mockHandler{executeFunc: func(context.Context, []byte) (*TaskResult, error) {
		return nil, asynq.SkipRetry
	}})
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: addr}, asynq.Config{
		Concurrency: 1, Queues: map[string]int{"default": 1}, TaskCheckInterval: 10 * time.Millisecond,
		RetryDelayFunc: func(int, error, *asynq.Task) time.Duration { return 10 * time.Millisecond },
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error { return ProcessTask(ctx, task) })
	require.NoError(t, server.Start(mux))
	defer server.Shutdown()
	require.Eventually(t, func() bool {
		info, getErr := inspector.GetTaskInfo("default", id)
		return getErr == nil && info.State == asynq.TaskStateArchived
	}, 5*time.Second, 10*time.Millisecond)
	execution, err := getTaskExecutionByTaskID(ctx, id)
	require.NoError(t, err)
	RegisterHandler(taskType, &mockHandler{executeFunc: func(context.Context, []byte) (*TaskResult, error) {
		return &TaskResult{Message: "retry success"}, nil
	}})
	retryID, err := RetryTask(ctx, execution.ID)
	require.NoError(t, err)
	_, err = DispatchTask(ctx, taskType, nil, "second")
	require.ErrorIs(t, err, asynq.ErrDuplicateTask)
	require.Equal(t, id, retryID)
	require.Eventually(t, func() bool {
		current, getErr := getTaskExecutionByTaskID(ctx, id)
		return getErr == nil && current.Status == TaskExecutionStatusSucceeded
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSingletonLongPendingIsRejectedByFixedTaskID(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()
	const taskType = "test:singleton_long_pending"
	RegisterTaskMeta(TaskMeta{Type: taskType, AsynqTask: taskType, Name: "long pending", Singleton: true, Queue: "default", Retryable: true, MaxRetry: 1})
	ctx := context.Background()
	id, err := DispatchTask(ctx, taskType, nil, "first")
	require.NoError(t, err)
	require.NotNil(t, testMiniRedis)
	testMiniRedis.FastForward(24*time.Hour + time.Second)
	inspector := GetAsynqInspector()
	client := GetAsynqClient()
	_, err = client.Enqueue(asynq.NewTask(taskType, nil), asynq.TaskID(id), asynq.Unique(24*time.Hour))
	require.ErrorIs(t, err, asynq.ErrTaskIDConflict)
	_, err = DispatchTask(ctx, taskType, nil, "again")
	require.ErrorIs(t, err, asynq.ErrDuplicateTask)
	info, err := inspector.GetTaskInfo("default", id)
	require.NoError(t, err)
	require.Equal(t, asynq.TaskStatePending, info.State)
	var count int64
	require.NoError(t, getDB(ctx).Model(&TaskExecution{}).Where("task_id = ? AND status = ?", id, TaskExecutionStatusPending).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
