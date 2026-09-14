// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/observability"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerObservabilityRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	accessLogRoute := apiGroup.Group("/access-logs")
	accessLogRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(accessLogRoute, "GET", observability.GetAccessLogsHandler)
		accessLogRoute.GET("/overview", observability.GetAccessLogOverviewHandler)
		accessLogRoute.GET("/folds", observability.GetFoldedAccessLogsHandler)
		accessLogRoute.GET("/folds/ip-summary", observability.GetFoldedAccessLogIPsHandler)
		accessLogRoute.GET("/ip-summary", observability.GetAccessLogIPSummariesHandler)
		accessLogRoute.GET("/ip-summary/trend", observability.GetAccessLogIPTrendHandler)
		accessLogRoute.GET("/ip-summary/analysis", observability.GetAccessLogIPAnalysisHandler)
		accessLogRoute.POST("/cleanup", observability.CleanupAccessLogsHandler)
	}
}
