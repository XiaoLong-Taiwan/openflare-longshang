// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package openflare registers OpenFlare HTTP routes.
// Management console APIs are mounted via RegisterV1Routes under /api/v1/d.
// Agent/Relay/Tunnel protocol routes are mounted via RegisterRoutes under /api/v1.
package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
)

// RegisterRoutes mounts Agent/Relay/Tunnel protocol routes under the /api/v1 group.
func RegisterRoutes(apiV1Router core.RouterExtension, _ contracts.AuthService) {
	registerAgentRoutes(apiV1Router)
	registerRelayRoutes(apiV1Router)
	registerTunnelRoutes(apiV1Router)
}
