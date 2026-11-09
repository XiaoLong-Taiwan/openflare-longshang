package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectProcRuntimeVerifiesOwnerConfigAndExecutable(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "openresty")
	config := filepath.Join(root, "nginx.conf")
	if err := os.WriteFile(binary, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	writeProcFixture(t, root, 42, 1, 1000, "S", "100", "nginx: master process "+binary+" -c "+config, binary)
	writeProcFixture(t, root, 43, 42, 1000, "S", "101", "nginx: worker process", binary)
	writeProcFixture(t, root, 50, 1, 1000, "S", "102", "nginx: master process "+binary+" -c "+filepath.Join(root, "other.conf"), binary)
	writeProcFixture(t, root, 60, 1, 2000, "S", "103", "nginx: master process "+binary+" -c "+config, binary)
	writeProcFixture(t, root, 70, 42, 1000, "Z", "104", "nginx: worker process", binary)
	processes, err := inspectProcRuntime(root, executable, binary, config, 1000)
	if err != nil {
		t.Fatal(err)
	}
	master, err := findRuntimeMaster(processes)
	if err != nil || master.PID != 42 || master.StartTime != "100" {
		t.Errorf("findRuntimeMaster(%v) = %v, %v, want master 42 start 100", processes, master, err)
	}
	managed := runtimeDescendants(processes, runtimeMasterIdentities(processes))
	if len(managed) != 2 {
		t.Errorf("runtimeDescendants(%v) = %v, want master 42 and worker 43", processes, managed)
	}
}

func TestInspectProcRuntimeRejectsChangedMasterExecutable(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "openresty")
	other := filepath.Join(root, "other")
	for _, path := range []string{binary, other} {
		if err := os.WriteFile(path, []byte(path), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "nginx.conf")
	writeProcFixture(t, root, 42, 1, 1000, "S", "100", "nginx: master process "+binary+" -c "+config, other)
	if processes, err := inspectProcRuntime(root, executable, binary, config, 1000); err == nil {
		t.Errorf("inspectProcRuntime(changed executable) = %v, nil, want refusal", processes)
	}
}

func TestInspectProcRuntimeRejectsEscapingStatusSymlink(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "openresty")
	if err := os.WriteFile(binary, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "nginx.conf")
	writeProcFixture(t, root, 42, 1, 1000, "S", "100", "nginx: master process "+binary+" -c "+config, binary)
	status := filepath.Join(root, "42", "status")
	if err := os.Rename(status, filepath.Join(root, "outside-status")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "outside-status"), status); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	if processes, err := inspectProcRuntime(root, executable, binary, config, 1000); err == nil {
		t.Errorf("inspectProcRuntime(escaping status symlink) = %v, nil, want refusal", processes)
	}
}

func writeProcFixture(t *testing.T, root string, pid, parentPID, uid int, state, start, command, binary string) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprint(pid))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	fields := make([]string, 20)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0], fields[1], fields[19] = state, fmt.Sprint(parentPID), start
	for name, value := range map[string]string{
		"status":  fmt.Sprintf("Uid:\t%d\t%d\t%d\t%d\n", uid, uid, uid, uid),
		"cmdline": command + "\x00",
		"stat":    fmt.Sprintf("%d (nginx) %s", pid, strings.Join(fields, " ")),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Link(binary, filepath.Join(dir, "exe")); err != nil {
		t.Fatal(err)
	}
}
