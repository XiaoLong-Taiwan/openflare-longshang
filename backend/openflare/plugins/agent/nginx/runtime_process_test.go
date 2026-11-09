package nginx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReloadRepairsInvalidPIDWithoutStarting(t *testing.T) {
	for _, value := range []string{"", "invalid", "999999\n"} {
		t.Run(value, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "nginx.conf")
			pidPath := filepath.Join(dir, "nginx.pid")
			if err := os.WriteFile(config, []byte("pid \""+filepath.ToSlash(pidPath)+"\";\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(pidPath, []byte(value), 0o600); err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{}
			master := runtimeProcess{PID: 42, Master: true, StartTime: "123"}
			signalled := false
			executor := &PathExecutor{ConfigPath: config, Runner: runner,
				inspectProcesses: func() ([]runtimeProcess, error) { return []runtimeProcess{master}, nil },
				signalProcess:    func(got runtimeProcess, quit bool) error { signalled = got == master && !quit; return nil },
			}
			if err := executor.Reload(context.Background()); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(pidPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "42\n" || !signalled || len(runner.calls) != 0 {
				t.Errorf("Reload(%q) pid=%q signalled=%t commands=%v, want 42, reload signal, no start", value, data, signalled, runner.calls)
			}
		})
	}
}

func TestRecoveryRefusesAmbiguousOrUninspectableRuntime(t *testing.T) {
	for _, processes := range [][]runtimeProcess{nil, {{PID: 42, Master: true}, {PID: 43, Master: true}}} {
		runner := &fakeRunner{}
		executor := &PathExecutor{Runner: runner, inspectProcesses: func() ([]runtimeProcess, error) {
			if processes == nil {
				return nil, errors.New("permission denied")
			}
			return processes, nil
		}}
		if err := executor.Reload(context.Background()); err == nil || len(runner.calls) != 0 {
			t.Errorf("Reload(%v) error=%v commands=%v, want refusal without start", processes, err, runner.calls)
		}
	}
}

func TestRestartWaitsForOwnedWorkersAndIgnoresUnrelated(t *testing.T) {
	inspections := 0
	runner := &fakeRunner{}
	executor := &PathExecutor{Runner: runner,
		inspectProcesses: func() ([]runtimeProcess, error) {
			inspections++
			switch inspections {
			case 1:
				return []runtimeProcess{{PID: 42, Master: true, StartTime: "a"}, {PID: 43, ParentPID: 42, StartTime: "b"}, {PID: 50}}, nil
			case 2:
				return []runtimeProcess{{PID: 43, ParentPID: 1, StartTime: "b"}, {PID: 50}}, nil
			default:
				return []runtimeProcess{{PID: 50}}, nil
			}
		},
		signalProcess: func(process runtimeProcess, quit bool) error {
			if process.PID != 42 || !quit {
				t.Errorf("Restart signal=%v quit=%t, want master 42 quit", process, quit)
			}
			return nil
		},
	}
	if err := executor.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if inspections != 3 || len(runner.calls) != 1 {
		t.Errorf("Restart inspections=%d commands=%v, want 3 and one start", inspections, runner.calls)
	}
}

func TestRestartDeadlineNeverStartsSecondMaster(t *testing.T) {
	runner := &fakeRunner{}
	executor := &PathExecutor{Runner: runner,
		inspectProcesses: func() ([]runtimeProcess, error) { return []runtimeProcess{{PID: 42, Master: true}}, nil },
		signalProcess:    func(runtimeProcess, bool) error { return nil },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := executor.Restart(ctx); !errors.Is(err, context.DeadlineExceeded) || len(runner.calls) != 0 {
		t.Errorf("Restart(deadline) error=%v commands=%v, want deadline and no start", err, runner.calls)
	}
}

func TestRuntimeDescendantsRejectsReusedPID(t *testing.T) {
	identities := map[int]string{42: "original", 43: "worker"}
	got := runtimeDescendants([]runtimeProcess{{PID: 42, StartTime: "reused"}, {PID: 43, ParentPID: 1, StartTime: "worker"}, {PID: 50}}, identities)
	if len(got) != 1 || got[0].PID != 43 {
		t.Errorf("runtimeDescendants(reused PID)=%v, want worker 43", got)
	}
}
