// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"Wavelet/core"
)

// baselineRoutesFile 是改造前遗留注册路径导出的 (方法 路径) 全集。
const baselineRoutesFile = "docs/superpowers/specs/baseline/routes-engine.txt"

func TestPluginRoutesContainGoldenBaseline(t *testing.T) {
	app := newOpenFlareApp(core.ProfileAPI, core.WithConfigSource(testSource(t)))
	if err := app.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := app.Reconcile(); err != nil {
		t.Fatal(err)
	}

	got := routeSet(app.Context())
	want := loadBaseline(t)
	for _, drop := range []string{
		"GET /api/health",
		"GET /healthz",
		"POST /api/cap/challenge",
		"POST /api/cap/redeem",
	} {
		delete(want, drop)
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing golden route %s", k)
		}
	}
	for _, must := range []string{
		"GET /api/healthz",
		"POST /api/v1/cap/challenge",
		"POST /api/v1/cap/redeem",
	} {
		if !got[must] {
			t.Errorf("missing required route %s", must)
		}
	}
	for _, drop := range []string{
		"GET /api/health",
		"GET /healthz",
		"POST /api/cap/challenge",
		"POST /api/cap/redeem",
	} {
		if got[drop] {
			t.Errorf("removed route still registered: %s", drop)
		}
	}
}

func routeSet(ctx *core.Context) map[string]bool {
	set := make(map[string]bool)
	for _, rd := range ctx.Router().Routes() {
		set[rd.Method+" "+rd.Path] = true
	}
	return set
}

func loadBaseline(t *testing.T) map[string]bool {
	t.Helper()
	path := locateFile(t, baselineRoutesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read baseline %s: %v", path, err)
	}
	set := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[line] = true
		}
	}
	if len(set) == 0 {
		t.Fatalf("baseline %s is empty", path)
	}
	return set
}

func locateFile(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for range 8 {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Join(dir, "..")
	}
	t.Fatalf("%s not found above %s", rel, filepath.Dir(thisFile))
	return ""
}
