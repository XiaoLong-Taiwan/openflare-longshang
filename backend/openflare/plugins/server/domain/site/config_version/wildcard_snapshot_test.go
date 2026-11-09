// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package config_version

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/domain/site/zone"
	oftls "Wavelet/openflare/plugins/server/domain/tls"
	"Wavelet/openflare/plugins/server/kernel/runtimeconfig"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
	openrestyrender "Wavelet/openflare/share/render/openresty"
)

func TestWildcardDomainConfigBundle(t *testing.T) {
	t.Cleanup(setupConfigVersionTestDB(t))
	ctx := context.Background()
	if err := repository.DB(ctx).AutoMigrate(&model.TLSCertificate{}); err != nil {
		t.Fatal(err)
	}
	oldSecret := runtimeconfig.SessionSecret()
	runtimeconfig.SetSessionSecret("wildcard-snapshot-test-secret")
	t.Cleanup(func() { runtimeconfig.SetSessionSecret(oldSecret) })
	certPEM, keyPEM := generateTestCertKeyPairForSnapshotForDomain(t, "*.s3.example.com")
	cert, err := oftls.CreateCertificate(ctx, oftls.CertificateInput{Name: "wildcard", CertPEM: certPEM, KeyPEM: keyPEM})
	if err != nil {
		t.Fatal(err)
	}
	root, err := zone.Create(ctx, zone.Input{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	domain, err := zone.CreateDomain(ctx, root.ID, zone.DomainInput{Domain: "*.s3.example.com", CertID: &cert.ID})
	if err != nil {
		t.Fatalf("CreateDomain(wildcard) error = %v", err)
	}
	route := &model.ProxyRoute{SiteName: "wildcard-site", OriginURL: "http://origin:8080", Upstreams: `["http://origin:8080"]`, Enabled: true, EnableHTTPS: true, RedirectHTTP: true}
	if err := repository.CreateProxyRouteRecord(ctx, route); err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceZoneDomainRouteBindings(ctx, route.ID, []uint{domain.ID}); err != nil {
		t.Fatal(err)
	}
	bundle, err := buildCurrentConfigBundle(ctx, true)
	if err != nil {
		t.Fatalf("buildCurrentConfigBundle(wildcard) error = %v", err)
	}
	var doc openrestyrender.Document
	if err := json.Unmarshal([]byte(bundle.SnapshotJSON), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Routes) != 1 || !slices.Equal(doc.Routes[0].Domains, []string{domain.Domain}) || !slices.Equal(doc.Routes[0].DomainCertIDs, []uint{cert.ID}) {
		t.Fatalf("snapshot routes = %+v, want wildcard domain and its certificate", doc.Routes)
	}
	if !strings.Contains(bundle.RouteConfig, "server_name *.s3.example.com;") || !strings.Contains(bundle.RouteConfig, "listen 443 ssl;") {
		t.Errorf("wildcard bundle missing HTTPS server: %s", bundle.RouteConfig)
	}
}
