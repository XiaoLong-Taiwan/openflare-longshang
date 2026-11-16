package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCloudflareHealthScheduleMigrationDeduplicatesRows(t *testing.T) {
	const canonical = "cloudflare:sync_all_groups"
	const legacy = "of_cloudflare_sync_all_groups"
	type schedule struct {
		ID                   int
		Name, TaskType, Cron string
		Payload              sql.NullString
		IsActive             bool
		CreatedAt, UpdatedAt string
	}
	type testCase struct {
		name       string
		firstType  string
		firstOn    bool
		secondType string
		secondOn   bool
	}
	tests := []testCase{{name: "missing"}}
	for _, firstType := range []string{canonical, legacy} {
		for _, firstOn := range []bool{false, true} {
			for _, secondType := range []string{"", canonical, legacy} {
				for _, secondOn := range []bool{false, true} {
					if secondType == "" && secondOn {
						continue
					}
					tests = append(tests, testCase{
						name:      fmt.Sprintf("first=%s_on=%t_second=%s_on=%t", firstType, firstOn, secondType, secondOn),
						firstType: firstType, firstOn: firstOn, secondType: secondType, secondOn: secondOn,
					})
				}
			}
		}
	}
	for _, dialect := range []string{"sqlite", "postgres"} {
		data, err := os.ReadFile(filepath.Join(dialect, "00026_cloudflare_sync_all_groups_schedule.sql"))
		if err != nil {
			t.Fatal(err)
		}
		up, down, ok := strings.Cut(string(data), "-- +goose Down")
		if !ok {
			t.Fatalf("%s schedule migration missing Down", dialect)
		}
		for _, tt := range tests {
			t.Run(dialect+"/"+tt.name, func(t *testing.T) {
				db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "schedule.db")), &gorm.Config{})
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
				if err := db.Exec(`CREATE TABLE w_schedules (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, task_type TEXT NOT NULL, cron TEXT NOT NULL, payload TEXT, is_active BOOLEAN NOT NULL, created_at DATETIME, updated_at DATETIME)`).Error; err != nil {
					t.Fatal(err)
				}
				unrelated := schedule{ID: 900, Name: "unrelated", TaskType: "of_ssl_renew", Cron: "0 1 * * *", IsActive: true, CreatedAt: "2026-01-01 00:00:00", UpdatedAt: "2026-01-02 00:00:00"}
				if err := db.Table("w_schedules").Create(&unrelated).Error; err != nil {
					t.Fatal(err)
				}
				first := schedule{ID: 42, Name: "custom health schedule", TaskType: tt.firstType, Cron: "17 */6 * * *", Payload: sql.NullString{String: "null", Valid: true}, IsActive: tt.firstOn, CreatedAt: "2026-02-01 00:00:00", UpdatedAt: "2026-02-02 00:00:00"}
				if tt.firstType == legacy {
					first.Payload = sql.NullString{}
				}
				if tt.secondType != "" {
					second := schedule{ID: 80, Name: "duplicate", TaskType: tt.secondType, Cron: "*/5 * * * *", IsActive: tt.secondOn}
					if err := db.Table("w_schedules").Create(&second).Error; err != nil {
						t.Fatal(err)
					}
				}
				if tt.firstType != "" {
					if err := db.Table("w_schedules").Create(&first).Error; err != nil {
						t.Fatal(err)
					}
				}
				var snapshot []schedule
				for _, step := range []struct{ name, sql string }{{"Up", up}, {"repeat Up", up}, {"Down", down}, {"Up after Down", up}} {
					if err := db.Exec(step.sql).Error; err != nil {
						t.Fatalf("00026 %s error = %v, want nil", step.name, err)
					}
					var rows []schedule
					if err := db.Raw(`SELECT id, name, task_type, cron, payload, is_active, CAST(created_at AS TEXT) AS created_at, CAST(updated_at AS TEXT) AS updated_at FROM w_schedules ORDER BY id`).Scan(&rows).Error; err != nil {
						t.Fatal(err)
					}
					if len(rows) != 2 {
						t.Fatalf("00026 %s rows = %+v, want one health schedule and unrelated row", step.name, rows)
					}
					if step.name == "Up" {
						if tt.firstType != "" {
							first.TaskType = canonical
							if rows[0] != first || rows[1] != unrelated {
								t.Errorf("00026 Up rows = %+v, want %+v", rows, []schedule{first, unrelated})
							}
						} else {
							seed := rows[1]
							if rows[0] != unrelated || seed.ID <= unrelated.ID || seed.TaskType != canonical || seed.Cron != "0 */4 * * *" || seed.Payload != (sql.NullString{String: "{}", Valid: true}) || !seed.IsActive || seed.Name == "" || seed.CreatedAt == "" || seed.UpdatedAt == "" {
								t.Errorf("00026 Up missing schedule rows = %+v, want unchanged unrelated row and auto-ID active four-hour seed", rows)
							}
						}
						snapshot = rows
					} else if !reflect.DeepEqual(rows, snapshot) {
						t.Errorf("00026 %s rows = %+v, want unchanged %+v (Down is irreversible no-op)", step.name, rows, snapshot)
					}
				}
			})
		}
	}
}
