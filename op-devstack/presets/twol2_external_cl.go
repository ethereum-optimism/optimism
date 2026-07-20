package presets

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// TwoL2ExternalCLInterop is a two-L2 interop setup without a supernode or
// supervisor: each chain runs a stock op-node sequencer (with its own EL and
// batcher) plus a dedicated verifier EL whose consensus client is created
// through the L2CLFactory seam (WithL2CLFactory). When no factory is injected,
// the verifier CLs are stock op-nodes, so the preset also works standalone.
type TwoL2ExternalCLInterop struct {
	TwoL2

	// TestSequencer provides deterministic block building on both L2 chains.
	TestSequencer *dsl.TestSequencer

	// L2ELA and L2ELB are the sequencer EL nodes for transaction submission.
	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	// L2AVerifierCL and L2BVerifierCL are the per-chain verifier consensus
	// clients — factory-provided when WithL2CLFactory is set, stock op-node
	// otherwise. L2ACL/L2BCL (from TwoL2) remain the sequencer CLs.
	L2AVerifierCL *dsl.L2CLNode
	L2BVerifierCL *dsl.L2CLNode

	// L2AVerifierEL and L2BVerifierEL are the execution layers driven by the
	// verifier consensus clients, distinct from the sequencer ELs and joined
	// to them only by L1 (and whatever sync the verifier CL implements).
	L2AVerifierEL *dsl.L2ELNode
	L2BVerifierEL *dsl.L2ELNode

	// L2BatcherA and L2BatcherB provide access to the batchers for pausing/resuming.
	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher

	// Faucets for funding test accounts.
	FaucetA *dsl.Faucet
	FaucetB *dsl.Faucet

	// Wallet for test account management.
	Wallet *dsl.HDWallet

	// Funders for creating funded EOAs.
	FunderA *dsl.Funder
	FunderB *dsl.Funder

	// GenesisTime is the genesis timestamp of the L2 chains.
	GenesisTime uint64

	// InteropActivationTime is the timestamp when interop becomes active.
	InteropActivationTime uint64

	// DelaySeconds is the delay from genesis to interop activation.
	DelaySeconds uint64

	timeTravel *clock.AdvancingClock
}

// L2UserRPCURLs returns the user-RPC URLs for both sequencer L2 EL nodes in
// canonical (A, B) order.
func (s *TwoL2ExternalCLInterop) L2UserRPCURLs() []string {
	return []string{s.L2ELA.Escape().UserRPC(), s.L2ELB.Escape().UserRPC()}
}

// AdvanceTime advances the time-travel clock if enabled.
func (s *TwoL2ExternalCLInterop) AdvanceTime(amount time.Duration) {
	s.T.Require().NotNil(s.timeTravel, "attempting to advance time on incompatible system")
	s.timeTravel.AdvanceTime(amount)
}

// NewTwoL2ExternalCLInterop creates a fresh TwoL2ExternalCLInterop target for
// the current test. Use delaySeconds=0 for interop at genesis. Inject an
// external verifier consensus-client implementation with WithL2CLFactory; the
// factory is consulted once per chain with a verifier launch context carrying
// the shared interop dependency set.
func NewTwoL2ExternalCLInterop(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2ExternalCLInterop {
	presetCfg, _ := collectSupportedPresetConfig(t, "NewTwoL2ExternalCLInterop", opts, twoL2ExternalCLInteropPresetSupportedOptionKinds)
	return twoL2ExternalCLInteropFromRuntime(t, sysgo.NewTwoL2ExternalCLInteropRuntimeWithConfig(t, delaySeconds, presetCfg))
}

func twoL2ExternalCLInteropFromRuntime(t devtest.T, runtime *sysgo.MultiChainRuntime) *TwoL2ExternalCLInterop {
	twoL2, components := twoL2FromRuntime(t, runtime)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a runtime chain")
	t.Require().NotNil(chainB, "missing l2b runtime chain")
	verifierA := chainA.Followers["verifier"]
	verifierB := chainB.Followers["verifier"]
	t.Require().NotNil(verifierA, "missing l2a verifier node")
	t.Require().NotNil(verifierB, "missing l2b verifier node")

	l2AVerifierEL := newL2ELFrontend(
		t,
		verifierA.Name,
		chainA.Network.ChainID(),
		verifierA.EL.UserRPC(),
		verifierA.EL.EngineRPC(),
		verifierA.EL.JWTPath(),
		chainA.Network.RollupConfig(),
		verifierA.EL,
	)
	l2AVerifierCL := newL2CLFrontend(t, verifierA.Name, chainA.Network.ChainID(), verifierA.CL.UserRPC(), verifierA.CL)
	l2AVerifierCL.attachEL(l2AVerifierEL)

	l2BVerifierEL := newL2ELFrontend(
		t,
		verifierB.Name,
		chainB.Network.ChainID(),
		verifierB.EL.UserRPC(),
		verifierB.EL.EngineRPC(),
		verifierB.EL.JWTPath(),
		chainB.Network.RollupConfig(),
		verifierB.EL,
	)
	l2BVerifierCL := newL2CLFrontend(t, verifierB.Name, chainB.Network.ChainID(), verifierB.CL.UserRPC(), verifierB.CL)
	l2BVerifierCL.attachEL(l2BVerifierEL)

	l2ANet, ok := twoL2.L2A.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network A")
	l2ANet.AddL2ELNode(l2AVerifierEL)
	l2ANet.AddL2CLNode(l2AVerifierCL)

	l2BNet, ok := twoL2.L2B.Escape().(*presetL2Network)
	t.Require().True(ok, "expected preset L2 network B")
	l2BNet.AddL2ELNode(l2BVerifierEL)
	l2BNet.AddL2CLNode(l2BVerifierCL)

	testSequencer := newTestSequencerFrontend(
		t,
		runtime.TestSequencer.Name,
		runtime.TestSequencer.AdminRPC,
		runtime.TestSequencer.ControlRPC,
		runtime.TestSequencer.JWTSecret,
	)

	genesisTime := twoL2.L2A.Escape().RollupConfig().Genesis.L2Time
	preset := &TwoL2ExternalCLInterop{
		TwoL2:                 *twoL2,
		TestSequencer:         dsl.NewTestSequencer(testSequencer),
		L2ELA:                 dsl.NewL2ELNode(components.l2AEL),
		L2ELB:                 dsl.NewL2ELNode(components.l2BEL),
		L2AVerifierCL:         dsl.NewL2CLNode(l2AVerifierCL),
		L2BVerifierCL:         dsl.NewL2CLNode(l2BVerifierCL),
		L2AVerifierEL:         dsl.NewL2ELNode(l2AVerifierEL),
		L2BVerifierEL:         dsl.NewL2ELNode(l2BVerifierEL),
		L2BatcherA:            dsl.NewL2Batcher(components.l2ABatcher),
		L2BatcherB:            dsl.NewL2Batcher(components.l2BBatcher),
		FaucetA:               components.faucetA,
		FaucetB:               components.faucetB,
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
		timeTravel:            runtime.TimeTravel,
	}
	preset.FunderA = dsl.NewFunder(preset.Wallet, preset.FaucetA, preset.L2ELA)
	preset.FunderB = dsl.NewFunder(preset.Wallet, preset.FaucetB, preset.L2ELB)
	return preset
}
