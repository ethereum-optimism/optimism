package manual

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
)

func TestVerifierManualSync(gt *testing.T) {
	t := devtest.SerialT(gt)

	// Disable ELP2P and Batcher
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	logger := t.Logger()

	delta := uint64(7)
	sys.L2CL.Advanced(types.LocalUnsafe, delta, 30)

	// Disable Derivation
	sys.L2CLB.Stop()

	startBlockNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number

	// Manual Block insertion using engine APIs
	for i := uint64(1); i <= delta; i++ {
		blockNum := startBlockNum + i
		block := sys.L2EL.BlockRefByNumber(blockNum)
		// Validator does not have canonical nor noncanonical block for blockNum
		_, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(t.Ctx(), blockNum)
		require.Error(err, ethereum.NotFound)
		_, err = sys.L2ELB.Escape().EthClient().BlockRefByHash(t.Ctx(), block.Hash)
		require.Error(err, ethereum.NotFound)

		// Insert payload
		logger.Info("NewPayload", "target", blockNum)
		sys.L2ELB.NewPayload(sys.L2EL, blockNum).IsValid()
		// Payload valid but not canonicalized. Cannot fetch block by number
		_, err = sys.L2ELB.Escape().EthClient().BlockRefByNumber(t.Ctx(), blockNum)
		require.Error(err, ethereum.NotFound)
		// Now fetchable by hash
		require.Equal(blockNum, sys.L2ELB.BlockRefByHash(block.Hash).Number)

		// FCU
		logger.Info("ForkchoiceUpdate", "target", blockNum)
		sys.L2ELB.ForkchoiceUpdate(sys.L2EL, blockNum, 0, 0, nil).IsValid()
		// Payload valid and canonicalized
		require.Equal(block.Hash, sys.L2ELB.BlockRefByNumber(blockNum).Hash)
		require.Equal(blockNum, sys.L2ELB.BlockRefByHash(block.Hash).Number)
	}

	// Check correctly synced by comparing with sequencer EL
	res := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	require.Equal(startBlockNum+delta, res.Number)
	require.Equal(sys.L2EL.BlockRefByNumber(startBlockNum+delta).Hash, res.Hash)
}
