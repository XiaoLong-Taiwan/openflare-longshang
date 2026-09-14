// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/site/zone"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerZoneRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	zoneGroup := apiGroup.Group("/zones")
	zoneGroup.Use(apiutil.AdminMiddlewares(auth)...)
	apiutil.RegisterCollection(zoneGroup, "GET", zone.ListHandler)
	apiutil.RegisterCollection(zoneGroup, "POST", zone.CreateHandler)
	zoneGroup.GET("/:id/overview", zone.GetOverviewHandler)
	zoneGroup.GET("/:id/stats", zone.GetStatsHandler)
	zoneGroup.POST("/:id/update", zone.UpdateHandler)
	zoneGroup.POST("/:id/delete", zone.DeleteHandler)
	zoneGroup.POST("/:id/domains", zone.CreateDomainHandler)
	zoneGroup.POST("/:id/domains/:domainId/update", zone.UpdateDomainHandler)
	zoneGroup.POST("/:id/domains/:domainId/delete", zone.DeleteDomainHandler)
}
