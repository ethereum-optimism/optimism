//go:build linux

package forge

import (
	"os"
	"strconv"
	"strings"
)

// memoryLimit returns the memory limit of the cgroup at /sys/fs/cgroup, the container's cgroup in
// a container, or 0 when it has none.
func memoryLimit() uint64 {
	for _, file := range []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"} {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		// cgroup v2 writes "max", and cgroup v1 a value near 2^63, for no limit.
		limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if err != nil || limit >= 1<<62 {
			return 0
		}
		return limit
	}
	return 0
}
