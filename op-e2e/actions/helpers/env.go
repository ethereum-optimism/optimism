package helpers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type Env struct {
	Log  log.Logger
	Logs *testlog.CapturingHandler

	SetupData *e2eutils.SetupData

	Miner       *L1Miner
	Seq         *L2Sequencer
	SeqEngine   *L2Engine
	Verifier    *L2Verifier
	VerifEngine *L2Engine
	Batcher     *L2Batcher
}

func SetupEnv(t StatefulTesting) (env Env) {
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())
	genesisOffset := hexutil.Uint64(0)

	log, logs := testlog.CaptureLogger(t, log.LevelDebug)
	env.Log, env.Logs = log, logs

	// Activate Holocene at genesis
	// TODO: make configurabe
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisGraniteTimeOffset = &genesisOffset
	dp.DeployConfig.L2GenesisHoloceneTimeOffset = &genesisOffset

	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	env.SetupData = sd
	env.Miner, env.SeqEngine, env.Seq = SetupSequencerTest(t, sd, log)
	env.Miner.ActL1SetFeeRecipient(common.Address{'A'})
	env.Seq.ActL2PipelineFull(t)
	env.VerifEngine, env.Verifier = SetupVerifier(t, sd, log, env.Miner.L1Client(t, sd.RollupCfg), env.Miner.BlobStore(), &sync.Config{})
	rollupSeqCl := env.Seq.RollupClient()
	env.Batcher = NewL2Batcher(log, sd.RollupCfg, DefaultBatcherCfg(dp),
		rollupSeqCl, env.Miner.EthClient(), env.SeqEngine.EthClient(), env.SeqEngine.EngineClient(t, sd.RollupCfg))

	return
}
