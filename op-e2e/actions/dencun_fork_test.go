package actions

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestDencunL1Fork(gt *testing.T) {
	// t := NewDefaultTesting(gt)
	// dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)

	// sd := e2eutils.Setup(t, dp, defaultAlloc)
	// activation := sd.L1Cfg.Timestamp + 24
	// sd.L1Cfg.Config.CancunTime = &activation
	// log := testlog.Logger(t, log.LvlDebug)

	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	activation := sd.L1Cfg.Timestamp + 24
	sd.L1Cfg.Config.CancunTime = &activation
	log := testlog.Logger(t, log.LvlDebug)
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	l1Head := miner.l1Chain.CurrentBlock()
	require.False(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun not active yet")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks, crossing the fork boundary
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t) //// test fails here as cancun activates
	/*
		########## BAD BLOCK #########
		Block: 2 (0xf227de28aae80d15e8a50c5639a9b31be65fdefe97e21eadd23b957bd3d6ee5a)
		Error: header is missing beaconRoot
		Platform: geth 0.1.0-unstable go1.21.3 arm64 darwin
		Chain config: &params.ChainConfig{ChainID:900, HomesteadBlock:0, DAOForkBlock:<nil>, DAOForkSupport:false, EIP150Block:0, EIP155Block:0, EIP158Block:0, ByzantiumBlock:0, ConstantinopleBlock:0, PetersburgBlock:0, IstanbulBlock:0, MuirGlacierBlock:0, BerlinBlock:0, LondonBlock:0, ArrowGlacierBlock:0, GrayGlacierBlock:0, MergeNetsplitBlock:0, ShanghaiTime:(*uint64)(0x1400038f4a0), CancunTime:(*uint64)(0x14000364f98), PragueTime:(*uint64)(nil), VerkleTime:(*uint64)(nil), BedrockBlock:<nil>, RegolithTime:(*uint64)(nil), CanyonTime:(*uint64)(nil), TerminalTotalDifficulty:0, TerminalTotalDifficultyPassed:true, Ethash:(*params.EthashConfig)(nil), Clique:(*params.CliqueConfig)(nil), IsDevMode:false, Optimism:(*params.OptimismConfig)(nil)}
		Receipts:
		##############################
	*/
	miner.ActEmptyBlock(t)
	// verify Cancun is active
	l1Head = miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active")

	//BEFORE MERGE OF PR #7993: Also, add a few blob txs in as dummy data TODO

	// build L2 chain up to and including L2 blocks referencing Cancun L1 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	// verify verifier accepted Cancun L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes Cancun headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}
