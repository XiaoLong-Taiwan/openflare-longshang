// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package httpapi

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server/domain/tls"
	"Wavelet/openflare/plugins/server/kernel/apiutil"
)

func registerTLSRoutes(apiGroup core.RouterExtension, auth contracts.AuthService) {
	tlsCertificateRoute := apiGroup.Group("/tls-certificates")
	tlsCertificateRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(tlsCertificateRoute, "GET", tls.GetCertificates)
		tlsCertificateRoute.GET("/:id", tls.GetCertificateDetail)
		tlsCertificateRoute.GET("/:id/content", tls.GetCertificateContentHandler)
		apiutil.RegisterCollection(tlsCertificateRoute, "POST", tls.CreateCertificateHandler)
		tlsCertificateRoute.POST("/:id/update", tls.UpdateCertificateHandler)
		tlsCertificateRoute.POST("/:id/update-acme", tls.UpdateACMECertificateHandler)
		tlsCertificateRoute.POST("/:id/convert-acme", tls.ConvertCertificateToACMEHandler)
		tlsCertificateRoute.POST("/import-file", tls.ImportCertificateFile)
		tlsCertificateRoute.POST("/:id/delete", tls.DeleteCertificateHandler)
		tlsCertificateRoute.POST("/apply", tls.ApplyCertificateHandler)
		tlsCertificateRoute.POST("/:id/renew", tls.RenewCertificateHandler)
	}

	acmeAccountRoute := apiGroup.Group("/acme-accounts")
	acmeAccountRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		acmeAccountRoute.GET("/default", tls.GetDefaultAcmeAccountHandler)
	}

	dnsAccountRoute := apiGroup.Group("/dns-accounts")
	dnsAccountRoute.Use(apiutil.AdminMiddlewares(auth)...)
	{
		apiutil.RegisterCollection(dnsAccountRoute, "GET", tls.GetDNSAccounts)
		apiutil.RegisterCollection(dnsAccountRoute, "POST", tls.CreateDNSAccountHandler)
		dnsAccountRoute.POST("/:id/update", tls.UpdateDNSAccountHandler)
		dnsAccountRoute.POST("/:id/delete", tls.DeleteDNSAccountHandler)
	}
}
