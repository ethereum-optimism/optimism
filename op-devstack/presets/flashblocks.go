package presets

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type SingleChainWithFlashblocks struct {
	*Minimal

	L2OPRBuilder  *dsl.OPRBuilderNode
	L2RollupBoost *dsl.RollupBoostNode
	TestSequencer *dsl.TestSequencer
}

func (m *SingleChainWithFlashblocks) L2Networks() []*dsl.L2Network {
	return []*dsl.L2Network{
		m.L2Chain,
	}
}

func (m *SingleChainWithFlashblocks) StandardBridge() *dsl.StandardBridge {
	return dsl.NewStandardBridge(m.T, m.L2Chain, m.L1EL)
}

func (m *SingleChainWithFlashblocks) DisputeGameFactory() *proofs.DisputeGameFactory {
	return proofs.NewDisputeGameFactory(m.T, m.L1Network, m.L1EL.EthClient(), m.L2Chain.DisputeGameFactoryProxyAddr(), m.L2CL, m.L2EL, nil, m.challengerConfig)
}

func (m *SingleChainWithFlashblocks) AdvanceTime(amount time.Duration) {
	m.Minimal.AdvanceTime(amount)
}

func NewSingleChainWithFlashblocks(t devtest.T, opts ...Option) *SingleChainWithFlashblocks {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewSingleChainWithFlashblocks", opts, singleChainWithFlashblocksPresetSupportedOptionKinds)
	runtime := sysgo.NewFlashblocksRuntimeWithConfig(t, presetCfg)
	return singleChainWithFlashblocksFromRuntime(t, runtime)
}

func singleChainWithFlashblocksFromRuntime(t devtest.T, runtime *sysgo.SingleChainRuntime) *SingleChainWithFlashblocks {
	t.Require().NotNil(runtime.Flashblocks, "missing flashblocks support")
	l1ChainID := runtime.L1Network.ChainID()
	l2ChainID := runtime.L2Network.ChainID()

	l1Network := newPresetL1Network(t, "l1", runtime.L1Network.ChainConfig())
	l1EL := newL1ELFrontend(t, "l1", l1ChainID, runtime.L1EL.UserRPC())
	l1CL := newL1CLFrontend(t, "l1", l1ChainID, runtime.L1CL.BeaconHTTPAddr(), runtime.L1CL.FakePoS())
	l1Network.AddL1ELNode(l1EL)
	l1Network.AddL1CLNode(l1CL)

	l2Chain := newPresetL2Network(
		t,
		"l2a",
		runtime.L2Network.ChainConfig(),
		runtime.L2Network.RollupConfig(),
		runtime.L2Network.Deployment(),
		newKeyring(runtime.Keys, t.Require()),
		l1Network,
	)

	l2EL := newL2ELFrontend(
		t,
		"sequencer",
		l2ChainID,
		runtime.L2EL.UserRPC(),
		runtime.L2EL.EngineRPC(),
		runtime.L2EL.JWTPath(),
		runtime.L2Network.RollupConfig(),
	)
	l2CL := newL2CLFrontend(
		t,
		"sequencer",
		l2ChainID,
		runtime.L2CL.UserRPC(),
		runtime.L2CL,
	)

	l2OPRBuilder := newOPRBuilderFrontend(
		t,
		"sequencer-builder",
		l2ChainID,
		runtime.Flashblocks.Builder.UserRPC(),
		runtime.Flashblocks.Builder.FlashblocksWSURL(),
		runtime.L2Network.RollupConfig(),
		runtime.Flashblocks.Builder,
	)
	l2RollupBoost := newRollupBoostFrontend(
		t,
		"rollup-boost",
		l2ChainID,
		runtime.Flashblocks.RollupBoost.UserRPC(),
		runtime.Flashblocks.RollupBoost.FlashblocksWSURL(),
		runtime.L2Network.RollupConfig(),
		runtime.Flashblocks.RollupBoost,
	)
	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)

	l2Chain.AddL2ELNode(l2EL)
	l2Chain.AddL2CLNode(l2CL)
	l2Chain.AddOPRBuilderNode(l2OPRBuilder)
	l2Chain.AddRollupBoostNode(l2RollupBoost)
	l2CL.attachEL(l2EL)
	l2CL.attachOPRBuilderNode(l2OPRBuilder)
	l2CL.attachRollupBoostNode(l2RollupBoost)

	faucetL1Frontend := newFaucetFrontendForChain(t, runtime.FaucetService, l1ChainID)
	faucetL2Frontend := newFaucetFrontendForChain(t, runtime.FaucetService, l2ChainID)
	l1Network.AddFaucet(faucetL1Frontend)
	l2Chain.AddFaucet(faucetL2Frontend)
	faucetL1 := dsl.NewFaucet(faucetL1Frontend)
	faucetL2 := dsl.NewFaucet(faucetL2Frontend)

	l1ELDSL := dsl.NewL1ELNode(l1EL)
	l1CLDSL := dsl.NewL1CLNode(l1CL)
	l2ELDSL := dsl.NewL2ELNode(l2EL)
	l2CLDSL := dsl.NewL2CLNode(l2CL)

	minimal := &Minimal{
		Log:       t.Logger(),
		T:         t,
		L1Network: dsl.NewL1Network(l1Network, l1ELDSL, l1CLDSL),
		L1EL:      l1ELDSL,
		L1CL:      l1CLDSL,
		L2Chain:   dsl.NewL2Network(l2Chain, l2ELDSL, l2CLDSL, l1ELDSL, nil, nil),
		L2EL:      l2ELDSL,
		L2CL:      l2CLDSL,
		Wallet:    dsl.NewRandomHDWallet(t, 30), // Random for test isolation
		FaucetL1:  faucetL1,
		FaucetL2:  faucetL2,
	}
	minimal.FunderL1 = dsl.NewFunder(minimal.Wallet, minimal.FaucetL1, minimal.L1EL)
	minimal.FunderL2 = dsl.NewFunder(minimal.Wallet, minimal.FaucetL2, minimal.L2EL)

	return &SingleChainWithFlashblocks{
		L2OPRBuilder:  dsl.NewOPRBuilderNode(l2OPRBuilder),
		L2RollupBoost: dsl.NewRollupBoostNode(l2RollupBoost),
		Minimal:       minimal,
		TestSequencer: dsl.NewTestSequencer(testSequencer),
	}
}

func newFaucetFrontendForChain(t devtest.T, faucetService *faucet.Service, chainID eth.ChainID) *faucetFrontend {
	faucetName, faucetRPC, ok := defaultFaucetForChain(faucetService, chainID)
	t.Require().Truef(ok, "missing default faucet for chain %s", chainID)

	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), faucetRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)

	return newPresetFaucet(t, faucetName, chainID, rpcCl)
}

func defaultFaucetForChain(faucetService *faucet.Service, chainID eth.ChainID) (string, string, bool) {
	if faucetService == nil {
		return "", "", false
	}
	faucetID, ok := faucetService.Defaults()[chainID]
	if !ok {
		return "", "", false
	}
	return faucetID.String(), faucetService.FaucetEndpoint(faucetID), true
}
