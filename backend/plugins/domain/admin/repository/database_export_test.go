package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"Wavelet/plugins/domain/admin/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestExportSQLiteSnapshotIncludesWAL(t *testing.T) {
	dir := t.TempDir()
	source, err := gorm.Open(sqlite.Open(filepath.Join(dir, "source.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := source.DB()
	if err != nil {
		t.Fatal(err)
	}
	repository.SetDBService(stubDBService{db: source})
	t.Cleanup(func() {
		repository.SetDBService(nil)
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, query := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA wal_autocheckpoint = 0",
		"CREATE TABLE snapshot_data (value TEXT)",
		"INSERT INTO snapshot_data VALUES ('latest committed data')",
	} {
		if err := source.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(dir, "snapshot.db")
	if err := repository.ExportSQLiteSnapshot(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	snapshot, err := gorm.Open(sqlite.Open(target), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	snapshotDB, err := snapshot.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := snapshotDB.Close(); err != nil {
			t.Error(err)
		}
	})
	var value string
	if err := snapshot.Raw("SELECT value FROM snapshot_data").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != "latest committed data" {
		t.Errorf("ExportSQLiteSnapshot(WAL) value = %q, want latest committed data", value)
	}
	var integrity string
	if err := snapshot.Raw("PRAGMA integrity_check").Scan(&integrity).Error; err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Errorf("ExportSQLiteSnapshot(WAL) integrity = %q, want ok", integrity)
	}
}
