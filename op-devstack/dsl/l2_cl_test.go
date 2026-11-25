package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestSyncStatusHead(t *testing.T) {
	syncStatus := &eth.SyncStatus{
		UnsafeL2:    eth.L2BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 30},
		SafeL2:      eth.L2BlockRef{Number: 20},
		FinalizedL2: eth.L2BlockRef{Number: 10},
	}

	for _, test := range []struct {
		lvl  safety.Level
		want eth.L2BlockRef
	}{
		{safety.LocalUnsafe, syncStatus.UnsafeL2},
		{safety.LocalSafe, syncStatus.LocalSafeL2},
		{safety.CrossSafe, syncStatus.SafeL2},
		{safety.Finalized, syncStatus.FinalizedL2},
	} {
		t.Run(string(test.lvl), func(t *testing.T) {
			got, err := syncStatusHead(syncStatus, test.lvl)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}

	// CrossUnsafe is a message-safety level only; there is no cross-unsafe chain head.
	t.Run("cross-unsafe", func(t *testing.T) {
		_, err := syncStatusHead(syncStatus, safety.CrossUnsafe)
		require.ErrorIs(t, err, errNoChainHeadForLevel)
	})
}

func TestSyncStatusRegression(t *testing.T) {
	prev := &eth.SyncStatus{
		CurrentL1:   eth.L1BlockRef{Number: 20},
		UnsafeL2:    eth.L2BlockRef{Number: 40},
		LocalSafeL2: eth.L2BlockRef{Number: 30},
	}

	t.Run("unchanged", func(t *testing.T) {
		require.NoError(t, syncStatusRegression(prev, prev))
	})
	t.Run("advanced", func(t *testing.T) {
		cur := *prev
		cur.CurrentL1.Number++
		cur.UnsafeL2.Number++
		cur.LocalSafeL2.Number++
		require.NoError(t, syncStatusRegression(prev, &cur))
	})
	// A reset zeroes the sync status before derivation restarts from an older L1 block.
	t.Run("reset", func(t *testing.T) {
		require.Error(t, syncStatusRegression(prev, &eth.SyncStatus{}))
	})
	for name, regress := range map[string]func(*eth.SyncStatus){
		"current-l1": func(s *eth.SyncStatus) { s.CurrentL1.Number-- },
		"unsafe":     func(s *eth.SyncStatus) { s.UnsafeL2.Number-- },
		"local-safe": func(s *eth.SyncStatus) { s.LocalSafeL2.Number-- },
	} {
		t.Run(name, func(t *testing.T) {
			cur := *prev
			regress(&cur)
			require.Error(t, syncStatusRegression(prev, &cur))
		})
	}
}
