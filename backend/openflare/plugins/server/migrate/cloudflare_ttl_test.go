package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCloudflareTTLUpgradePreservesLegacyGroups(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ttl.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.Exec("CREATE TABLE of_cf_pointing_groups (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO of_cf_pointing_groups VALUES (1, 'legacy');").Error; err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("sqlite/00019_cloudflare_group_ttl.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(data), "-- +goose Down")
	if !ok {
		t.Fatal("TTL migration missing Down")
	}
	pg, err := os.ReadFile("postgres/00019_cloudflare_group_ttl.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(pg) != string(data) {
		t.Fatal("TTL migration dialects differ")
	}
	if err := db.Exec(up).Error; err != nil {
		t.Fatal(err)
	}
	var ttl int
	if err := db.Raw("SELECT ttl FROM of_cf_pointing_groups WHERE id = 1").Scan(&ttl).Error; err != nil {
		t.Fatal(err)
	}
	if ttl != 0 {
		t.Errorf("legacy TTL = %d, want 0", ttl)
	}
	if err := db.Exec(down).Error; err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.Raw("SELECT name FROM of_cf_pointing_groups WHERE id = 1").Scan(&name).Error; err != nil || name != "legacy" {
		t.Fatalf("legacy group after Down = %q, %v, want legacy", name, err)
	}
}
