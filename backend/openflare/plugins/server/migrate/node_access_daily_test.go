package migrate

import (
	"os"
	"strings"
	"testing"
)

func TestNodeAccessDailySQLiteMigration(t *testing.T) {
	_, db := openStampDB(t)
	if _, err := db.Exec("CREATE TABLE of_node_access_logs (id BIGINT, logged_at DATETIME,node_id TEXT)"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("sqlite/00015_node_access_daily.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(data), "-- +goose Down")
	if _, err := db.Exec(parts[0]); err != nil {
		t.Fatalf("migration Up = %v, want nil", err)
	}
	if _, err := db.Exec("INSERT INTO of_node_access_daily (day,node_id,status_code,source) VALUES ('2026-11-03','n',200,'ingest')"); err != nil {
		t.Fatal(err)
	}
	var requests int
	if err := db.QueryRow("SELECT request_count FROM of_node_access_daily").Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if requests != 0 {
		t.Errorf("default request_count = %d, want 0", requests)
	}
	if _, err := db.Exec(parts[1]); err != nil {
		t.Fatalf("migration Down = %v, want nil", err)
	}
	if _, err := db.Exec(parts[0]); err != nil {
		t.Fatalf("migration re-Up = %v, want nil", err)
	}
	checkpoint, err := os.ReadFile("sqlite/00016_node_access_checkpoint.sql")
	if err != nil {
		t.Fatal(err)
	}
	steps := strings.Split(string(checkpoint), "-- +goose Down")
	for _, step := range []string{steps[0], steps[1], steps[0]} {
		if _, err := db.Exec(step); err != nil {
			t.Fatalf("checkpoint migration = %v, want nil", err)
		}
	}
	outbox, err := os.ReadFile("sqlite/00017_node_access_outbox.sql")
	if err != nil {
		t.Fatal(err)
	}
	steps = strings.Split(string(outbox), "-- +goose Down")
	for _, step := range []string{steps[0], steps[1], steps[0]} {
		if _, err := db.Exec(step); err != nil {
			t.Fatalf("outbox migration=%v, want nil", err)
		}
	}
}
