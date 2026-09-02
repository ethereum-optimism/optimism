package helpers

import (
	"crypto/ecdsa"
	"math/rand"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	e2ecfg "github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// L2FaultProofEnv is a test harness for a fault provable L2 chain.
type L2FaultProofEnv struct {
	log       log.Logger
	Logs      *testlog.CapturingHandler
	Batcher   *helpers.L2Batcher
	Sequencer *helpers.L2Sequencer
	Engine    *helpers.L2Engine
	engCl     *sources.EngineClient
	Sd        *e2eutils.SetupData
	Dp        *e2eutils.DeployParams
	Miner     *helpers.L1Miner
	Alice     *helpers.CrossLayerUser
	Bob       *helpers.CrossLayerUser
}

type DeployConfigOverride func(*genesis.DeployConfig)

func NewL2FaultProofEnv[c any](t helpers.Testing, testCfg *TestCfg[c], tp *e2eutils.TestParams, batcherCfg *helpers.BatcherCfg, deployConfigOverrides ...DeployConfigOverride) *L2FaultProofEnv {
	log, logs := testlog.CaptureLogger(t, log.LevelDebug)

	dp := NewDeployParams(t, tp, func(dp *e2eutils.DeployParams) {
		// Enable L2 feature.
		if testCfg.Hardfork == nil {
			t.Fatalf("HF not set")
		}
		dp.DeployConfig.ActivateForkAtGenesis(forks.Name(testCfg.Hardfork.Name))

		for _, override := range deployConfigOverrides {
			override(dp.DeployConfig)
		}
	})

	genesisAlloc := testCfg.Allocs
	if genesisAlloc == nil {
		genesisAlloc = helpers.DefaultAlloc
	}

	sd := e2eutils.Setup(t, dp, genesisAlloc)

	jwtPath := e2eutils.WriteDefaultJWT(t)

	miner := helpers.NewL1Miner(t, log.New("role", "l1-miner"), sd.L1Cfg)

	l1Cl, err := sources.NewL1Client(miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindStandard))
	require.NoError(t, err)
	engine := helpers.NewL2Engine(t, log.New("role", "sequencer-engine"), sd.L2Cfg, jwtPath, helpers.EngineWithP2P())
	l2EngineCl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	// Synthesise a one-chain depset when interop is scheduled; the sequencer's
	// FetchingAttributesBuilder constructor panics otherwise.
	if sd.RollupCfg.LagoonTime != nil && sd.DependencySet == nil {
		defaultDepSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
			eth.ChainIDFromBig(sd.RollupCfg.L2ChainID): {},
		})
		require.NoError(t, err)
		sd.DependencySet = defaultDepSet
	}
	sequencer := helpers.NewL2Sequencer(t, log.New("role", "sequencer"), l1Cl, miner.BlobStore(), altda.Disabled, l2EngineCl, sd.RollupCfg, sd.L1Cfg.Config, sd.DependencySet, 0)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	sequencer.ActL2PipelineFull(t)
	engCl := engine.EngineClient(t, sd.RollupCfg)

	// Set the batcher key to the secret key of the batcher
	batcherCfg.BatcherKey = dp.Secrets.Batcher
	batcher := helpers.NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engCl)

	addresses := e2eutils.CollectAddresses(sd, dp)
	l1EthCl := miner.EthClient()
	l2EthCl := engine.EthClient()
	l1UserEnv := &helpers.BasicUserEnv[*helpers.L1Bindings]{
		EthCl:          l1EthCl,
		Signer:         types.LatestSigner(sd.L1Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       helpers.NewL1Bindings(t, l1EthCl, e2ecfg.DefaultAllocType),
	}
	l2UserEnv := &helpers.BasicUserEnv[*helpers.L2Bindings]{
		EthCl:          l2EthCl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       helpers.NewL2Bindings(t, l2EthCl, engine.GethClient()),
	}
	alice := helpers.NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(0xa57b)), e2ecfg.DefaultAllocType)
	alice.L1.SetUserEnv(l1UserEnv)
	alice.L2.SetUserEnv(l2UserEnv)
	bob := helpers.NewCrossLayerUser(log, dp.Secrets.Bob, rand.New(rand.NewSource(0xbeef)), e2ecfg.DefaultAllocType)
	bob.L1.SetUserEnv(l1UserEnv)
	bob.L2.SetUserEnv(l2UserEnv)

	return &L2FaultProofEnv{
		log:       log,
		Logs:      logs,
		Batcher:   batcher,
		Sequencer: sequencer,
		Engine:    engine,
		engCl:     engCl,
		Sd:        sd,
		Dp:        dp,
		Miner:     miner,
		Alice:     alice,
		Bob:       bob,
	}
}

type FixtureInputParam func(f *FixtureInputs)

type CheckResult func(helpers.Testing, error)

func ExpectNoError() CheckResult {
	return func(t helpers.Testing, err error) {
		require.NoError(t, err, "fault proof program should have succeeded")
	}
}

func ExpectError(expectedErr error) CheckResult {
	return func(t helpers.Testing, err error) {
		require.ErrorIs(t, err, expectedErr, "fault proof program should have failed with expected error")
	}
}

// WithL2Claim overrides the claimed post-state. For the interop program that is the commitment to
// the claimed PreState (keccak of its encoding), not a bare output root — so a junk value is still
// a junk claim, but an honest override has to be built the same way WithSuperDefaults builds it.
func WithL2Claim(claim common.Hash) FixtureInputParam {
	return func(f *FixtureInputs) {
		f.L2Claim = claim
	}
}

func WithL1Head(head common.Hash) FixtureInputParam {
	return func(f *FixtureInputs) {
		f.L1Head = head
	}
}

// WithL2RPCTracker sets the L2RPCTracker to observe L2 JSON-RPC calls made by the program host.
func WithL2RPCTracker(tracker *L2RPCTracker) FixtureInputParam {
	return func(f *FixtureInputs) {
		f.L2RPCTracker = tracker
	}
}

// WithCorruptClaim instructs the SP1 super-range executor to corrupt the claim the guest sees,
// after witness collection has run on the honest one, so the guest rejects it. Used for the
// invalid-claim (soundness) test path. Has no effect on the native fault-proof program.
func WithCorruptClaim() FixtureInputParam {
	return func(f *FixtureInputs) {
		f.CorruptClaim = true
	}
}

// WithSP1NativeCore instructs the SP1 super-range executor to collect the witnesses and then
// replay them through the shared native cores instead of executing the SP1 ELF. This is useful for
// broad, faster kona-sp1 coverage; keep at least one default SP1 execute test for ELF/IO smoke
// coverage. Has no effect on the native fault-proof program.
func WithSP1NativeCore() FixtureInputParam {
	return func(f *FixtureInputs) {
		f.SP1NativeCore = true
	}
}

// RunFaultProofProgramFromGenesis runs the fault proof program for each state transition from
// genesis up to the provided l2 block num.
func (env *L2FaultProofEnv) RunFaultProofProgramFromGenesis(t helpers.Testing, finalL2BlockNum uint64, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	genesisBlockNum := env.Sd.RollupCfg.Genesis.L2.Number
	for l2ClaimBlockNum := genesisBlockNum + 1; l2ClaimBlockNum <= finalL2BlockNum; l2ClaimBlockNum++ {
		env.RunFaultProofProgram(t, l2ClaimBlockNum, checkResult, fixtureInputParams...)
	}
}

// RunFaultProofProgram runs the fault proof program for a single state transition, from the provided l2 block num - 1 to the provided l2 block num.
func (env *L2FaultProofEnv) RunFaultProofProgram(t helpers.Testing, l2ClaimBlockNum uint64, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	defaultParam := WithSuperDefaults(t, l2ClaimBlockNum, env.Sequencer.L2Verifier, env.Engine)
	combinedParams := []FixtureInputParam{defaultParam}
	combinedParams = append(combinedParams, fixtureInputParams...)
	RunFaultProofProgram(t, env.log, env.Miner, checkResult, combinedParams...)
}

// RunSP1SuperRangeProgram runs the kona-sp1 super-range guest in SP1 execute mode over the
// super-root transition into the provided l2 block num.
//
// The executor is a separate process that resolves the transition itself from
// `superroot_atTimestamp`, so this serves the verifier's superroot API over loopback HTTP and
// hands it that address. WithSuperDefaults still supplies the claimed timestamp and the L2 source
// endpoints; its agreed pre-state and claim go unused on this path.
func (env *L2FaultProofEnv) RunSP1SuperRangeProgram(t helpers.Testing, l2ClaimBlockNum uint64, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	l2 := env.Sequencer.L2Verifier
	combinedParams := []FixtureInputParam{
		WithSuperDefaults(t, l2ClaimBlockNum, l2, env.Engine),
		WithSupernode(l2.StartSuperRootHTTPRPC(t)),
	}
	combinedParams = append(combinedParams, fixtureInputParams...)
	RunSP1SuperRangeProgram(t, env.log, env.Miner, checkResult, combinedParams...)
}

type TestParam func(p *e2eutils.TestParams)

func NewTestParams(params ...TestParam) *e2eutils.TestParams {
	dfault := helpers.DefaultRollupTestParams()
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

type DeployParam func(p *e2eutils.DeployParams)

func NewDeployParams(t helpers.Testing, tp *e2eutils.TestParams, params ...DeployParam) *e2eutils.DeployParams {
	dfault := e2eutils.MakeDeployParams(t, tp)
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

type BatcherCfgParam func(c *helpers.BatcherCfg)

func NewBatcherCfg(params ...BatcherCfgParam) *helpers.BatcherCfg {
	dfault := &helpers.BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		DataAvailabilityType: batcherFlags.BlobsType,
	}
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

const L1BlockTime = 12

// RotateBatcher updates the on-chain batcher address in the SystemConfig contract and replaces
// env.Batcher with a new one using the given key. The L1 transaction is mined in a new L1 block.
func (env *L2FaultProofEnv) RotateBatcher(t helpers.Testing, newBatcherKey *ecdsa.PrivateKey) {
	t.Helper()
	newAddr := crypto.PubkeyToAddress(newBatcherKey.PublicKey)

	sysCfgContract, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
	require.NoError(t, err)
	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
	require.NoError(t, err)
	_, err = sysCfgContract.SetBatcherHash(sysCfgOwner, eth.AddressAsLeftPaddedHash(newAddr))
	require.NoError(t, err)

	env.Miner.ActL1StartBlock(L1BlockTime)(t)
	env.Miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	env.Miner.ActL1EndBlock(t)

	batcherCfg := NewBatcherCfg()
	batcherCfg.BatcherKey = newBatcherKey
	env.Batcher = helpers.NewL2Batcher(
		env.log, env.Sd.RollupCfg, batcherCfg,
		env.Sequencer.RollupClient(), env.Miner.EthClient(),
		env.Engine.EthClient(), env.engCl,
	)
}

// BatchAndMine batches the current unsafe chain to L1 and mines the L1 block containing the
// batcher transaction.
func (env *L2FaultProofEnv) BatchAndMine(t helpers.Testing) {
	t.Helper()
	env.Batcher.ActSubmitAll(t)
	env.Miner.ActL1StartBlock(L1BlockTime)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)
}

// BatchMineAndSync calls env.BatchAndMine and then has the sequencer derive up to the l1 head.
// Returns the L2 Safe Block Reference
func (env *L2FaultProofEnv) BatchMineAndSync(t helpers.Testing) eth.L2BlockRef {
	t.Helper()
	env.BatchAndMine(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	// Assertions
	syncStatus := env.Sequencer.SyncStatus()
	require.Equal(t, syncStatus.UnsafeL2, syncStatus.SafeL2, "UnsafeL2 should equal SafeL2")

	return syncStatus.SafeL2
}
