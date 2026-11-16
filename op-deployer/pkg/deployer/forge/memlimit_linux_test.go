//go:build linux

package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmallerLimit(t *testing.T) {
	const meminfo = "MemTotal:       16303492 kB\nMemFree:         1000000 kB\n"
	const host = 16303492 << 10
	for _, tt := range []struct {
		name    string
		meminfo string
		cgroup  string
		want    uint64
	}{
		{name: "no cgroup limit", meminfo: meminfo, cgroup: "max\n", want: host},
		{name: "cgroup limit", meminfo: meminfo, cgroup: "4294967296\n", want: 4 << 30},
		{name: "cgroup limit above the host", meminfo: meminfo, cgroup: "68719476736\n", want: host},
		{name: "cgroup v1 without limit", meminfo: meminfo, cgroup: "9223372036854771712\n", want: host},
		{name: "no cgroup", meminfo: meminfo, want: host},
		{name: "no meminfo", cgroup: "4294967296\n", want: 4 << 30},
		{name: "nothing", want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, smallerLimit(parseMemTotal([]byte(tt.meminfo)), parseCgroupLimit([]byte(tt.cgroup))))
		})
	}
}
