package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCloudflareCustomTargetUpgradeAndRollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "custom.db")), &gorm.Config{})
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
	if err := db.Exec("CREATE TABLE of_cf_pointing_groups (id INTEGER PRIMARY KEY, name TEXT, ttl INTEGER); INSERT INTO of_cf_pointing_groups VALUES (1, 'legacy', 600);").Error; err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("sqlite/00021_cloudflare_custom_target.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(data), "-- +goose Down")
	if !ok {
		t.Fatal("custom target migration missing Down")
	}
	if err := db.Exec(up).Error; err != nil {
		t.Fatalf("custom target Up error = %v, want nil", err)
	}
	var row struct {
		Name, TargetMode, RecordType, RecordContent string
		TTL                                         int
	}
	if err := db.Raw("SELECT * FROM of_cf_pointing_groups WHERE id = 1").Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Name != "legacy" || row.TTL != 600 || row.TargetMode != "" || row.RecordType != "" || row.RecordContent != "" {
		t.Errorf("legacy group after Up = %+v, want unchanged name/TTL and zero-value target fields", row)
	}
	pg, err := os.ReadFile("postgres/00021_cloudflare_custom_target.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pg), strings.TrimSpace(up)) || !strings.Contains(string(pg), "ALTER COLUMN desired_ip TYPE VARCHAR(253)") {
		t.Error("Postgres migration must share target columns and widen desired_ip")
	}
	if err := db.Exec(down).Error; err != nil {
		t.Fatalf("custom target Down error = %v, want nil", err)
	}
	if db.Migrator().HasColumn("of_cf_pointing_groups", "target_mode") {
		t.Error("Down retains target_mode, want removed")
	}
	if err := db.Raw("SELECT name, ttl FROM of_cf_pointing_groups WHERE id = 1").Scan(&row).Error; err != nil || row.Name != "legacy" || row.TTL != 600 {
		t.Errorf("legacy after Down = %+v, %v, want legacy TTL 600", row, err)
	}
}
