package proofs_test

import (
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_PragueForkAfterGenesis(gt *testing.T) {
	type testCase struct {
		name         string
		useSetCodeTx bool
	}

	dynamiceFeeCase := testCase{
		name: "dynamicFeeTx", useSetCodeTx: false,
	}
	setCodeCase := testCase{
		name: "setCodeTx", useSetCodeTx: true,
	}

	runL1PragueTest := func(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(),
			helpers.NewBatcherCfg(
				func(c *actionsHelpers.BatcherCfg) {
					c.DataAvailabilityType = batcherFlags.CalldataType
				},
			),
			func(dp *genesis.DeployConfig) {
				dp.L1PragueTimeOffset = ptr(hexutil.Uint64(24)) // Activate at second l1 block
			},
		)

		miner, batcher, verifier, sequencer, engine := env.Miner, env.Batcher, env.Sequencer, env.Sequencer, env.Engine

		l1Block, err := legacybindings.NewL1Block(predeploys.L1BlockAddr, engine.EthClient())
		require.NoError(t, err)

		// utils
		checkVerifierDerivedToL1Head := func(t actionsHelpers.StatefulTesting) {
			l1Head := miner.L1Chain().CurrentBlock()
			currentL1 := verifier.SyncStatus().CurrentL1
			require.Equal(t, l1Head.Number.Int64(), int64(currentL1.Number), "verifier should derive up to and including the L1 head")
			require.Equal(t, l1Head.Hash(), currentL1.Hash, "verifier should derive up to and including the L1 head")
		}

		buildUnsafeL2AndSubmit := func(useSetCode bool) {
			sequencer.ActL1HeadSignal(t)
			sequencer.ActBuildToL1Head(t)

			miner.ActL1StartBlock(12)(t)
			if useSetCode {
				batcher.ActBufferAll(t)
				batcher.ActL2ChannelClose(t)
				batcher.ActSubmitSetCodeTx(t)
			} else {
				batcher.ActSubmitAll(t)
			}
			miner.ActL1IncludeTx(batcher.BatcherAddr)(t)
			miner.ActL1EndBlock(t)
		}

		requirePragueStatusOnL1 := func(active bool, block *types.Header) {
			if active {
				require.True(t, env.Sd.L1Cfg.Config.IsPrague(block.Number, block.Time), "Prague should be active at block", block.Number.Uint64())
				require.NotNil(t, block.RequestsHash, "Prague header requests hash should be non-nil")
			} else {
				require.False(t, env.Sd.L1Cfg.Config.IsPrague(block.Number, block.Time), "Prague should not be active yet at block", block.Number.Uint64())
				require.Nil(t, block.RequestsHash, "Prague header requests hash should be nil")
			}
		}

		syncVerifierAndCheck := func(t actionsHelpers.StatefulTesting) {
			verifier.ActL1HeadSignal(t)
			verifier.ActL2PipelineFull(t)
			checkVerifierDerivedToL1Head(t)
		}

		checkL1BlockBlobBaseFee := func(t actionsHelpers.StatefulTesting, l2Block eth.L2BlockRef) {
			l1BlockID := l2Block.L1Origin
			l1BlockHeader := miner.L1Chain().GetHeaderByHash(l1BlockID.Hash)
			expectedBbf := eth.CalcBlobFeeDefault(l1BlockHeader)
			upstreamExpectedBbf := eip4844.CalcBlobFee(env.Sd.L1Cfg.Config, l1BlockHeader)
			require.Equal(t, expectedBbf.Uint64(), upstreamExpectedBbf.Uint64(), "expected blob base fee should match upstream calculation")
			bbf, err := l1Block.BlobBaseFee(&bind.CallOpts{BlockHash: l2Block.Hash})
			require.NoError(t, err, "failed to get blob base fee")
			require.Equal(t, expectedBbf.Uint64(), bbf.Uint64(), "l1Block blob base fee does not match expectation, l1BlockNum %d, l2BlockNum %d", l1BlockID.Number, l2Block.Number)
		}

		requireSafeHeadProgression := func(t actionsHelpers.StatefulTesting, safeL2Before, safeL2After eth.L2BlockRef, batchedWithSetCodeTx bool) {
			if batchedWithSetCodeTx {
				require.Equal(t, safeL2Before, safeL2After, "safe head should not have changed (SetCode / type 4 batcher tx ignored)")
				require.Equal(t, safeL2Before.L1Origin.Number, safeL2After.Number, "l1 origin of l2 safe should not have changed (SetCode / type 4 batcher tx ignored)")
			} else {
				require.Greater(t, safeL2After.Number, safeL2Before.Number, "safe head should have progressed (DynamicFee / type 2 batcher tx derived from)")
				require.Equal(t, verifier.SyncStatus().UnsafeL2.Number, safeL2After.Number, "safe head should equal unsafe head (DynamicFee / type 2 batcher tx derived from)")
				require.Greater(t, safeL2After.L1Origin.Number, safeL2Before.L1Origin.Number, "l1 origin of l2 safe should have progressed (DynamicFee / type 2 batcher tx tx derived from)")
			}
		}

		// Check initially Prague is not activated
		requirePragueStatusOnL1(false, miner.L1Chain().CurrentBlock())

		// Start op-nodes
		sequencer.ActL2PipelineFull(t)
		verifier.ActL2PipelineFull(t)

		// Build L1 blocks, crossing the fork boundary
		miner.ActEmptyBlock(t) // block 1
		miner.ActEmptyBlock(t) // Prague activates here (block 2)

		// Here's a block with a type 4 deposit transaction, sent to the OptimismPortal
		miner.ActL1StartBlock(12)(t) // block 3
		tx, err := actionsHelpers.PrepareSignedSetCodeTx(
			*uint256.MustFromBig(env.Sd.L1Cfg.Config.ChainID),
			env.Dp.Secrets.Alice,
			env.Alice.L1.Signer(),
			env.Alice.L1.PendingNonce(t), // nonce
			env.Sd.DeploymentsL1.OptimismPortalProxy,
			[]byte{})
		require.NoError(t, err, "failed to prepare set code tx")
		err = miner.EthClient().SendTransaction(t.Ctx(), tx)
		require.NoError(t, err, "failed to send set code tx")
		miner.ActL1IncludeTx(env.Alice.Address())(t)
		miner.ActL1EndBlock(t)

		// Check that Prague is active on L1
		requirePragueStatusOnL1(true, miner.L1Chain().CurrentBlock())

		// Cache safe head before verifier sync
		safeL2Initial := verifier.SyncStatus().SafeL2

		// Build an empty L2 block which has a pre-prague L1 origin, and check the blob fee is correct
		sequencer.ActL2EmptyBlock(t)
		l1OriginHeader := miner.L1Chain().GetHeaderByHash(verifier.SyncStatus().UnsafeL2.L1Origin.Hash)
		requirePragueStatusOnL1(false, l1OriginHeader)
		checkL1BlockBlobBaseFee(t, verifier.SyncStatus().UnsafeL2)

		// Build L2 unsafe chain and batch it to L1 using either DynamicFee or
		// EIP-7702 SetCode txs
		// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-7702.md
		buildUnsafeL2AndSubmit(testCfg.Custom.useSetCodeTx)

		// Check verifier derived from Prague L1 blocks
		syncVerifierAndCheck(t)

		// Check safe head did or did not change,
		// depending on tx type used by batcher:
		safeL2AfterFirstBatch := verifier.SyncStatus().SafeL2
		requireSafeHeadProgression(t, safeL2Initial, safeL2AfterFirstBatch, testCfg.Custom.useSetCodeTx)

		sequencer.ActBuildToL1Head(t) // Advance L2 chain until L1 origin has Prague active

		// Check that the l1 origin is now a Prague block, and that the blob fee is correct
		l1Origin := miner.L1Chain().GetHeaderByNumber(verifier.SyncStatus().UnsafeL2.L1Origin.Number)
		requirePragueStatusOnL1(true, l1Origin)
		checkL1BlockBlobBaseFee(t, verifier.SyncStatus().UnsafeL2)

		// Batch and sync again
		buildUnsafeL2AndSubmit(testCfg.Custom.useSetCodeTx)
		syncVerifierAndCheck(t)
		safeL2AfterSecondBatch := verifier.SyncStatus().SafeL2
		requireSafeHeadProgression(t, safeL2AfterFirstBatch, safeL2AfterSecondBatch, testCfg.Custom.useSetCodeTx)

		env.RunFaultProofProgram(t, safeL2AfterSecondBatch.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[testCase]()
	defer matrix.Run(gt)
	matrix.
		AddDefaultTestCasesWithName(dynamiceFeeCase.name, dynamiceFeeCase, helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork), runL1PragueTest).
		AddDefaultTestCasesWithName(setCodeCase.name, setCodeCase, helpers.NewForkMatrix(helpers.Holocene, helpers.LatestFork), runL1PragueTest)
}
