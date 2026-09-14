// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"
	"time"

	"Wavelet/core"
	"Wavelet/openflare/plugins/agent/config"
)

func TestPluginIdentity(t *testing.T) {
	p := New("./agent.json")
	if got := p.Name(); got != "agent" {
		t.Errorf("Name() = %q, want %q", got, "agent")
	}
	if got := p.Type(); got != DriverTypeAgent {
		t.Errorf("Type() = %q, want %q", got, DriverTypeAgent)
	}
	// 驱动类型必须等于 profile 字符串，否则内核的 profile 过滤会漏掉本驱动。
	if got, want := string(p.Type()), string(core.Profile("agent")); got != want {
		t.Errorf("driver type %q must equal profile %q", got, want)
	}
}

func TestApplyFailsOnMissingConfig(t *testing.T) {
	ctx := core.NewContext(context.Background())
	if err := New("./does-not-exist.json").Apply(ctx); err == nil {
		t.Fatal("Apply(missing config) error = nil, want error")
	}
}

func TestNewGeoIPUpdaterWiresCountryAndCity(t *testing.T) {
	cfg := &config.Config{
		MMDBPath:            "/data/GeoLite2-Country.mmdb",
		MMDBDownloadURL:     "https://geo.example/GeoLite2-Country.mmdb",
		CityMMDBPath:        "/data/GeoLite2-City.mmdb",
		CityMMDBDownloadURL: "https://geo.example/GeoLite2-City.mmdb",
		MMDBUpdateInterval:  config.MillisecondDuration(time.Hour),
	}
	got := newGeoIPUpdater(cfg)
	if got.MMDBPath != cfg.MMDBPath || got.DownloadURL != cfg.MMDBDownloadURL ||
		got.CityMMDBPath != cfg.CityMMDBPath || got.CityDownloadURL != cfg.CityMMDBDownloadURL {
		t.Fatalf("GeoIP updater wiring incomplete: %#v", got)
	}
	if want := cfg.MMDBUpdateInterval.Duration(); got.UpdateInterval != want {
		t.Errorf("UpdateInterval = %v, want %v", got.UpdateInterval, want)
	}
}
