package driver_http

import (
	"strings"
	"testing"
)

func TestParseTrustedProxyConfigDefaultsToCloudflare(t *testing.T) {
	cidrs, err := parseTrustedProxyConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cidrs) != len(cloudflareProxyCIDRs) {
		t.Fatalf("got %d CIDRs, want %d", len(cidrs), len(cloudflareProxyCIDRs))
	}
}

func TestParseTrustedProxyConfigModes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "disabled", raw: `{"enabled":false}`, want: 0},
		{name: "override", raw: `{"enabled":true,"mode":"override","cidrs":["10.0.0.0/8"]}`, want: 1},
		{name: "append", raw: `{"enabled":true,"mode":"append","cidrs":["10.0.0.0/8"]}`, want: len(cloudflareProxyCIDRs) + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, err := parseTrustedProxyConfig(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(cidrs) != tt.want {
				t.Fatalf("got %d CIDRs, want %d", len(cidrs), tt.want)
			}
		})
	}
}

func TestParseTrustedProxyConfigRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{
		`{"enabled":true,"mode":"unknown"}`,
		`{"enabled":true,"mode":"override","cidrs":["not-a-cidr"]}`,
	} {
		if _, err := parseTrustedProxyConfig(raw); err == nil || !strings.Contains(err.Error(), "trusted proxy") {
			t.Fatalf("expected trusted proxy validation error for %s, got %v", raw, err)
		}
	}
}
