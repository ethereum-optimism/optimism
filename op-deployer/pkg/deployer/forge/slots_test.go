package forge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlotCount(t *testing.T) {
	for _, tt := range []struct {
		cpus     int
		memLimit uint64
		want     int
	}{
		{cpus: 4, memLimit: 12 * forgeRunMemory, want: 9},
		{cpus: 64, memLimit: 4*forgeRunMemory + forgeRunMemory/2, want: 3},
		{cpus: 16, memLimit: forgeRunMemory / 2, want: 1},
		{cpus: 12, want: 12},
		{cpus: 0, want: 1},
	} {
		require.Equal(t, tt.want, slotCount(tt.cpus, tt.memLimit), "cpus %d, memory limit %d", tt.cpus, tt.memLimit)
	}
}
