// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"Wavelet/core"
	"Wavelet/core/extpoints"
	cf "Wavelet/openflare/plugins/server/domain/cloudflare"
	"Wavelet/openflare/plugins/server/domain/fleet"
	"Wavelet/openflare/plugins/server/domain/observability"
	"Wavelet/openflare/plugins/server/domain/pages"
	"Wavelet/openflare/plugins/server/domain/tls"
	oftask "Wavelet/openflare/plugins/server/kernel/task"
)

func registerOpenFlareTasks(ctx *core.Context) {
	registerOFTask(ctx, fleet.SSLRenewTask, &fleet.SSLRenewHandler{}, fleet.SSLRenewMeta)
	registerOFTask(ctx, fleet.WAFIPGroupSyncTask, &fleet.WAFIPGroupSyncHandler{}, fleet.WAFIPGroupSyncMeta)
	registerOFTask(ctx, fleet.UptimeKumaSyncTask, &fleet.UptimeKumaSyncHandler{}, fleet.UptimeKumaSyncMeta)
	registerOFTask(ctx, fleet.LogDBSwitchTask, &observability.LogDBSwitchHandler{}, fleet.LogDBSwitchMeta)

	registerOFTask(ctx, cf.SyncMemberTask, &cf.SyncMemberTaskHandler{}, cf.SyncMemberMeta)
	registerOFTask(ctx, cf.SyncGroupTask, &cf.SyncGroupTaskHandler{}, cf.SyncGroupMeta)
	registerOFTask(ctx, cf.SyncByNodeTask, &cf.SyncByNodeTaskHandler{}, cf.SyncByNodeMeta)

	registerOFTask(ctx, pages.PagesSourceScanTask, &pages.SourceScanHandler{}, pages.PagesSourceScanMeta)
	registerOFTask(ctx, pages.PagesSourceActionTask, &pages.SourceActionHandler{}, pages.PagesSourceActionMeta)

	registerOFTask(ctx, tls.SSLSingleRenewTask, &tls.SSLSingleRenewHandler{}, tls.SSLSingleRenewMeta)
}

func registerOFTask(ctx *core.Context, pattern string, handler oftask.TaskHandler, meta oftask.TaskMeta) {
	opts := []extpoints.TaskOption{
		extpoints.WithTaskMeta(meta.ToDTO()),
	}
	if meta.MaxRetry > 0 {
		opts = append(opts, extpoints.WithTaskRetry(meta.MaxRetry))
	}
	ctx.Task().Register(pattern, func(c context.Context, payload []byte) error {
		_, err := handler.Execute(c, payload)
		return err
	}, opts...)
}
