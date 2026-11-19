package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteOutboxRetryMigrationPreservesExistingRows(t *testing.T) {
	data, err := os.ReadFile("sqlite/00025_outbox_retry_state.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(data), "-- +goose Down")
	if !ok {
		t.Fatal("00025 missing Down section")
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "outbox.db")), &gorm.Config{})
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
	if err := db.Exec(`CREATE TABLE of_node_access_outbox (id BIGINT PRIMARY KEY, payload TEXT NOT NULL DEFAULT '', created_at DATETIME);
INSERT INTO of_node_access_outbox VALUES (42, '{"node_id":"node-a"}', '2026-01-01 00:00:00'), (43, '', NULL);`).Error; err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ name, sql string }{{"Up", up}, {"Down", down}, {"Up after Down", up}} {
		if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(step.sql).Error }); err != nil {
			t.Fatalf("00025 %s with existing rows error = %v, want nil", step.name, err)
		}
		var count int64
		if err := db.Raw(`SELECT count(*) FROM of_node_access_outbox WHERE (id = 42 AND payload = '{"node_id":"node-a"}' AND created_at = '2026-01-01 00:00:00') OR (id = 43 AND payload = '' AND created_at IS NULL)`).Scan(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("00025 %s preserved rows = %d, want 2", step.name, count)
		}
	}
	if err := db.Exec(`INSERT INTO of_node_access_outbox (id, payload) VALUES (44, 'new')`).Error; err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Attempts      int
		NextAttemptAt time.Time
		LastError     string
	}
	if err := db.Raw(`SELECT attempts, next_attempt_at, last_error FROM of_node_access_outbox ORDER BY id`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("00025 rows = %d, want 3", len(rows))
	}
	for i, row := range rows {
		if row.Attempts != 0 || row.LastError != "" || row.NextAttemptAt.IsZero() || row.NextAttemptAt.After(time.Now().UTC()) {
			t.Errorf("00025 retry defaults row %d = %+v, want zero attempts, empty error and due timestamp", i, row)
		}
	}
	var indexes int64
	if err := db.Raw(`SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_node_access_outbox_retry'`).Scan(&indexes).Error; err != nil {
		t.Fatal(err)
	}
	if indexes != 1 {
		t.Errorf("00025 retry indexes = %d, want 1", indexes)
	}
}
