package manual

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

	// Check safe and finalized is stil at genesis
	safe := sys.L2ELB.SafeHead().IsGenesis()
	finalized := sys.L2ELB.FinalizedHead().IsGenesis()
	currentUnsafe := sys.L2ELB.UnsafeHead()
	logger.Info("Current", "unsafe head num", currentUnsafe.BlockRef.Number)
	rewindDelta := uint64(2)
	rewindTarget := currentUnsafe.BlockRef.Number - rewindDelta
	prevUnsafe := sys.L2ELB.BlockRefByNumber(rewindTarget)

	// Try to rewind unsafe head
	logger.Info("ForkchoiceUpdateRaw", "target", prevUnsafe.ID().Number)
	sys.L2ELB.ForkchoiceUpdateRaw(prevUnsafe.Hash, safe.BlockRef.Hash, finalized.BlockRef.Hash, nil).IsValid()
	// As expected, did not rewind
	require.Equal(sys.L2ELB.UnsafeHead().BlockRef.Hash, currentUnsafe.BlockRef.Hash)

	// Lets trigger a manual rewind using reorg
	// We first create a altered payload
	payload := sys.L2ELB.PayloadByNumber(rewindTarget)
	// Inject diff and patch hash. To get VALID from engine_newPayload
	payload.ExecutionPayload.FeeRecipient = common.MaxAddress
	actual, ok := payload.CheckBlockHash()
	require.False(ok)
	payload.ExecutionPayload.BlockHash = actual
	_, ok = payload.CheckBlockHash()
	require.True(ok)
	// Now inject to the EL
	logger.Info("NewPayloadRaw", "target", payload.ID().Number)
	sys.L2ELB.NewPayloadRaw(payload).IsValid()
	// FCU to the forked target
	logger.Info("ForkchoiceUpdateRaw", "target", payload.ID().Number)
	sys.L2ELB.ForkchoiceUpdateRaw(actual, safe.BlockRef.Hash, finalized.BlockRef.Hash, nil).IsValid()
	// Check unsafe reorg is triggered
	newUnsafe := sys.L2ELB.UnsafeHead().BlockRef
	require.Equal(newUnsafe.Hash, actual)
	require.Equal(newUnsafe.Number, rewindTarget)
	logger.Info("Reorg", "targetNum", rewindTarget, "prev", prevUnsafe.Hash, "curr", actual)

	// Now FCU to trigger a rewind
	logger.Info("ForkchoiceUpdate", "target", rewindTarget)
	sys.L2ELB.ForkchoiceUpdate(sys.L2EL, rewindTarget, 0, 0, nil).IsValid()
	// Check unsafe rewind is triggered
	rewindedUnsafe := sys.L2ELB.UnsafeHead().BlockRef
	require.Equal(rewindedUnsafe.Number, prevUnsafe.Number)
	require.Equal(rewindedUnsafe.Hash, prevUnsafe.Hash)
	logger.Info("Rewind", "targetNum", rewindTarget, "curr", prevUnsafe.Hash)
	// Check canonical
	require.Equal(sys.L2EL.BlockRefByNumber(rewindedUnsafe.Number).Hash, rewindedUnsafe.Hash)
}
