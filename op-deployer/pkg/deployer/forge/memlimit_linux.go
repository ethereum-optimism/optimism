//go:build linux

package forge

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
)

// memoryLimit returns the memory of the process: the smaller of the host's memory and the limit of
// the cgroup at /sys/fs/cgroup, the container's cgroup in a container. It returns 0 when it knows
// neither.
func memoryLimit() uint64 {
	meminfo, _ := os.ReadFile("/proc/meminfo")
	var cgroup []byte
	for _, file := range []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"} {
		if data, err := os.ReadFile(file); err == nil {
			cgroup = data
			break
		}
	}
	return smallerLimit(parseMemTotal(meminfo), parseCgroupLimit(cgroup))
}

// smallerLimit returns the smaller of two memory limits, where 0 means no limit.
func smallerLimit(a, b uint64) uint64 {
	if a == 0 || (b != 0 && b < a) {
		return b
	}
	return a
}

// parseMemTotal returns the MemTotal of /proc/meminfo in bytes, or 0.
func parseMemTotal(meminfo []byte) uint64 {
	scanner := bufio.NewScanner(bytes.NewReader(meminfo))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 && fields[0] == "MemTotal:" && fields[2] == "kB" {
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0
			}
			return kb << 10
		}
	}
	return 0
}

// parseCgroupLimit returns the memory limit of a cgroup's limit file, or 0 for no limit.
func parseCgroupLimit(data []byte) uint64 {
	// cgroup v2 writes "max", and cgroup v1 a value near 2^63, for no limit.
	limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || limit >= 1<<62 {
		return 0
	}
	return limit
}
