//go:build !linux

package nginx

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestParsePSRuntimeSeparatesManagedAndForeignMasters(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nginx.conf")
	otherConfigPath := filepath.Join(t.TempDir(), "nginx.conf")
	output := fmt.Sprintf("42 1 1000 Mon Nov 9 12:00:00 2026 nginx: master process openresty -c %s\n43 42 1000 Mon Nov 9 12:00:00 2026 nginx: worker process\n50 1 1000 Mon Nov 9 12:00:00 2026 nginx: master process openresty -c %s", configPath, otherConfigPath)
	processes, err := parsePSRuntime(output, "openresty", configPath, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(processes) != 3 || !processes[0].Master || processes[2].Master {
		t.Errorf("parsePSRuntime=%v, want only master 42 managed", processes)
	}
	if got := runtimeDescendants(processes, runtimeMasterIdentities(processes)); len(got) != 2 {
		t.Errorf("managed descendants=%v, want 42 and 43", got)
	}
}
