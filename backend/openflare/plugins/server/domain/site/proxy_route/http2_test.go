package proxy_route

import (
	"context"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestHTTP2CreateUpdateAndLegacyOmission(t *testing.T) {
	t.Cleanup(setupProxyRouteTestDB(t))
	ctx := context.Background()
	domain := createZoneDomain(t, ctx, "http2.example.com", nil)
	input := Input{SiteName: "http2", ZoneDomainIDs: []uint{domain.ID}, OriginURL: "http://origin.example.com", Enabled: true}
	created, err := CreateProxyRoute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !created.EnableHTTP2 {
		t.Error("CreateProxyRoute(omitted HTTP2)=false, want true")
	}
	disabled := false
	input.EnableHTTP2 = &disabled
	updated, err := UpdateProxyRoute(ctx, created.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.EnableHTTP2 {
		t.Error("UpdateProxyRoute(false)=true, want false")
	}
	input.EnableHTTP2 = nil
	updated, err = UpdateProxyRoute(ctx, created.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetProxyRouteByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.EnableHTTP2 || stored.EnableHTTP2 {
		t.Error("UpdateProxyRoute(omitted HTTP2) reset disabled route")
	}
	input.EnableHTTP2 = &disabled
	input.SiteName = "disabled-create"
	domain = createZoneDomain(t, ctx, "disabled.example.com", nil)
	input.ZoneDomainIDs = []uint{domain.ID}
	created, err = CreateProxyRoute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.EnableHTTP2 {
		t.Error("CreateProxyRoute(false)=true, want false")
	}
}
