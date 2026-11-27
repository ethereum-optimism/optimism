package authority_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
)

// TestKonaNodeDerivesABatchFromTheActionL1 checks the harness: a kona-node and its op-reth derive
// the Go sequencer's blocks from a batch the test mines on the action L1, block for block.
func TestKonaNodeDerivesABatchFromTheActionL1(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	sys := newSystem(t, 1)
	a := sys.chains[0]
	p := a.producer

	sys.miner.ActEmptyBlock(t)
	head := p.buildToL1Head()
	require.NotZero(t, head.Number, "the sequencer built blocks")
	sys.mine(p.batch())
	a.postUnsafe(p.payload(1))

	status := a.awaitLocalSafe(head.Number)
	derived, err := a.rollupClient().OutputAtBlock(t.Ctx(), head.Number)
	require.NoError(t, err)
	require.Equal(t, head.Hash, derived.BlockRef.Hash, "the kona-node derives the sequencer's block")
	t.Logf("derived to %s from L1 %s", status.LocalSafeL2, status.CurrentL1)
}
