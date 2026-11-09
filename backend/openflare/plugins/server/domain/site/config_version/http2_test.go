package config_version

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
	openresty "Wavelet/openflare/share/render/openresty"
)

func TestHTTP2SnapshotAndPolicyDiff(t *testing.T) {
	t.Cleanup(setupConfigVersionTestDB(t))
	ctx := context.Background()
	zone := &model.Zone{Domain: "example.com"}
	if err := repository.DB(ctx).Create(zone).Error; err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		route := &model.ProxyRoute{SiteName: "http2", OriginURL: "http://origin:8080", Upstreams: `[]`, EnableHTTP2: enabled}
		if err := repository.CreateProxyRouteRecord(ctx, route); err != nil {
			t.Fatal(err)
		}
		if err := repository.DB(ctx).Create(&model.ZoneDomain{ZoneID: zone.ID, ProxyRouteID: &route.ID, Domain: fmt.Sprintf("http2-%t.example.com", enabled)}).Error; err != nil {
			t.Fatal(err)
		}
		routes, err := buildSnapshotRoutes(ctx, []*model.ProxyRoute{route})
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(routes[0])
		if err != nil {
			t.Fatal(err)
		}
		var rendered openresty.Route
		if err := json.Unmarshal(data, &rendered); err != nil {
			t.Fatal(err)
		}
		if rendered.EnableHTTP2 == nil || *rendered.EnableHTTP2 != enabled {
			t.Errorf("Snapshot(HTTP2=%t)=%s, want explicit HTTP2", enabled, data)
		}
	}
	if snapshotRoutePolicyEqual(snapshotRoute{EnableHTTP2: true}, snapshotRoute{EnableHTTP2: false}) {
		t.Error("snapshotRoutePolicyEqual(HTTP2 changed)=true, want false")
	}
}

func TestHTTP2LegacySnapshotPolicyDefaultsToEnabled(t *testing.T) {
	for _, input := range []string{`{}`, `{"enable_http2":true}`, `{"enable_http2":false}`} {
		var route snapshotRoute
		if err := json.Unmarshal([]byte(input), &route); err != nil {
			t.Fatal(err)
		}
		want := input != `{"enable_http2":false}`
		if route.EnableHTTP2 != want {
			t.Errorf("UnmarshalJSON(%s).EnableHTTP2 = %t, want %t", input, route.EnableHTTP2, want)
		}
	}
}
