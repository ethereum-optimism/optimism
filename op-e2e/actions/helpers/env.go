package helpers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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

type EnvOpt struct {
	DeployConfigMod func(*genesis.DeployConfig)
}

func WithActiveFork(fork rollup.ForkName, offset *uint64) EnvOpt {
	return EnvOpt{
		DeployConfigMod: func(d *genesis.DeployConfig) {
			d.ActivateForkAtOffset(fork, offset)
		},
	}
}

func WithActiveGenesisFork(fork rollup.ForkName) EnvOpt {
	return WithActiveFork(fork, new(uint64))
}

func SetupEnv(t StatefulTesting, opts ...EnvOpt) (env Env) {
	dp := e2eutils.MakeDeployParams(t, DefaultRollupTestParams())

	log, logs := testlog.CaptureLogger(t, log.LevelDebug)
	env.Log, env.Logs = log, logs

	dp.DeployConfig.ActivateForkAtGenesis(rollup.LatestFork)
	for _, opt := range opts {
		if dcMod := opt.DeployConfigMod; dcMod != nil {
			dcMod(dp.DeployConfig)
		}
	}

	sd := e2eutils.Setup(t, dp, DefaultAlloc)
	env.SetupData = sd
	env.Miner, env.SeqEngine, env.Seq = SetupSequencerTest(t, sd, log)
	env.Miner.ActL1SetFeeRecipient(common.Address{'A'})
	env.VerifEngine, env.Verifier = SetupVerifier(t, sd, log, env.Miner.L1Client(t, sd.RollupCfg), env.Miner.BlobStore(), &sync.Config{})
	rollupSeqCl := env.Seq.RollupClient()
	env.Batcher = NewL2Batcher(log, sd.RollupCfg, DefaultBatcherCfg(dp),
		rollupSeqCl, env.Miner.EthClient(), env.SeqEngine.EthClient(), env.SeqEngine.EngineClient(t, sd.RollupCfg))

	return
}

func (env Env) ActBatchSubmitAllAndMine(t Testing) (l1InclusionBlock *types.Block) {
	env.Batcher.ActSubmitAll(t)
	batchTx := env.Batcher.LastSubmitted
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	return env.Miner.ActL1EndBlock(t)
}
