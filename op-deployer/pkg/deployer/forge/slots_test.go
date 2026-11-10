package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotCount(t *testing.T) {
	const gib = 1 << 30
	for _, tt := range []struct {
		cpus     int
		memLimit uint64
		want     int
	}{
		{cpus: 16, want: 8},
		{cpus: 16, memLimit: 8 * gib, want: 5},
		{cpus: 4, memLimit: 64 * gib, want: 2},
		{cpus: 16, memLimit: gib, want: 1},
		{cpus: 1, want: 1},
	} {
		require.Equal(t, tt.want, slotCount(tt.cpus, tt.memLimit), "cpus %d, memory limit %d", tt.cpus, tt.memLimit)
	}
}
