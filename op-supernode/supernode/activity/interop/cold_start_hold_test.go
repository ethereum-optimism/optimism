package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestEmptyVerifiedDB_ColdStartHold covers the empty-verifiedDB contract of
// LatestVerifiedL2Block and VerifiedBlockAtL1 (issue #22127 follow-up):
// an empty DB may only be reported as the pre-activation anchor cap on a
// genuine activation bootstrap. While cold-start init is incomplete, or when
// the cold-start origin is past activation (a resume with a history gap, e.g.
// a restart that lost the datadir), the accessors must return ErrNotStarted so
// the super authority holds the previous heads instead of publishing the
// anchor — which would regress the EL's safe/finalized labels and force a full
// re-derivation on the next engine reset.
func TestEmptyVerifiedDB_ColdStartHold(t *testing.T) {
	l1Block := eth.L1BlockRef{Hash: common.Hash{0x01}, Number: 1000, Time: 999}

	requireHold := func(t *testing.T, i *Interop, chainID eth.ChainID) {
		t.Helper()
		blockID, ts, err := i.LatestVerifiedL2Block(chainID)
		require.ErrorIs(t, err, ErrNotStarted, "LatestVerifiedL2Block must hold")
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, uint64(0), ts)

		blockID, ts, err = i.VerifiedBlockAtL1(chainID, l1Block)
		require.ErrorIs(t, err, ErrNotStarted, "VerifiedBlockAtL1 must hold")
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, uint64(0), ts)
	}

	t.Run("cold start incomplete holds", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()
		h.interop.backfillCompleted.Store(false)

		requireHold(t, h.interop, h.Mock(10).id)
	})

	t.Run("origin past activation holds", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()
		require.Greater(t, h.interop.verificationStartTimestamp, h.interop.activationTimestamp,
			"harness models a cold start whose origin is past activation")

		requireHold(t, h.interop, h.Mock(10).id)
	})

	t.Run("activation bootstrap returns the anchor cap", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()
		h.interop.verificationStartTimestamp = h.interop.activationTimestamp
		chainID := h.Mock(10).id

		blockID, ts, err := h.interop.LatestVerifiedL2Block(chainID)
		require.NoError(t, err)
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, h.interop.activationTimestamp-1, ts)

		blockID, ts, err = h.interop.VerifiedBlockAtL1(chainID, l1Block)
		require.NoError(t, err)
		require.Equal(t, eth.BlockID{}, blockID)
		require.Equal(t, h.interop.activationTimestamp-1, ts)
	})

	t.Run("hold clears after first commit", func(t *testing.T) {
		h := newInteropTestHarness(t).
			WithChain(10, nil).
			Build()
		chainID := h.Mock(10).id
		requireHold(t, h.interop, chainID)

		head := eth.BlockID{Hash: common.Hash{0xaa}, Number: 2000}
		require.NoError(t, h.commitVerified(VerifiedResult{
			Timestamp:   h.interop.verificationStartTimestamp,
			L1Inclusion: eth.BlockID{Number: 900},
			L2Heads:     map[eth.ChainID]eth.BlockID{chainID: head},
		}))

		blockID, ts, err := h.interop.LatestVerifiedL2Block(chainID)
		require.NoError(t, err)
		require.Equal(t, head, blockID)
		require.Equal(t, h.interop.verificationStartTimestamp, ts)

		blockID, ts, err = h.interop.VerifiedBlockAtL1(chainID, l1Block)
		require.NoError(t, err)
		require.Equal(t, head, blockID)
		require.Equal(t, h.interop.verificationStartTimestamp, ts)
	})
}
