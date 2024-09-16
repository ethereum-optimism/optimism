package helpers

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func SetupSafeDBTest(t actions.Testing, config *e2eutils.TestParams) (*e2eutils.SetupData, *actions.L1Miner, *actions.L2Sequencer, *actions.L2Verifier, *actions.L2Engine, *actions.L2Batcher) {
	dp := e2eutils.MakeDeployParams(t, config)

	sd := e2eutils.Setup(t, dp, actions.DefaultAlloc)
	logger := testlog.Logger(t, log.LevelDebug)

	return SetupSafeDBTestActors(t, dp, sd, logger)
}

func SetupSafeDBTestActors(t actions.Testing, dp *e2eutils.DeployParams, sd *e2eutils.SetupData, log log.Logger) (*e2eutils.SetupData, *actions.L1Miner, *actions.L2Sequencer, *actions.L2Verifier, *actions.L2Engine, *actions.L2Batcher) {
	dir := t.TempDir()
	db, err := safedb.NewSafeDB(log, dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	miner, seqEngine, sequencer := actions.SetupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	verifEngine, verifier := actions.SetupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{}, actions.WithSafeHeadListener(db))
	rollupSeqCl := sequencer.RollupClient()
	batcher := actions.NewL2Batcher(log, sd.RollupCfg, actions.DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))
	return sd, miner, sequencer, verifier, verifEngine, batcher
}
