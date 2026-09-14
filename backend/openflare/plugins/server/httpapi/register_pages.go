// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/pages"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerPagesRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	pagesRoute := apiGroup.Group("/pages")
	pagesRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(pagesRoute, "GET", pages.ListProjectsHandler)
		pagesRoute.GET("/:id", pages.GetProjectHandler)
		apiutil.RegisterCollection(pagesRoute, "POST", pages.CreateProjectHandler)
		pagesRoute.POST("/:id/update", pages.UpdateProjectHandler)
		pagesRoute.POST("/:id/delete", pages.DeleteProjectHandler)
		pagesRoute.GET("/:id/source", pages.GetSourceHandler)
		pagesRoute.POST("/:id/source/update", pages.UpdateSourceHandler)
		pagesRoute.POST("/:id/source/delete", pages.DeleteSourceHandler)
		pagesRoute.POST("/:id/source/check", pages.CheckSourceHandler)
		pagesRoute.POST("/:id/source/sync", pages.SyncSourceHandler)
		pagesRoute.GET("/:id/deployments", pages.ListDeploymentsHandler)
		pagesRoute.POST("/:id/deployments/upload", pages.UploadDeploymentHandler)
		pagesRoute.POST("/:id/deployments/upload-from-url", pages.UploadDeploymentFromURLHandler)
		pagesRoute.POST("/:id/deployments/:deployment_id/activate", pages.ActivateDeploymentHandler)
		pagesRoute.POST("/:id/deployments/:deployment_id/delete", pages.DeleteDeploymentHandler)
		pagesRoute.GET("/deployments/:deployment_id/files", pages.ListDeploymentFilesHandler)
	}
}
