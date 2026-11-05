package migrate

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestNodeAccessPostgresIndexMigrationSafety(t *testing.T) {
	files := []string{"00016_node_access_checkpoint.sql", "00017_node_access_outbox.sql"}
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_node_access_pending ON ONLY of_node_access_logs (summary_tracked, logged_at, id);",
		"CREATE INDEX IF NOT EXISTS idx_node_access_sample ON ONLY of_node_access_logs (node_id, logged_at, sample_key);",
	}
	unsafeIndex := regexp.MustCompile(`(?i)CREATE\s+INDEX[^;]*\bON\s+(?:ONLY\s+)?of_node_access_logs\b[^;]*;`)
	for i, name := range files {
		data, err := os.ReadFile("postgres/" + name)
		if err != nil {
			t.Fatal(err)
		}
		up, down, ok := strings.Cut(strings.ReplaceAll(string(data), "\r\n", "\n"), "-- +goose Down")
		if !ok {
			t.Fatalf("migration %s has no Down section, want reversible migration", name)
		}
		guard := "IF EXISTS (SELECT 1 FROM pg_class WHERE oid = 'of_node_access_logs'::regclass AND relkind = 'p') THEN"
		block := "-- +goose StatementBegin\nDO $$\nBEGIN\n    " + guard + "\n        " + indexes[i] + "\n    END IF;\nEND\n$$;\n-- +goose StatementEnd"
		if !strings.Contains(up, block) {
			t.Errorf("migration %s missing guarded ON ONLY block, want metadata-only partition index and no ordinary-table build", name)
		}
		matches := unsafeIndex.FindAllString(up, -1)
		if len(matches) != 1 || matches[0] != indexes[i] {
			t.Errorf("migration %s log index statements = %q, want only %q", name, matches, indexes[i])
		}
		indexName := strings.Fields(indexes[i])[5]
		if !strings.Contains(down, "DROP INDEX IF EXISTS "+indexName+";") {
			t.Errorf("migration %s Down missing drop for %s, want matching index cleanup", name, indexName)
		}
	}
}

func TestNodeAccessPostgresIndexMigrationIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("Close() = %v, want nil", err)
		}
	})
	for _, partitioned := range []bool{false, true} {
		tx := db.Begin()
		if tx.Error != nil {
			t.Fatal(tx.Error)
		}
		t.Cleanup(func() { tx.Rollback() })
		schema := "node_access_index_" + time.Now().Format("20060102150405.000000000")
		schema = strings.ReplaceAll(schema, ".", "_")
		setup := `CREATE SCHEMA "` + schema + `"; SET LOCAL search_path TO "` + schema + `";
CREATE TABLE of_node_access_logs (id BIGINT, logged_at TIMESTAMPTZ, node_id TEXT, summary_tracked BOOLEAN NOT NULL DEFAULT false)`
		if partitioned {
			setup += ` PARTITION BY RANGE (logged_at)`
		}
		setup += `;
CREATE TABLE of_node_access_batches (batch_key TEXT PRIMARY KEY);
CREATE TABLE of_node_access_quota (day TEXT);`
		if partitioned {
			setup += `CREATE TABLE old_partition PARTITION OF of_node_access_logs FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');`
		}
		setup += `INSERT INTO of_node_access_logs VALUES (1, '2026-01-02', 'n', false);`
		if err := tx.Exec(setup).Error; err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"00016_node_access_checkpoint.sql", "00017_node_access_outbox.sql"} {
			data, err := os.ReadFile("postgres/" + name)
			if err != nil {
				t.Fatal(err)
			}
			up, _, _ := strings.Cut(string(data), "-- +goose Down")
			for range 2 {
				if err := tx.Exec(up).Error; err != nil {
					t.Fatalf("migration %s (partitioned=%t) = %v, want idempotent Up", name, partitioned, err)
				}
			}
		}
		var indexCount int64
		if err := tx.Raw(`SELECT count(*) FROM pg_index WHERE indrelid = 'of_node_access_logs'::regclass`).Scan(&indexCount).Error; err != nil {
			t.Fatal(err)
		}
		want := int64(0)
		if partitioned {
			want = 2
		}
		if indexCount != want {
			t.Errorf("parent index count (partitioned=%t) = %d, want %d", partitioned, indexCount, want)
		}
		if partitioned {
			if err := tx.Raw(`SELECT count(*) FROM pg_index WHERE indrelid = 'old_partition'::regclass`).Scan(&indexCount).Error; err != nil {
				t.Fatal(err)
			}
			if indexCount != 0 {
				t.Errorf("old partition index count = %d, want 0 (no recursive build)", indexCount)
			}
			if err := tx.Exec(`CREATE TABLE new_partition PARTITION OF of_node_access_logs FOR VALUES FROM ('2026-02-01') TO ('2026-03-01')`).Error; err != nil {
				t.Fatal(err)
			}
			if err := tx.Raw(`SELECT count(*) FROM pg_index WHERE indrelid = 'new_partition'::regclass`).Scan(&indexCount).Error; err != nil {
				t.Fatal(err)
			}
			if indexCount != 2 {
				t.Errorf("new partition index count = %d, want 2 inherited indexes", indexCount)
			}
		}
		if err := tx.Rollback().Error; err != nil {
			t.Fatal(err)
		}
	}
}
