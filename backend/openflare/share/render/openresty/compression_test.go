package openresty

import (
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/model"
)

func TestDefaultCompressionTemplates(t *testing.T) {
	for _, template := range []string{"", model.DefaultOpenRestyMainConfigTemplate} {
		for _, enabled := range []bool{false, true} {
			got := RenderMainConfig(Document{OpenRestyConfig: ConfigSnapshot{
				MainConfigTemplate: template,
				GzipEnabled:        enabled,
			}})
			for _, want := range []string{
				"gzip " + onOff(enabled) + ";",
				"gzip_vary on;",
				"gzip_types text/plain text/css text/xml application/javascript application/json application/xml application/rss+xml application/atom+xml image/svg+xml;",
				"proxy_request_buffering off;",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("RenderMainConfig(template=%q, gzip=%t) = %q, want %q", template, enabled, got, want)
				}
			}
			if strings.Contains(got, "text/event-stream") {
				t.Errorf("RenderMainConfig(gzip=%t) = %q, want no SSE compression type", enabled, got)
			}
		}
	}
}

func TestSavedCompressionTemplatePreserved(t *testing.T) {
	legacy := strings.ReplaceAll(defaultMainConfigTemplate, "    gzip_vary on;\n", "")
	legacy = strings.ReplaceAll(legacy, "    gzip_types text/plain text/css text/xml application/javascript application/json application/xml application/rss+xml application/atom+xml image/svg+xml;\n", "")
	custom := strings.ReplaceAll(defaultMainConfigTemplate, "gzip_vary on;", "gzip_vary off;")
	for _, template := range []string{legacy, custom} {
		got := RenderMainConfig(Document{OpenRestyConfig: ConfigSnapshot{MainConfigTemplate: template}})
		if strings.Contains(got, "gzip_vary on;") {
			t.Errorf("RenderMainConfig(template=%q) = %q, want saved gzip directives preserved", template, got)
		}
	}
}

func TestRequestBufferingGlobalAndRouteOverride(t *testing.T) {
	enabled := true
	for _, global := range []bool{false, true} {
		main := RenderMainConfig(Document{OpenRestyConfig: ConfigSnapshot{ProxyRequestBuffering: global}})
		if want := "proxy_request_buffering " + onOff(global) + ";"; !strings.Contains(main, want) {
			t.Errorf("RenderMainConfig(buffering=%t) = %q, want %q", global, main, want)
		}
		for _, override := range []*bool{nil, new(bool), &enabled} {
			got, err := RenderRouteConfig(Document{
				OpenRestyConfig: ConfigSnapshot{ProxyRequestBuffering: global},
				Routes: []Route{{ID: 1, Domains: []string{"example.com"}, Enabled: true,
					Upstreams:   []string{"http://127.0.0.1:8080"},
					ProxyConfig: ProxyConfig{ProxyRequestBuffering: override},
				}},
			}, nil)
			if err != nil {
				t.Fatalf("RenderRouteConfig(buffering=%t, override=%v) error = %v", global, override, err)
			}
			if override == nil {
				if strings.Contains(got, "proxy_request_buffering ") {
					t.Errorf("RenderRouteConfig(override=nil) = %q, want inherited buffering", got)
				}
			} else if want := "proxy_request_buffering " + onOff(*override) + ";"; !strings.Contains(got, want) {
				t.Errorf("RenderRouteConfig(override=%t) = %q, want %q", *override, got, want)
			}
		}
	}
}
