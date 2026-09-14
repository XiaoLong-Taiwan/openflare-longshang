// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/fleet/node"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerNodeRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	nodeRoute := apiGroup.Group("/nodes")
	nodeRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		nodeRoute.GET("/bootstrap-token", node.GetBootstrapTokenHandler)
		nodeRoute.POST("/bootstrap-token/rotate", node.RotateBootstrapTokenHandler)
		apiutil.RegisterCollection(nodeRoute, "GET", node.ListNodesHandler)
		apiutil.RegisterCollection(nodeRoute, "POST", node.CreateNodeHandler)
		nodeRoute.GET("/:id/agent-release", node.GetAgentReleaseHandler)
		nodeRoute.POST("/:id/update", node.UpdateNodeHandler)
		nodeRoute.POST("/:id/delete", node.DeleteNodeHandler)
		nodeRoute.POST("/:id/agent-update", node.RequestAgentUpdateHandler)
		nodeRoute.POST("/:id/openresty-restart", node.RequestOpenrestyRestartHandler)
		nodeRoute.POST("/:id/force-sync", node.RequestForceSyncHandler)
		nodeRoute.GET("/:id/observability", node.GetObservabilityHandler)
		nodeRoute.POST("/:id/observability/cleanup", node.CleanupHealthEventsHandler)
	}
}
