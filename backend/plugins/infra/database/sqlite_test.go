package database

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnableSQLiteWAL(t *testing.T) {
	for _, tc := range []struct{ name, path, mode string }{
		{"file", filepath.Join(t.TempDir(), "database.db"), "wal"},
		{"memory", ":memory:", "memory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, err := gorm.Open(sqlite.Open(tc.path), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := database.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := sqlDB.Close(); err != nil {
					t.Error(err)
				}
			})
			if err := enableSQLiteWAL(database); err != nil {
				t.Fatal(err)
			}
			var mode string
			if err := database.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
				t.Fatal(err)
			}
			if mode != tc.mode {
				t.Fatalf("journal mode = %q, want %q", mode, tc.mode)
			}
			if err := enableSQLiteWAL(database); err != nil {
				t.Fatalf("enable WAL again: %v", err)
			}
		})
	}
}
