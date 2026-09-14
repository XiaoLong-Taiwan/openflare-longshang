// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"strings"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/runtimeconfig"

	"github.com/pressly/goose/v3"
)

func TestClickHouseMigrationFilesEmbedded(t *testing.T) {
	entries, err := clickhouseMigrationFS.ReadDir(clickhouseMigrationDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", clickhouseMigrationDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded ClickHouse migrations, got none")
	}

	foundNodeLogs := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.Contains(entry.Name(), "user_access") {
			t.Errorf("user access log migration %s must not be embedded", entry.Name())
		}
		if entry.Name() == "202606200001_create_node_access_logs.sql" {
			foundNodeLogs = true
		}
	}
	if !foundNodeLogs {
		t.Fatal("expected 202606200001_create_node_access_logs.sql in embedded migrations")
	}
}

func TestClickHouseGooseDialect(t *testing.T) {
	if err := goose.SetDialect("clickhouse"); err != nil {
		t.Fatalf("SetDialect(clickhouse) error = %v", err)
	}
}

func TestUpSkipsWhenDisabled(t *testing.T) {
	t.Cleanup(runtimeconfig.Override(runtimeconfig.DatabaseEnabled(), false))
	if err := UpClickHouse(); err != nil {
		t.Fatalf("UpClickHouse() error = %v, want nil when ClickHouse disabled", err)
	}
}
