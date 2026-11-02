package observability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/state"
)

func TestCollectAccessLogsBatchLeavesUnreadRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	line := "{\"ts\":\"2026-11-02T08:00:00Z\",\"path\":\"/\",\"status\":200}\n"
	if err := os.WriteFile(path, []byte(strings.Repeat(line, 1001)), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AccessLogPath: path}
	store := state.NewStore(filepath.Join(dir, "state.json"))
	first := CollectAccessLogs(cfg, store)
	second := CollectAccessLogs(cfg, store)
	if len(first) != 1000 || len(second) != 1 {
		t.Fatalf("CollectAccessLogs(1001 rows) batches = %d, %d, want 1000, 1", len(first), len(second))
	}
}

func TestCollectAccessLogsWaitsForCompleteLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	line := "{\"ts\":\"2026-11-02T08:00:00Z\",\"path\":\"/\",\"status\":200}"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AccessLogPath: path}
	store := state.NewStore(filepath.Join(dir, "state.json"))
	if got := len(CollectAccessLogs(cfg, store)); got != 0 {
		t.Fatalf("CollectAccessLogs(unterminated line) = %d rows, want 0", got)
	}
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := len(CollectAccessLogs(cfg, store)); got != 1 {
		t.Fatalf("CollectAccessLogs(completed line) = %d rows, want 1", got)
	}
}

func TestCollectAccessLogsBoundsHugeLineAcrossRestarts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	line := "{\"ts\":\"2026-11-02T08:00:00Z\",\"path\":\"/after\",\"status\":200}\n"
	data := strings.Repeat("x", 2*1024*1024+123) + "\n" + line
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AccessLogPath: path}
	statePath := filepath.Join(dir, "state.json")
	for i := 1; i <= 2; i++ {
		store := state.NewStore(statePath)
		if logs := CollectAccessLogs(cfg, store); len(logs) != 0 {
			t.Fatalf("CollectAccessLogs(huge line round %d) = %v, want empty", i, logs)
		}
		snapshot, err := store.Load()
		if err != nil || snapshot.AccessLogOffset != int64(i*1024*1024) || !snapshot.AccessLogDiscarding {
			t.Fatalf("CollectAccessLogs(round %d) snapshot=%+v error=%v, want bounded offset and persisted discard", i, snapshot, err)
		}
	}
	store := state.NewStore(statePath)
	logs := CollectAccessLogs(cfg, store)
	if len(logs) != 1 || logs[0].Path != "/after" {
		t.Fatalf("CollectAccessLogs(after huge line) = %v, want /after only", logs)
	}
}

func TestCollectAccessLogsLimitsEncodedSingleRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	line := "{\"ts\":\"2026-11-02T08:00:00Z\",\"user_agent\":\"" + strings.Repeat("<", 3000) + "\",\"status\":200}\n"
	line += "{\"ts\":\"2026-11-02T08:00:00Z\",\"path\":\"/ok\",\"status\":200}\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	logs := CollectAccessLogs(&config.Config{AccessLogPath: path}, state.NewStore(filepath.Join(dir, "state.json")))
	if len(logs) != 1 || logs[0].Path != "/ok" {
		t.Fatalf("CollectAccessLogs(JSON escaping expansion) = %v, want /ok only", logs)
	}
}
