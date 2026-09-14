// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/runtimeconfig"
)

func TestSealAndOpenSensitiveValue(t *testing.T) {
	previous := runtimeconfig.Get()
	SetSessionSecret("cloudflare-pointing-test-secret")
	t.Cleanup(func() {
		SetSessionSecret(previous.SessionSecret)
		runtimeconfig.Set(previous)
	})

	sealed, err := Seal(`{"api_token":"secret-token"}`)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if !strings.HasPrefix(sealed, Prefix) {
		t.Fatalf("Seal() = %q, want prefix %q", sealed, Prefix)
	}
	if strings.Contains(sealed, "secret-token") {
		t.Fatalf("Seal() = %q, want token redacted", sealed)
	}

	opened, err := Open(sealed)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if want := `{"api_token":"secret-token"}`; opened != want {
		t.Errorf("Open(Seal(value)) = %q, want %q", opened, want)
	}
}

func TestSealWithoutSessionSecretKeepsPlaintextCompatibility(t *testing.T) {
	previous := runtimeconfig.Get()
	SetSessionSecret("")
	runtimeconfig.Set(runtimeconfig.Snapshot{})
	t.Cleanup(func() {
		SetSessionSecret(previous.SessionSecret)
		runtimeconfig.Set(previous)
	})

	sealed, err := Seal(" legacy-value ")
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if sealed != "legacy-value" {
		t.Errorf("Seal() = %q, want %q", sealed, "legacy-value")
	}

	opened, err := Open(sealed)
	if err != nil {
		t.Fatalf("Open(plaintext) error = %v", err)
	}
	if opened != "legacy-value" {
		t.Errorf("Open(plaintext) = %q, want %q", opened, "legacy-value")
	}
}

func TestOpenEncryptedValueRequiresSessionSecret(t *testing.T) {
	previous := runtimeconfig.Get()
	SetSessionSecret("cloudflare-pointing-test-secret")
	sealed, err := Seal("secret")
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	SetSessionSecret("")
	runtimeconfig.Set(runtimeconfig.Snapshot{})
	t.Cleanup(func() {
		SetSessionSecret(previous.SessionSecret)
		runtimeconfig.Set(previous)
	})
	if _, err := Open(sealed); err == nil {
		t.Fatal("Open(encrypted) error = nil, want missing session secret error")
	}
}
