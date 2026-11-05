// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package admin_test

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/plugins/domain/admin"
	"Wavelet/plugins/domain/admin/service"
	"Wavelet/plugins/domain/upload"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminPluginUnit(t *testing.T) {
	ctx := core.NewContext(context.Background())
	p := admin.New()
	assert.Equal(t, "admin", p.Name())
	assert.Equal(t, "1.0.0", p.Manifest().Version)
	require.NoError(t, p.Apply(ctx))

	// Verify routes
	routes := ctx.Router().Routes()
	assert.NotEmpty(t, routes)
	var hasRobots bool
	for _, r := range routes {
		if r.Path == "/robots.txt" && r.Method == "GET" {
			hasRobots = true
			break
		}
	}
	assert.True(t, hasRobots, "admin plugin must register /robots.txt")

	// Verify tasks
	_, ok := ctx.Tasks().Get("logs:db_switch")
	require.True(t, ok)
	_, ok = ctx.Tasks().Get("system:cleanup")
	require.True(t, ok)
	_, ok = ctx.Tasks().Get("system:force-log-cleanup")
	require.True(t, ok)
	if _, ok := ctx.Tasks().Get("database:maintenance"); !ok {
		t.Error("admin.Apply() did not register database:maintenance for manual execution")
	}
	if schedule, ok := ctx.Schedule().Get("system:cleanup"); !ok || schedule.Spec != "0 3 * * *" {
		t.Errorf("admin.Apply() system:cleanup schedule = %+v, present = %t, want daily at 03:00", schedule, ok)
	}
	if schedule, ok := ctx.Schedule().Get("database:maintenance"); ok {
		t.Errorf("admin.Apply() registered database:maintenance schedule = %+v, want manual execution only", schedule)
	}

	// Verify settings
	setting, ok := ctx.Settings().Get("admin.system_cleanup_cron")
	require.True(t, ok)
	assert.Equal(t, "0 3 * * *", setting.Default)

	provider, err := core.Inject[contracts.PublicConfigProvider](ctx)
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestSystemCleanupRegistrationOrder(t *testing.T) {
	for _, order := range []string{"admin_then_upload", "upload_then_admin"} {
		t.Run(order, func(t *testing.T) {
			ctx := core.NewContext(context.Background())
			plugins := []core.Plugin{admin.New(), upload.New()}
			if order == "upload_then_admin" {
				plugins[0], plugins[1] = plugins[1], plugins[0]
			}
			for _, plugin := range plugins {
				if err := plugin.Apply(ctx); err != nil {
					t.Fatalf("%s.Apply(%s) = %v, want nil", plugin.Name(), order, err)
				}
			}
			cleanup, ok := ctx.Tasks().Get("system:cleanup")
			if !ok {
				t.Fatalf("Tasks.Get(system:cleanup, %s) missing, want admin task", order)
			}
			if _, ok := cleanup.Handler.(*service.SystemCleanupHandler); !ok {
				t.Errorf("system:cleanup handler(%s) = %T, want *service.SystemCleanupHandler", order, cleanup.Handler)
			}
			if _, ok := ctx.Tasks().Get("upload:cleanup-orphans"); ok {
				t.Errorf("Tasks.Get(upload:cleanup-orphans, %s) present, want event-only cleanup", order)
			}
			schedule, ok := ctx.Schedule().Get("system:cleanup")
			if !ok || schedule.Spec != "0 3 * * *" {
				t.Errorf("Schedule.Get(system:cleanup, %s) = %+v, present=%t, want daily at 03:00", order, schedule, ok)
			}
		})
	}
}

func TestAdminMigrationsIncludeTaskExecutionsAndSchedules(t *testing.T) {
	ctx := core.NewContext(context.Background())
	p := admin.New()
	require.NoError(t, p.Apply(ctx))

	entry, ok := ctx.Migrations().Get("admin")
	require.True(t, ok, "admin plugin must register migrations")
	assert.Equal(t, "admin", entry.PluginID)

	// Verify sqlite migration files include w_schedules and w_task_executions
	sqliteDir, err := entry.FS.Open("migrations/sqlite/00001_initial.sql")
	require.NoError(t, err)
	defer sqliteDir.Close()

	stat, err := sqliteDir.Stat()
	require.NoError(t, err)
	buf := make([]byte, stat.Size())
	_, err = sqliteDir.Read(buf)
	require.NoError(t, err)
	content := string(buf)

	assert.Contains(t, content, "CREATE TABLE IF NOT EXISTS w_task_executions")
	assert.Contains(t, content, "CREATE TABLE IF NOT EXISTS w_schedules")
	assert.Contains(t, content, "CREATE TABLE IF NOT EXISTS w_system_configs")
	assert.Contains(t, content, "CREATE TABLE IF NOT EXISTS w_templates")

	// Verify postgres migration files include w_schedules and w_task_executions
	pgDir, err := entry.FS.Open("migrations/postgres/00001_initial.sql")
	require.NoError(t, err)
	defer pgDir.Close()

	stat, err = pgDir.Stat()
	require.NoError(t, err)
	buf = make([]byte, stat.Size())
	_, err = pgDir.Read(buf)
	require.NoError(t, err)
	pgContent := string(buf)

	assert.Contains(t, pgContent, "CREATE TABLE IF NOT EXISTS w_task_executions")
	assert.Contains(t, pgContent, "CREATE TABLE IF NOT EXISTS w_schedules")
	assert.Contains(t, pgContent, "CREATE TABLE IF NOT EXISTS w_system_configs")
	assert.Contains(t, pgContent, "CREATE TABLE IF NOT EXISTS w_templates")
}
