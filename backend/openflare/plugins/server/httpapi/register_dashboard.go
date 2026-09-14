// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/dashboard"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerDashboardRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	dashboardRoute := apiGroup.Group("/dashboard")
	dashboardRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		dashboardRoute.GET("/overview", dashboard.GetOverviewHandler)
	}
}
