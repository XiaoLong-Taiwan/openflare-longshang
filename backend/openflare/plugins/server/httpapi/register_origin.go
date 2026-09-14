// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/site/origin"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerOriginRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	originRoute := apiGroup.Group("/origins")
	originRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(originRoute, "GET", origin.GetOrigins)
		originRoute.GET("/:id", origin.GetOrigin)
		apiutil.RegisterCollection(originRoute, "POST", origin.CreateOriginHandler)
		originRoute.POST("/:id/update", origin.UpdateOriginHandler)
		originRoute.POST("/:id/delete", origin.DeleteOriginHandler)
	}
}
