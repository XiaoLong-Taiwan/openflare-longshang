// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/site/proxy_route"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerProxyRouteRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	proxyRouteGroup := apiGroup.Group("/proxy-routes")
	proxyRouteGroup.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(proxyRouteGroup, "GET", proxy_route.GetProxyRoutes)
		proxyRouteGroup.GET("/:id", proxy_route.GetProxyRouteHandler)
		apiutil.RegisterCollection(proxyRouteGroup, "POST", proxy_route.CreateProxyRouteHandler)
		proxyRouteGroup.POST("/:id/update", proxy_route.UpdateProxyRouteHandler)
		proxyRouteGroup.POST("/:id/delete", proxy_route.DeleteProxyRouteHandler)
	}
}
