// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/site/config_version"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerConfigVersionRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	configVersionGroup := apiGroup.Group("/config-versions")
	configVersionGroup.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(configVersionGroup, "GET", config_version.ListConfigVersionsHandler)
		configVersionGroup.GET("/active", config_version.GetActiveConfigVersionHandler)
		configVersionGroup.GET("/preview", config_version.PreviewConfigVersionHandler)
		configVersionGroup.GET("/diff", config_version.DiffConfigVersionHandler)
		configVersionGroup.GET("/:id", config_version.GetConfigVersionHandler)
		configVersionGroup.POST("/publish", config_version.PublishConfigVersionHandler)
		configVersionGroup.POST("/:id/activate", config_version.ActivateConfigVersionHandler)
		configVersionGroup.POST("/cleanup", config_version.CleanupConfigVersionsHandler)
	}
}
