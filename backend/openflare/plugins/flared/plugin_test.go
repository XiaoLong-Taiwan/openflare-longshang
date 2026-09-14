// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package flared

import (
	"context"
	"testing"

	"Wavelet/core"
)

func TestPluginIdentity(t *testing.T) {
	p := New("./flared.json")
	if got := p.Name(); got != "flared" {
		t.Errorf("Name() = %q, want %q", got, "flared")
	}
	// 驱动类型必须等于 profile 字符串，否则内核的 profile 过滤会漏掉本驱动。
	if got, want := string(p.Type()), string(core.Profile("flared")); got != want {
		t.Errorf("Type() = %q, want profile %q", got, want)
	}
}

func TestApplyFailsOnMissingConfig(t *testing.T) {
	ctx := core.NewContext(context.Background())
	if err := New("./does-not-exist.json").Apply(ctx); err == nil {
		t.Fatal("Apply(missing config) error = nil, want error")
	}
}
