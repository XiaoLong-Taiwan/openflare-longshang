package proxy_route

import (
	"context"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
	"Wavelet/openflare/plugins/server/kernel/repository"
)

func TestListPageSearchAndBounds(t *testing.T) {
	t.Cleanup(setupProxyRouteTestDB(t))
	ctx := context.Background()
	for _, name := range []string{"alpha", "beta", "literal_%"} {
		if err := repository.CreateProxyRouteRecord(ctx, &model.ProxyRoute{SiteName: name, OriginURL: "http://origin.example.com", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	domain := createZoneDomain(t, ctx, "unique.example.com", nil)
	if err := repository.DB(ctx).Model(domain).Update("proxy_route_id", 1).Error; err != nil {
		t.Fatal(err)
	}
	for _, query := range []ListQuery{{Page: 1, PageSize: 1}, {Page: 2, PageSize: 1}, {Search: "UNIQUE"}, {Search: "_%"}, {Search: "missing"}, {Page: -1}, {PageSize: -1}} {
		got, err := ListPage(ctx, query)
		if query.Page < 0 || query.PageSize < 0 {
			if err == nil {
				t.Errorf("ListPage(%+v) error = nil, want invalid query", query)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ListPage(%+v) error = %v", query, err)
		}
		want := int64(3)
		if query.Search != "" {
			want = 1
		}
		if query.Search == "missing" {
			want = 0
		}
		if got.Total != want || len(got.Results) > got.PageSize {
			t.Errorf("ListPage(%+v) = %+v, want total %d and bounded page", query, got, want)
		}
	}
}
