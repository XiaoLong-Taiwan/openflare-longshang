package openresty

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHTTP2RouteRenderWithPagesSWAndProxyOverrides(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, sw := range []bool{false, true} {
			cfg := ConfigSnapshot{HTTP3Enabled: true, SWOfflineEnabled: sw, SWOfflineDomains: []string{"example.com"}}
			for name, rendered := range map[string]string{
				"proxy": renderHTTPSServerWithProxyConfig("example.com", "site", "http://origin:8080", "", 1, nil, routeCacheConfig{}, routeLimitConfig{}, routeUpstreamConfig{}, false, false, "", "", sw, cfg, ProxyConfig{ProxyReadTimeout: "91s"}, enabled),
				"pages": renderHTTPSPagesServerHTTP2("example.com", "site", 1, nil, routeLimitConfig{}, false, false, "", "", sw, cfg, enabled),
			} {
				want := "http2 off;"
				if enabled {
					want = "http2 on;"
				}
				if !strings.Contains(rendered, want) || strings.Count(rendered, "http2 ") != 1 {
					t.Errorf("Render(%s, HTTP2=%t, SW=%t) missing %q", name, enabled, sw, want)
				}
				if !strings.Contains(rendered, "listen 443 quic;") {
					t.Errorf("Render(%s, HTTP2=%t) lost HTTP3", name, enabled)
				}
				if name == "proxy" && !strings.Contains(rendered, "proxy_read_timeout 91s;") {
					t.Error("HTTP2 render lost proxy override")
				}
			}
		}
	}
}

func TestHTTP2LegacySnapshotDefaultsToEnabled(t *testing.T) {
	for _, input := range []string{`{"enable_https":true}`, `{"enable_https":true,"enable_http2":false}`, `{"enable_https":true,"enable_http2":true}`} {
		var route Route
		if err := json.Unmarshal([]byte(input), &route); err != nil {
			t.Fatal(err)
		}
		got := route.EnableHTTP2 == nil || *route.EnableHTTP2
		want := !strings.Contains(input, `"enable_http2":false`)
		if got != want {
			t.Errorf("HTTP2(%s)=%t, want %t", input, got, want)
		}
	}
}
