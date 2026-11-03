// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/core/extpoints"
	"Wavelet/plugins/domain/admin/handler"
	"Wavelet/plugins/domain/admin/service"
	"Wavelet/plugins/drivers/driver_asynq_worker"
	"encoding/json"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listTaskTypesResponse struct {
	ErrorMsg string                  `json:"error_msg"`
	Data     []contracts.TaskMetaDTO `json:"data"`
}

func TestForceCleanupLogStorageHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := core.NewContext(t.Context())
	miniRedis := miniredis.RunT(t)
	driver_asynq_worker.AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: miniRedis.Addr()})
	t.Cleanup(func() {
		_ = driver_asynq_worker.AsynqClient.Close()
		driver_asynq_worker.AsynqClient = nil
	})
	worker := driver_asynq_worker.New()
	require.NoError(t, worker.Apply(ctx))
	ctx.Task().Register(service.ForceLogCleanupTask, func(_ any) error { return nil },
		extpoints.WithTaskType("force_log_cleanup"),
		extpoints.WithTaskName("強制清理舊日誌"),
		extpoints.WithTaskRetry(3),
		extpoints.WithTaskQueue("default"),
		extpoints.WithTaskRetryable(true),
	)
	taskSvc, err := core.Inject[contracts.TaskService](ctx)
	require.NoError(t, err)
	service.SetTaskService(taskSvc)
	t.Cleanup(service.ResetServices)

	r := gin.New()
	r.POST("/api/v1/admin/logs/storage/force-cleanup", handler.ForceCleanupLogStorage)

	for _, test := range []struct {
		name string
		body string
		want int
	}{
		{name: "requires confirmation", body: `{"confirmed":false}`, want: http.StatusBadRequest},
		{name: "queues confirmed cleanup", body: `{"confirmed":true}`, want: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logs/storage/force-cleanup", strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, test.want, w.Code)
			if test.want == http.StatusOK {
				inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: miniRedis.Addr()})
				defer inspector.Close()
				tasks, err := inspector.ListPendingTasks("default")
				require.NoError(t, err)
				assert.Len(t, tasks, 1)
				assert.Equal(t, service.ForceLogCleanupTask, tasks[0].Type)
			}
		})
	}
}

func TestListTaskTypesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := core.NewContext(t.Context())
	worker := driver_asynq_worker.New()
	require.NoError(t, worker.Apply(ctx))

	ctx.Task().Register("logs:db_switch", func(_ any) error { return nil },
		extpoints.WithTaskType("logs_db_switch"),
		extpoints.WithTaskName("切换日志数据库"),
		extpoints.WithTaskDescription("复制迁移用户访问日志并在成功后切换日志主库"),
		extpoints.WithTaskCategory("system"),
		extpoints.WithTaskRetry(3),
		extpoints.WithTaskQueue("default"),
		extpoints.WithTaskRetryable(true),
		extpoints.WithTaskParams(contracts.TaskParamDTO{
			Name:        "target",
			Label:       "目标日志库",
			Type:        "string",
			Required:    true,
			Placeholder: "postgres|sqlite|clickhouse",
			Description: "迁移目标",
		}),
	)

	taskSvc, err := core.Inject[contracts.TaskService](ctx)
	require.NoError(t, err)
	service.SetTaskService(taskSvc)

	r := gin.New()
	r.GET("/api/v1/admin/tasks/types", handler.ListTaskTypes)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tasks/types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp listTaskTypesResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.ErrorMsg)
	require.NotEmpty(t, resp.Data)

	var found bool
	for _, task := range resp.Data {
		if task.Type == "logs_db_switch" {
			found = true
			assert.Equal(t, "logs:db_switch", task.AsynqTask)
			assert.Equal(t, "切换日志数据库", task.Name)
			assert.Equal(t, "复制迁移用户访问日志并在成功后切换日志主库", task.Description)
			assert.Equal(t, "system", task.Category)
			assert.Equal(t, 3, task.MaxRetry)
			assert.Equal(t, "default", task.Queue)
			assert.True(t, task.Retryable)
			require.Len(t, task.Params, 1)
			assert.Equal(t, "target", task.Params[0].Name)
			assert.Equal(t, "目标日志库", task.Params[0].Label)
			assert.Equal(t, "string", task.Params[0].Type)
			assert.True(t, task.Params[0].Required)
			assert.Equal(t, "postgres|sqlite|clickhouse", task.Params[0].Placeholder)
		}
	}
	assert.True(t, found, "expected logs_db_switch in task types")

	// Ensure NO task in the list has an empty Type or Name, which breaks SelectItem key/value in frontend
	for _, task := range resp.Data {
		assert.NotEmpty(t, task.Type, "task.type must never be empty")
		assert.NotEmpty(t, task.Name, "task.name must never be empty")
	}
}
