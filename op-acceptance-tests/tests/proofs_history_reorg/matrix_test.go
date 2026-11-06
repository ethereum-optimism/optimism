package proofs_history_reorg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlocksInRethMemory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		threshold uint64
		height    uint64
		want      uint64
	}{
		{name: "nothing persisted below the first run", threshold: 7, height: 7, want: 7},
		{name: "first run keeps the buffer target", threshold: 7, height: 8, want: 5},
		{name: "grows by one per block", threshold: 7, height: 10, want: 7},
		{name: "next run three blocks later at threshold 7", threshold: 7, height: 11, want: 5},
		{name: "nothing persisted below the first run at threshold 50", threshold: 50, height: 50, want: 50},
		{name: "first run at threshold 50", threshold: 50, height: 51, want: 5},
		{name: "just before the second run at threshold 50", threshold: 50, height: 96, want: 50},
		{name: "second run at threshold 50", threshold: 50, height: 97, want: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := reorgCase{persistenceThreshold: tc.threshold}
			require.Equal(t, tc.want, c.blocksInRethMemory(tc.height))
		})
	}
}

func TestInMemoryDepthsFitRethMemoryWithinOnePersistenceCycle(t *testing.T) {
	for _, c := range reorgMatrix() {
		if c.kind != inMemoryReorg {
			continue
		}
		t.Run(c.name(), func(t *testing.T) {
			for forkPoint := uint64(1); forkPoint < 200; forkPoint++ {
				padding := uint64(0)
				for !c.forkPointInMemory(forkPoint + padding + c.depth) {
					padding++
				}
				require.Less(t, padding, c.persistenceThreshold, "fork point %d", forkPoint)
			}
		})
	}
}

func TestOnDiskDepthsExceedRethMemory(t *testing.T) {
	for _, c := range reorgMatrix() {
		if c.kind != onDiskReorg {
			continue
		}
		t.Run(c.name(), func(t *testing.T) {
			for h := c.persistenceThreshold + 1; h < 400; h++ {
				require.False(t, c.forkPointInMemory(h), "height %d", h)
			}
		})
	}
}
