package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// The old branch can remain in verifiedDB while reorg processing walks back.
// A replacement L1 branch reaching the same height must not finalize its data.
func TestVerifiedBlockAtL1RejectsReorgedInclusion(t *testing.T) {
	h := newInteropTestHarness(t).WithChain(10, nil).Build()
	chain := h.Mock(10).id
	retained := eth.BlockID{Number: 100, Hash: common.Hash{1}}
	abandoned := eth.BlockID{Number: 101, Hash: common.Hash{2}}
	require.NoError(t, h.commitVerified(VerifiedResult{Timestamp: 1000, L1Inclusion: eth.BlockID{Number: 5, Hash: common.Hash{5}}, L2Heads: map[eth.ChainID]eth.BlockID{chain: retained}}))
	require.NoError(t, h.commitVerified(VerifiedResult{Timestamp: 1001, L1Inclusion: eth.BlockID{Number: 6, Hash: common.Hash{6}}, L2Heads: map[eth.ChainID]eth.BlockID{chain: abandoned}}))
	l1 := &mockL1Source{blocks: map[uint64]eth.L1BlockRef{
		5:  {Number: 5, Hash: common.Hash{5}},
		6:  {Number: 6, Hash: common.Hash{0xf6}},
		10: {Number: 10, Hash: common.Hash{10}},
	}}
	h.interop.l1Checker = newL1ConsistencyChecker(l1)
	head, ts, err := h.interop.VerifiedBlockAtL1(chain, l1.blocks[10])
	require.NoError(t, err)
	require.Equal(t, retained, head, "a higher finalized L1 number does not make an abandoned inclusion canonical")
	require.Equal(t, uint64(1000), ts)
	// The read must not mutate the pending rewind or remove its evidence.
	_, err = h.interop.verifiedDB.Get(1001)
	require.NoError(t, err)
	delete(l1.blocks, 6)
	_, _, err = h.interop.VerifiedBlockAtL1(chain, l1.blocks[10])
	require.Error(t, err, "unavailable L1 ancestry must fail closed")
}

func TestRewindDoesNotSleepForEveryVerifiedTimestamp(t *testing.T) {
	h := newInteropTestHarness(t).WithActivation(100).WithChain(10, func(m *mockChainContainer) {
		m.currentL1 = eth.BlockRef{Number: 1000, Hash: common.Hash{1}}
		m.blockAtTimestamp = eth.L2BlockRef{Number: 500, Hash: common.Hash{2}}
	}).Build()
	h.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID, _ map[eth.ChainID]eth.BlockID, _ *frontierVerificationView) (Result, error) {
		return Result{Timestamp: ts, L1Inclusion: eth.BlockID{Number: 100}, L2Heads: blocks}, nil
	}
	h.interop.cycleVerifyFn = func(uint64, map[eth.ChainID]eth.BlockID, *frontierVerificationView) (Result, error) {
		return Result{}, nil
	}
	for range 2 {
		_, err := h.interop.progressAndRecord()
		require.NoError(t, err)
	}
	h.interop.l1Checker = inconsistentL1Checker{}
	delay, err := h.interop.progress()
	require.NoError(t, err)
	ts, ok := h.interop.verifiedDB.LastTimestamp()
	require.True(t, ok)
	require.Equal(t, uint64(101), ts)
	require.Zero(t, delay, "a successful rewind should continue immediately instead of sleeping once per stale timestamp")
}
