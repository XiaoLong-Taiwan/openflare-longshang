// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
)

// V1BasePath is the OpenFlare console API prefix under /api/v1.
const V1BasePath = "/api/v1/d"

// RegisterV1Routes mounts OpenFlare management console APIs under /api/v1/d
// and OpenFlare-owned admin updater routes under /api/v1/admin/update*.
func RegisterV1Routes(apiV1Router core.RouterExtension, auth contracts.AuthService) {
	group := apiV1Router.Group("/d")
	registerOptionRoutes(group, auth)
	registerOriginRoutes(group, auth)
	registerApplyLogRoutes(group, auth)
	registerProxyRouteRoutes(group, auth)
	registerNodeRoutes(group, auth)
	registerWAFRoutes(group, auth)
	registerTLSRoutes(group, auth)
	registerCloudflareRoutes(group, auth)
	registerZoneRoutes(group, auth)
	registerConfigVersionRoutes(group, auth)
	registerPagesRoutes(group, auth)
	registerDashboardRoutes(group, auth)
	registerObservabilityRoutes(group, auth)
	registerUpdaterRoutes(apiV1Router, auth)
}
