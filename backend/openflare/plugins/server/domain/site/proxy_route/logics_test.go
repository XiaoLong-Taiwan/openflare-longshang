// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package proxy_route

import (
	"context"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProxyRouteTestDB(t *testing.T) func() {
	t.Helper()
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, sqliteDB.AutoMigrate(
		&model.ProxyRoute{},
		&model.Origin{},
		&model.Zone{},
		&model.ZoneDomain{},
		&model.TLSCertificate{},
		&model.PagesProject{},
	))
	repository.SetDBForTest(sqliteDB)
	return func() { repository.SetDBForTest(nil) }
}

func createZoneDomain(t *testing.T, ctx context.Context, domain string, certID *uint) *model.ZoneDomain {
	t.Helper()
	zone := &model.Zone{Domain: "example.com"}
	var existing model.Zone
	if err := repository.DB(ctx).Where("domain = ?", zone.Domain).First(&existing).Error; err == nil {
		zone = &existing
	} else {
		require.NoError(t, repository.DB(ctx).Create(zone).Error)
	}
	item := &model.ZoneDomain{ZoneID: zone.ID, Domain: domain, CertID: certID}
	require.NoError(t, repository.DB(ctx).Create(item).Error)
	return item
}

func TestCreateProxyRouteBindsZoneDomains(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domainA := createZoneDomain(t, ctx, "api.example.com", nil)
	domainB := createZoneDomain(t, ctx, "www.example.com", nil)

	view, err := CreateProxyRoute(ctx, Input{SiteName: "api", ZoneDomainIDs: []uint{domainA.ID, domainB.ID}, OriginURL: "http://origin.example.com:8080", Enabled: true})
	require.NoError(t, err)
	assert.Equal(t, []uint{domainA.ID, domainB.ID}, view.ZoneDomainIDs)
	require.Len(t, view.ZoneDomains, 2)
	assert.Equal(t, "api.example.com", view.ZoneDomains[0].Domain)
}

func TestCreateProxyRouteStoresStructuredUpstreamTargets(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "api.example.com", nil)

	view, err := CreateProxyRoute(ctx, Input{
		SiteName:       "api",
		ZoneDomainIDs:  []uint{domain.ID},
		UpstreamTargets: []UpstreamTargetInput{
			{URL: "http://primary.example.com:8080", Priority: 0},
			{URL: "http://backup.example.com:8080", Priority: 2},
		},
		LoadBalancing: "least_conn",
		Enabled:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, []UpstreamTargetInput{
		{URL: "http://primary.example.com:8080", Priority: 0},
		{URL: "http://backup.example.com:8080", Priority: 2},
	}, view.UpstreamTargets)
	assert.Equal(t, "least_conn", view.LoadBalancing)
	assert.Equal(t, []string{
		"http://primary.example.com:8080",
		"http://backup.example.com:8080",
	}, view.UpstreamList)
}

func TestCreateProxyRouteOrdersStructuredUpstreamTargetsByPriority(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "api.example.com", nil)

	view, err := CreateProxyRoute(ctx, Input{
		SiteName:      "api",
		ZoneDomainIDs: []uint{domain.ID},
		UpstreamTargets: []UpstreamTargetInput{
			{URL: "http://backup.example.com:8080", Priority: 2},
			{URL: "http://primary.example.com:8080", Priority: 0},
		},
		Enabled: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []UpstreamTargetInput{
		{URL: "http://primary.example.com:8080", Priority: 0},
		{URL: "http://backup.example.com:8080", Priority: 2},
	}, view.UpstreamTargets)
	assert.Equal(t, "http://primary.example.com:8080", view.OriginURL)
}

func TestCreateProxyRouteRejectsInvalidStructuredUpstreamTarget(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "api.example.com", nil)

	_, err := CreateProxyRoute(ctx, Input{
		SiteName:      "api",
		ZoneDomainIDs: []uint{domain.ID},
		UpstreamTargets: []UpstreamTargetInput{
			{URL: "http://primary.example.com:8080", Priority: -1},
		},
	})
	require.EqualError(t, err, errProxyRouteUpstreamPriority)

	_, err = CreateProxyRoute(ctx, Input{
		SiteName:      "priority-tiers",
		ZoneDomainIDs: []uint{domain.ID},
		UpstreamTargets: []UpstreamTargetInput{
			{URL: "http://primary.example.com:8080", Priority: 0},
			{URL: "http://backup-one.example.com:8080", Priority: 1},
			{URL: "http://backup-two.example.com:8080", Priority: 2},
		},
	})
	require.EqualError(t, err, errProxyRouteUpstreamPriorityTier)

	_, err = CreateProxyRoute(ctx, Input{
		SiteName:      "other",
		ZoneDomainIDs: []uint{domain.ID},
		OriginURL:     "http://origin.example.com:8080",
		LoadBalancing: "invalid",
	})
	require.EqualError(t, err, errProxyRouteLoadBalancing)
}

func TestCreateProxyRouteRejectsInvalidZoneDomainBindings(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "api.example.com", nil)
	base := Input{SiteName: "api", OriginURL: "http://origin.example.com:8080"}

	_, err := CreateProxyRoute(ctx, base)
	require.EqualError(t, err, errProxyRouteZoneDomainsRequired)
	base.ZoneDomainIDs = []uint{domain.ID, domain.ID}
	_, err = CreateProxyRoute(ctx, base)
	require.EqualError(t, err, errProxyRouteZoneDomainDuplicate)

	first, err := CreateProxyRoute(ctx, Input{SiteName: "first", ZoneDomainIDs: []uint{domain.ID}, OriginURL: "http://origin.example.com:8080"})
	require.NoError(t, err)
	_, err = CreateProxyRoute(ctx, Input{SiteName: "second", ZoneDomainIDs: []uint{domain.ID}, OriginURL: "http://other.example.com:8080"})
	require.Error(t, err)
	require.NoError(t, DeleteProxyRoute(ctx, first.ID))
}

func TestCreateProxyRouteHTTPSRequiresCoveringCertificate(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "api.example.com", nil)
	_, err := CreateProxyRoute(ctx, Input{SiteName: "api", ZoneDomainIDs: []uint{domain.ID}, OriginURL: "http://origin.example.com:8080", EnableHTTPS: true})
	require.EqualError(t, err, errProxyRouteCertRequired)
}

func TestPagesRouteLocksAndRevalidatesTargetProject(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "pages.example.com", nil)
	activeDeploymentID := uint(99)
	project := &model.PagesProject{
		Name:               "Pages Site",
		Slug:               "pages-site",
		Enabled:            true,
		ActiveDeploymentID: &activeDeploymentID,
	}
	require.NoError(t, repository.DB(ctx).Create(project).Error)

	view, err := CreateProxyRoute(ctx, Input{
		SiteName:       "pages",
		ZoneDomainIDs:  []uint{domain.ID},
		UpstreamType:   proxyRouteUpstreamTypePages,
		PagesProjectID: &project.ID,
		Enabled:        true,
	})
	require.NoError(t, err)
	require.NotNil(t, view.PagesProjectID)
	assert.Equal(t, project.ID, *view.PagesProjectID)

	require.NoError(t, repository.DB(ctx).Delete(&model.PagesProject{}, project.ID).Error)
	route := &model.ProxyRoute{UpstreamType: proxyRouteUpstreamTypePages, PagesProjectID: &project.ID}
	err = repository.WithProxyRouteTx(ctx, func(tx *gorm.DB) error {
		return lockPagesProjectsForRouteMutation(tx, 0, route)
	})
	require.EqualError(t, err, errProxyRoutePagesNotFound)
}

func TestRouteCanMoveAwayFromAlreadyMissingPagesProject(t *testing.T) {
	cleanup := setupProxyRouteTestDB(t)
	defer cleanup()
	ctx := context.Background()
	missingProjectID := uint(404)
	route := &model.ProxyRoute{UpstreamType: "direct"}

	err := repository.WithProxyRouteTx(ctx, func(tx *gorm.DB) error {
		return lockPagesProjectsForRouteMutation(tx, missingProjectID, route)
	})
	require.NoError(t, err)
}

func TestNormalizeCachePolicyDefaultsAndLegacy(t *testing.T) {
	assert.Empty(t, normalizeCachePolicy(false, "static"))
	// Empty/url on write = legacy all (compat); UI sends static explicitly for new default.
	assert.Equal(t, proxyRouteCachePolicyAll, normalizeCachePolicy(true, ""))
	assert.Equal(t, proxyRouteCachePolicyStatic, normalizeCachePolicy(true, "static"))
	assert.Equal(t, proxyRouteCachePolicyAll, normalizeCachePolicy(true, "url"))
	assert.Equal(t, proxyRouteCachePolicyAll, normalizeCachePolicy(true, "all"))
	assert.Equal(t, proxyRouteCachePolicySuffix, normalizeCachePolicy(true, "suffix"))

	assert.Empty(t, displayCachePolicy(false, "all"))
	assert.Equal(t, proxyRouteCachePolicyAll, displayCachePolicy(true, ""))
	assert.Equal(t, proxyRouteCachePolicyAll, displayCachePolicy(true, "url"))
	assert.Equal(t, proxyRouteCachePolicyStatic, displayCachePolicy(true, "static"))

	rules, err := normalizeCacheRules(true, "url", []string{"css"})
	require.NoError(t, err)
	assert.Empty(t, rules)

	rules, err = normalizeCacheRules(true, "static", nil)
	require.NoError(t, err)
	assert.Empty(t, rules)

	_, err = normalizeCacheRules(true, "suffix", nil)
	require.Error(t, err)
}
