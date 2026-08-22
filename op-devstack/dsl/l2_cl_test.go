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
