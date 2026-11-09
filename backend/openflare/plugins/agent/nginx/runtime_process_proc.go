package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const procParentPIDIndex = 1
const procStartTimeIndex = 19

func inspectProcRuntime(root string, executable os.FileInfo, binary, configPath string, uid int) ([]runtimeProcess, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var processes []runtimeProcess
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || !entry.IsDir() {
			continue
		}
		process, found, err := inspectProcProcess(filepath.Join(root, entry.Name()), pid, executable, binary, configPath, uid)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if found {
			processes = append(processes, process)
		}
	}
	return processes, nil
}

func inspectProcProcess(dir string, pid int, executable os.FileInfo, binary, configPath string, uid int) (runtimeProcess, bool, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return runtimeProcess{}, false, err
	}
	defer func() { _ = root.Close() }()
	status, err := root.ReadFile("status")
	if err != nil {
		return runtimeProcess{}, false, err
	}
	owned := effectiveProcessUID(string(status)) == uid
	command, err := root.ReadFile("cmdline")
	if err != nil {
		return runtimeProcess{}, false, err
	}
	title := strings.ReplaceAll(string(command), "\x00", " ")
	if !strings.HasPrefix(title, "nginx: ") && !strings.HasPrefix(title, "openresty: ") {
		return runtimeProcess{}, false, nil
	}
	master := owned && matchesRuntimeMaster(title, configPath)
	exe, err := os.Stat(filepath.Join(dir, "exe"))
	if err != nil {
		if !os.IsPermission(err) {
			return runtimeProcess{}, false, err
		}
		fields := strings.Fields(title)
		if master && (len(fields) < 4 || fields[3] != binary) {
			return runtimeProcess{}, false, fmt.Errorf("cannot verify executable of openresty master %d", pid)
		}
	} else if !os.SameFile(executable, exe) {
		if master {
			return runtimeProcess{}, false, fmt.Errorf("executable changed for openresty master %d; refusing duplicate start", pid)
		}
		return runtimeProcess{}, false, nil
	}
	stat, err := root.ReadFile("stat")
	if err != nil {
		return runtimeProcess{}, false, err
	}
	closeParen := strings.LastIndexByte(string(stat), ')')
	if closeParen < 0 {
		return runtimeProcess{}, false, fmt.Errorf("invalid stat for process %d", pid)
	}
	fields := strings.Fields(string(stat)[closeParen+1:])
	if len(fields) <= procStartTimeIndex {
		return runtimeProcess{}, false, fmt.Errorf("incomplete stat for process %d", pid)
	}
	if fields[0] == "Z" {
		return runtimeProcess{}, false, nil
	}
	parentPID, err := strconv.Atoi(fields[procParentPIDIndex])
	if err != nil {
		return runtimeProcess{}, false, fmt.Errorf("invalid parent pid for process %d: %w", pid, err)
	}
	return runtimeProcess{PID: pid, ParentPID: parentPID, Master: master, StartTime: fields[procStartTimeIndex]}, true, nil
}

func effectiveProcessUID(status string) int {
	for _, line := range strings.Split(status, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "Uid:" {
			uid, err := strconv.Atoi(fields[2])
			if err == nil {
				return uid
			}
		}
	}
	return -1
}
