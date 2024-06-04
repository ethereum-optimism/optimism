package actions

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type verifierCfg struct {
	safeHeadListener safeDB
}

type VerifierOpt func(opts *verifierCfg)

func WithSafeHeadListener(l safeDB) VerifierOpt {
	return func(opts *verifierCfg) {
		opts.safeHeadListener = l
	}
}

func defaultVerifierCfg() *verifierCfg {
	return &verifierCfg{
		safeHeadListener: safedb.Disabled,
	}
}

func setupVerifier(t Testing, sd *e2eutils.SetupData, log log.Logger, l1F derive.L1Fetcher, blobSrc derive.L1BlobsFetcher, syncCfg *sync.Config, opts ...VerifierOpt) (*L2Engine, *L2Verifier) {
	cfg := defaultVerifierCfg()
	for _, opt := range opts {
		opt(cfg)
	}
	jwtPath := e2eutils.WriteDefaultJWT(t)
	engine := NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath, EngineWithP2P())
	engCl := engine.EngineClient(t, sd.RollupCfg)
	verifier := NewL2Verifier(t, log, l1F, blobSrc, plasma.Disabled, engCl, sd.RollupCfg, syncCfg, cfg.safeHeadListener)
	return engine, verifier
}

func setupVerifierOnlyTest(t Testing, sd *e2eutils.SetupData, log log.Logger) (*L1Miner, *L2Engine, *L2Verifier) {
	miner := NewL1Miner(t, log, sd.L1Cfg)
	l1Cl := miner.L1Client(t, sd.RollupCfg)
	engine, verifier := setupVerifier(t, sd, log, l1Cl, miner.BlobStore(), &sync.Config{})
	return miner, engine, verifier
}

func TestL2Verifier_SequenceWindow(gt *testing.T) {
	t := NewDefaultTesting(gt)
	p := &e2eutils.TestParams{
		MaxSequencerDrift:   10,
		SequencerWindowSize: 24,
		ChannelTimeout:      10,
		L1BlockTime:         15,
	}
	dp := e2eutils.MakeDeployParams(t, p)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	miner, engine, verifier := setupVerifierOnlyTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})

	// Make two sequence windows worth of empty L1 blocks. After we pass the first sequence window, the L2 chain should get blocks
	for miner.l1Chain.CurrentBlock().Number.Uint64() < sd.RollupCfg.SeqWindowSize*2 {
		miner.ActL1StartBlock(10)(t)
		miner.ActL1EndBlock(t)

		verifier.ActL2PipelineFull(t)

		l1Head := miner.l1Chain.CurrentBlock().Number.Uint64()
		expectedL1Origin := uint64(0)
		// as soon as we complete the sequence window, we force-adopt the L1 origin
		if l1Head >= sd.RollupCfg.SeqWindowSize {
			expectedL1Origin = l1Head - sd.RollupCfg.SeqWindowSize
		}
		require.Equal(t, expectedL1Origin, verifier.SyncStatus().SafeL2.L1Origin.Number, "L1 origin is forced in, given enough L1 blocks pass by")
		require.LessOrEqual(t, miner.l1Chain.GetBlockByNumber(expectedL1Origin).Time(), engine.l2Chain.CurrentBlock().Time, "L2 time higher than L1 origin time")
	}
	tip2N := verifier.SyncStatus()

	// Do a deep L1 reorg as deep as a sequence window, this should affect the safe L2 chain
	miner.ActL1RewindDepth(sd.RollupCfg.SeqWindowSize)(t)

	// Without new L1 block, the L1 appears to not be synced, and the node shouldn't reorg
	verifier.ActL2PipelineFull(t)
	require.Equal(t, tip2N.SafeL2, verifier.SyncStatus().SafeL2, "still the same after verifier work")

	// Make a new empty L1 block with different data than there was before.
	miner.ActL1SetFeeRecipient(common.Address{'B'})
	miner.ActL1StartBlock(10)(t)
	miner.ActL1EndBlock(t)
	reorgL1Block := miner.l1Chain.CurrentBlock()

	// Still no reorg, we need more L1 blocks first, before the reorged L1 block is forced in by sequence window
	verifier.ActL2PipelineFull(t)
	require.Equal(t, tip2N.SafeL2, verifier.SyncStatus().SafeL2)

	for miner.l1Chain.CurrentBlock().Number.Uint64() < sd.RollupCfg.SeqWindowSize*2 {
		miner.ActL1StartBlock(10)(t)
		miner.ActL1EndBlock(t)
	}

	// workaround: in L1Traversal we only recognize the reorg once we see origin N+1, we don't reorg to shorter L1 chains
	miner.ActL1StartBlock(10)(t)
	miner.ActL1EndBlock(t)

	// Now it will reorg
	verifier.ActL2PipelineFull(t)

	got := miner.l1Chain.GetBlockByHash(miner.l1Chain.GetBlockByHash(verifier.SyncStatus().SafeL2.L1Origin.Hash).Hash())
	require.Equal(t, reorgL1Block.Hash(), got.Hash(), "must have reorged L2 chain to the new L1 chain")
}
