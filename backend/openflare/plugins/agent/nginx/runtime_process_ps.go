//go:build !linux

package nginx

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	psCommandIndex      = 8
	psMasterBinaryIndex = 11
)

func parsePSRuntime(output, binary, configPath string, owner int) ([]runtimeProcess, error) {
	var processes []runtimeProcess
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) <= psCommandIndex {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		title := strings.Join(fields[psCommandIndex:], " ")
		if !strings.HasPrefix(title, "nginx: ") && !strings.HasPrefix(title, "openresty: ") {
			continue
		}
		master := uid == owner && matchesRuntimeMaster(title, configPath)
		if master && (len(fields) <= psMasterBinaryIndex || fields[psMasterBinaryIndex] != binary) {
			return nil, fmt.Errorf("cannot verify openresty master invocation %s", fields[0])
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, err
		}
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		processes = append(processes, runtimeProcess{PID: pid, ParentPID: parentPID, Master: master, StartTime: strings.Join(fields[3:psCommandIndex], " ")})
	}
	return processes, nil
}
