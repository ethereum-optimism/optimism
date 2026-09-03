package presets

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

// TwoL2ExternalCLInterop is a two-chain interop setup without a supernode or
// supervisor. Each chain has a stock sequencer and a dedicated verifier slot
// supplied through WithL2CLFactory (or ordinary client selection as fallback).
type TwoL2ExternalCLInterop struct {
	TwoL2

	TestSequencer *dsl.TestSequencer

	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	L2AVerifierCL *dsl.L2CLNode
	L2BVerifierCL *dsl.L2CLNode
	L2AVerifierEL *dsl.L2ELNode
	L2BVerifierEL *dsl.L2ELNode

	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher

	Wallet  *dsl.HDWallet
	FunderA *dsl.FunderEOA
	FunderB *dsl.FunderEOA

	GenesisTime           uint64
	InteropActivationTime uint64
	DelaySeconds          uint64

	timeTravel *clock.AdvancingClock
}

func (s *TwoL2ExternalCLInterop) L2UserRPCURLs() []string {
	return []string{s.L2ELA.Escape().UserRPC(), s.L2ELB.Escape().UserRPC()}
}

func (s *TwoL2ExternalCLInterop) AdvanceTime(amount time.Duration) {
	s.T.Require().NotNil(s.timeTravel, "attempting to advance time on incompatible system")
	s.L1EL.AdvanceTime(s.timeTravel, amount)
}

// NewTwoL2ExternalCLInterop constructs the no-supernode interop preset.
func NewTwoL2ExternalCLInterop(t devtest.T, delaySeconds uint64, opts ...Option) *TwoL2ExternalCLInterop {
	presetCfg, _ := collectSupportedPresetConfig(
		t, "NewTwoL2ExternalCLInterop", opts, twoL2ExternalCLInteropPresetSupportedOptionKinds,
	)
	return twoL2ExternalCLInteropFromRuntime(
		t, sysgo.NewTwoL2ExternalCLInteropRuntimeWithConfig(t, delaySeconds, presetCfg),
	)
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
		t, verifierA.Name, chainA.Network.ChainID(), verifierA.EL.UserRPC(), verifierA.EL.EngineRPC(),
		verifierA.EL.JWTPath(), chainA.Network.RollupConfig(), verifierA.EL,
	)
	l2AVerifierCL := newL2CLFrontend(
		t, verifierA.Name, chainA.Network.ChainID(), verifierA.CL.UserRPC(), verifierA.CL,
	)
	l2AVerifierCL.attachEL(l2AVerifierEL)

	l2BVerifierEL := newL2ELFrontend(
		t, verifierB.Name, chainB.Network.ChainID(), verifierB.EL.UserRPC(), verifierB.EL.EngineRPC(),
		verifierB.EL.JWTPath(), chainB.Network.RollupConfig(), verifierB.EL,
	)
	l2BVerifierCL := newL2CLFrontend(
		t, verifierB.Name, chainB.Network.ChainID(), verifierB.CL.UserRPC(), verifierB.CL,
	)
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
	out := &TwoL2ExternalCLInterop{
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
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + runtime.DelaySeconds,
		DelaySeconds:          runtime.DelaySeconds,
		timeTravel:            runtime.TimeTravel,
	}
	out.FunderA = newFunderEOA(t, runtime.Keys, out.L2ELA, out.Wallet)
	out.FunderB = newFunderEOA(t, runtime.Keys, out.L2ELB, out.Wallet)
	return out
}
