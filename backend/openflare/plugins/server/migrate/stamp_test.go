// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"Wavelet/core"
	"Wavelet/core/contracts"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const goldGooseVersion int64 = 202608090003

type testDB struct {
	db *gorm.DB
}

func (s testDB) GORM() *gorm.DB { return s.db }

func (s testDB) DB(ctx context.Context) *gorm.DB { return s.db.WithContext(ctx) }

func (s testDB) Named(string) *gorm.DB { return s.db }

func openStampDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "stamp.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error = %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("gdb.DB() error = %v", err)
	}
	return gdb, sqlDB
}

func createSchemaVersions(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS w_schema_versions (
		plugin_id   VARCHAR(64)  NOT NULL,
		version_id  BIGINT       NOT NULL,
		applied_at  DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (plugin_id, version_id)
	)`)
	if err != nil {
		t.Fatalf("create w_schema_versions error = %v", err)
	}
}

func createGooseDBVersion(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp DATETIME
	)`)
	if err != nil {
		t.Fatalf("create goose_db_version error = %v", err)
	}
	_, err = db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1), (?, 1)`, version)
	if err != nil {
		t.Fatalf("insert goose_db_version error = %v", err)
	}
}

func callLegacy(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	ctx := core.NewContext(context.Background())
	core.Provide[contracts.DBService](ctx, testDB{db: gdb})
	if err := Legacy(ctx); err != nil {
		t.Fatalf("Legacy() error = %v", err)
	}
}

func listStamps(t *testing.T, db *sql.DB) []stampRow {
	t.Helper()
	rows, err := db.Query(`SELECT plugin_id, version_id FROM w_schema_versions ORDER BY plugin_id, version_id`)
	if err != nil {
		t.Fatalf("list w_schema_versions error = %v", err)
	}
	defer func() { _ = rows.Close() }()

	var got []stampRow
	for rows.Next() {
		var r stampRow
		if err := rows.Scan(&r.PluginID, &r.VersionID); err != nil {
			t.Fatalf("scan w_schema_versions error = %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}
	return got
}

type stampRow struct {
	PluginID  string
	VersionID int64
}

func TestLegacyStampsGoldVersionIdempotent(t *testing.T) {
	gdb, sqlDB := openStampDB(t)
	createSchemaVersions(t, sqlDB)
	createGooseDBVersion(t, sqlDB, goldGooseVersion)

	callLegacy(t, gdb)

	got := listStamps(t, sqlDB)
	want := []stampRow{
		{PluginID: "openflare/legacy", VersionID: 0},
		{PluginID: "openflare/legacy", VersionID: goldGooseVersion},
		{PluginID: "server", VersionID: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("Legacy() stamps = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Legacy() stamps[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	callLegacy(t, gdb)
	gotAgain := listStamps(t, sqlDB)
	if len(gotAgain) != len(want) {
		t.Fatalf("Legacy() second call rows = %d, want %d", len(gotAgain), len(want))
	}
}

func TestLegacyWithoutGooseTableDoesNotStampLegacy(t *testing.T) {
	gdb, sqlDB := openStampDB(t)
	createSchemaVersions(t, sqlDB)

	callLegacy(t, gdb)

	got := listStamps(t, sqlDB)
	for _, row := range got {
		if row.PluginID == "openflare/legacy" {
			t.Errorf("Legacy() without goose_db_version wrote openflare/legacy %+v, want none", row)
		}
	}
}
