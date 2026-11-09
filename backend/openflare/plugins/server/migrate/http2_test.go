package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestHTTP2MigrationPreservesLegacyRoutes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "http2.db")), &gorm.Config{})
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
	if err := db.Exec("CREATE TABLE of_proxy_routes (id INTEGER PRIMARY KEY, site_name TEXT); INSERT INTO of_proxy_routes VALUES (1, 'legacy');").Error; err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("sqlite/00020_proxy_route_http2.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(data), "-- +goose Down")
	if !ok {
		t.Fatal("HTTP2 migration missing Down")
	}
	if err := db.Exec(up).Error; err != nil {
		t.Fatal(err)
	}
	var enabled bool
	if err := db.Raw("SELECT enable_http2 FROM of_proxy_routes WHERE id = 1").Scan(&enabled).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("HTTP2 migration legacy route=false, want true")
	}
	if err := db.Exec("UPDATE of_proxy_routes SET enable_http2 = 0 WHERE id = 1").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(down).Error; err != nil {
		t.Fatal(err)
	}
	var name string
	if err := db.Raw("SELECT site_name FROM of_proxy_routes WHERE id = 1").Scan(&name).Error; err != nil || name != "legacy" {
		t.Fatalf("HTTP2 Down name=%q error=%v, want legacy", name, err)
	}
}
