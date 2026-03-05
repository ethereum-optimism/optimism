package presets

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TwoL2 represents a two-L2 setup without interop considerations.
// It is useful for testing components which bridge multiple L2s without necessarily using interop.
type TwoL2 struct {
	Log          log.Logger
	T            devtest.T
	ControlPlane stack.ControlPlane

	L1Network *dsl.L1Network
	L1EL      *dsl.L1ELNode

	L2A   *dsl.L2Network
	L2B   *dsl.L2Network
	L2ACL *dsl.L2CLNode
	L2BCL *dsl.L2CLNode
}

func WithTwoL2() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}))
}

func WithTwoL2Supernode() stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSupernodeTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}))
}

// WithTwoL2SupernodeInterop specifies a two-L2 system using a shared supernode with interop enabled.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition from
// normal safety to interop-verified safety.
func WithTwoL2SupernodeInterop(delaySeconds uint64) stack.CommonOption {
	return stack.MakeCommon(sysgo.DefaultSupernodeInteropTwoL2System(&sysgo.DefaultTwoL2SystemIDs{}, delaySeconds))
}

func NewTwoL2(t devtest.T) *TwoL2 {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	l1Net := system.L1Network(match.FirstL1Network)
	l2a := system.L2Network(match.Assume(t, match.L2ChainA))
	l2b := system.L2Network(match.Assume(t, match.L2ChainB))
	l2aCL := l2a.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	l2bCL := l2b.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))

	require.NotEqual(t, l2a.ChainID(), l2b.ChainID())

	return &TwoL2{
		Log:          t.Logger(),
		T:            t,
		ControlPlane: orch.ControlPlane(),
		L1Network:    dsl.NewL1Network(l1Net),
		L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
		L2A:          dsl.NewL2Network(l2a, orch.ControlPlane()),
		L2B:          dsl.NewL2Network(l2b, orch.ControlPlane()),
		L2ACL:        dsl.NewL2CLNode(l2aCL, orch.ControlPlane()),
		L2BCL:        dsl.NewL2CLNode(l2bCL, orch.ControlPlane()),
	}
}

// TwoL2SupernodeInterop represents a two-L2 setup with a shared supernode that has interop enabled.
// This allows testing of cross-chain message verification at each timestamp.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition.
type TwoL2SupernodeInterop struct {
	TwoL2

	// Supernode provides access to the shared supernode for interop operations
	Supernode *dsl.Supernode

	// TestSequencer provides deterministic block building on both L2 chains.
	// Unlike the regular sequencer which uses wall-clock time, the TestSequencer
	// builds blocks at parent.Time + blockTime, making it ideal for same-timestamp tests.
	TestSequencer *dsl.TestSequencer

	// L2ELA and L2ELB provide access to the EL nodes for transaction submission
	L2ELA *dsl.L2ELNode
	L2ELB *dsl.L2ELNode

	// L2BatcherA and L2BatcherB provide access to the batchers for pausing/resuming
	L2BatcherA *dsl.L2Batcher
	L2BatcherB *dsl.L2Batcher

	// Faucets for funding test accounts
	FaucetA *dsl.Faucet
	FaucetB *dsl.Faucet

	// Wallet for test account management
	Wallet *dsl.HDWallet

	// Funders for creating funded EOAs
	FunderA *dsl.Funder
	FunderB *dsl.Funder

	// GenesisTime is the genesis timestamp of the L2 chains
	GenesisTime uint64

	// InteropActivationTime is the timestamp when interop becomes active
	InteropActivationTime uint64

	// DelaySeconds is the delay from genesis to interop activation
	DelaySeconds uint64

	// system holds the underlying system for advanced operations
	system stack.ExtensibleSystem
}

// AdvanceTime advances the time-travel clock if enabled.
func (s *TwoL2SupernodeInterop) AdvanceTime(amount time.Duration) {
	ttSys, ok := s.system.(stack.TimeTravelSystem)
	s.T.Require().True(ok, "attempting to advance time on incompatible system")
	ttSys.AdvanceTime(amount)
}

// SuperNodeClient returns an API for calling supernode-specific RPC methods
// like superroot_atTimestamp.
func (s *TwoL2SupernodeInterop) SuperNodeClient() apis.SupernodeQueryAPI {
	return s.Supernode.QueryAPI()
}

// NewTwoL2SupernodeInterop creates a TwoL2SupernodeInterop preset for acceptance tests.
// Use delaySeconds=0 for interop at genesis, or a positive value to test the transition.
// The delaySeconds must match what was passed to WithTwoL2SupernodeInterop in TestMain.
func NewTwoL2SupernodeInterop(t devtest.T, delaySeconds uint64) *TwoL2SupernodeInterop {
	system := shim.NewSystem(t)
	orch := Orchestrator()
	orch.Hydrate(system)

	l1Net := system.L1Network(match.FirstL1Network)
	l2a := system.L2Network(match.Assume(t, match.L2ChainA))
	l2b := system.L2Network(match.Assume(t, match.L2ChainB))
	l2aCL := l2a.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))
	l2bCL := l2b.L2CLNode(match.Assume(t, match.WithSequencerActive(t.Ctx())))

	require.NotEqual(t, l2a.ChainID(), l2b.ChainID())

	// Get genesis time from the DSL wrapper
	l2aNet := dsl.NewL2Network(l2a, orch.ControlPlane())
	genesisTime := l2aNet.Escape().RollupConfig().Genesis.L2Time

	// Get the supernode and its test control
	stackSupernode := system.Supernode(match.Assume(t, match.FirstSupernode))
	var testControl stack.InteropTestControl
	if sysgoOrch, ok := orch.(*sysgo.Orchestrator); ok {
		testControl = sysgoOrch.InteropTestControl(stackSupernode.ID())
	}

	// Get the test sequencer for deterministic block building
	var testSequencer *dsl.TestSequencer
	if len(system.TestSequencers()) > 0 {
		testSequencer = dsl.NewTestSequencer(system.TestSequencer(match.Assume(t, match.FirstTestSequencer)))
	}

	out := &TwoL2SupernodeInterop{
		TwoL2: TwoL2{
			Log:          t.Logger(),
			T:            t,
			ControlPlane: orch.ControlPlane(),
			L1Network:    dsl.NewL1Network(l1Net),
			L1EL:         dsl.NewL1ELNode(l1Net.L1ELNode(match.Assume(t, match.FirstL1EL))),
			L2A:          l2aNet,
			L2B:          dsl.NewL2Network(l2b, orch.ControlPlane()),
			L2ACL:        dsl.NewL2CLNode(l2aCL, orch.ControlPlane()),
			L2BCL:        dsl.NewL2CLNode(l2bCL, orch.ControlPlane()),
		},
		Supernode:             dsl.NewSupernodeWithTestControl(stackSupernode, testControl),
		TestSequencer:         testSequencer,
		L2ELA:                 dsl.NewL2ELNode(l2a.L2ELNode(match.Assume(t, match.FirstL2EL)), orch.ControlPlane()),
		L2ELB:                 dsl.NewL2ELNode(l2b.L2ELNode(match.Assume(t, match.FirstL2EL)), orch.ControlPlane()),
		L2BatcherA:            dsl.NewL2Batcher(l2a.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		L2BatcherB:            dsl.NewL2Batcher(l2b.L2Batcher(match.Assume(t, match.FirstL2Batcher))),
		FaucetA:               dsl.NewFaucet(l2a.Faucet(match.Assume(t, match.FirstFaucet))),
		FaucetB:               dsl.NewFaucet(l2b.Faucet(match.Assume(t, match.FirstFaucet))),
		Wallet:                dsl.NewRandomHDWallet(t, 30),
		GenesisTime:           genesisTime,
		InteropActivationTime: genesisTime + delaySeconds,
		DelaySeconds:          delaySeconds,
		system:                system,
	}
	out.FunderA = dsl.NewFunder(out.Wallet, out.FaucetA, out.L2ELA)
	out.FunderB = dsl.NewFunder(out.Wallet, out.FaucetB, out.L2ELB)
	return out
}

// =============================================================================
// Same-Timestamp Test Setup
// =============================================================================

// SameTimestampTestSetup provides a simplified setup for same-timestamp interop testing.
// It handles all the chain synchronization, sequencer control, and interop pausing
// needed to create blocks at the same timestamp on both chains.
type SameTimestampTestSetup struct {
	*TwoL2SupernodeInterop
	t devtest.T

	// Alice is a funded EOA on chain A
	Alice *dsl.EOA
	// Bob is a funded EOA on chain B
	Bob *dsl.EOA

	// EventLoggerA is the EventLogger contract address on chain A
	EventLoggerA common.Address
	// EventLoggerB is the EventLogger contract address on chain B
	EventLoggerB common.Address

	// NextTimestamp is the timestamp that will be used for the next blocks
	NextTimestamp uint64
	// ExpectedBlockNumA is the expected block number on chain A
	ExpectedBlockNumA uint64
	// ExpectedBlockNumB is the expected block number on chain B
	ExpectedBlockNumB uint64
}

// ForSameTimestampTesting sets up the system for same-timestamp interop testing.
// It syncs the chains, pauses interop, stops sequencers, and calculates expected positions.
// After calling this, you can use PrepareInitA/B to create same-timestamp message pairs.
func (s *TwoL2SupernodeInterop) ForSameTimestampTesting(t devtest.T) *SameTimestampTestSetup {
	// Create funded EOAs
	alice := s.FunderA.NewFundedEOA(eth.OneEther)
	bob := s.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event loggers
	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()

	// Sync chains and pause interop
	s.L2B.CatchUpTo(s.L2A)
	s.L2A.CatchUpTo(s.L2B)
	s.Supernode.EnsureInteropPaused(s.L2ACL, s.L2BCL, 10)

	// Stop sequencers
	s.L2ACL.StopSequencer()
	s.L2BCL.StopSequencer()

	// Get current state and synchronize timestamps
	unsafeA := s.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := s.L2ELB.BlockRefByLabel(eth.Unsafe)
	unsafeA, unsafeB = synchronizeChainsToSameTimestamp(t, s, unsafeA, unsafeB)

	blockTime := s.L2A.Escape().RollupConfig().BlockTime

	return &SameTimestampTestSetup{
		TwoL2SupernodeInterop: s,
		t:                     t,
		Alice:                 alice,
		Bob:                   bob,
		EventLoggerA:          eventLoggerA,
		EventLoggerB:          eventLoggerB,
		NextTimestamp:         unsafeA.Time + blockTime,
		ExpectedBlockNumA:     unsafeA.Number + 1,
		ExpectedBlockNumB:     unsafeB.Number + 1,
	}
}

// PrepareInitA creates a precomputed init message for chain A at the next timestamp.
func (s *SameTimestampTestSetup) PrepareInitA(rng *rand.Rand, logIdx uint32) *dsl.SameTimestampPair {
	return s.Alice.PrepareSameTimestampInit(rng, s.EventLoggerA, s.ExpectedBlockNumA, logIdx, s.NextTimestamp)
}

// PrepareInitB creates a precomputed init message for chain B at the next timestamp.
func (s *SameTimestampTestSetup) PrepareInitB(rng *rand.Rand, logIdx uint32) *dsl.SameTimestampPair {
	return s.Bob.PrepareSameTimestampInit(rng, s.EventLoggerB, s.ExpectedBlockNumB, logIdx, s.NextTimestamp)
}

// IncludeAndValidate builds blocks with deterministic timestamps using the TestSequencer,
// then validates interop and checks for expected reorgs.
//
// Unlike the regular sequencer which uses wall-clock time, the TestSequencer builds blocks
// at exactly parent.Time + blockTime, ensuring the blocks are at NextTimestamp.
func (s *SameTimestampTestSetup) IncludeAndValidate(txsA, txsB []*txplan.PlannedTx, expectReplacedA, expectReplacedB bool) {
	ctx := s.t.Ctx()

	require.NotNil(s.t, s.TestSequencer, "TestSequencer is required for deterministic timestamp tests")

	// Get parent blocks and chain IDs
	parentA := s.L2ELA.BlockRefByLabel(eth.Unsafe)
	parentB := s.L2ELB.BlockRefByLabel(eth.Unsafe)
	chainIDA := s.L2A.Escape().ChainID()
	chainIDB := s.L2B.Escape().ChainID()

	// Extract signed transaction bytes for chain A
	var rawTxsA [][]byte
	var txHashesA []common.Hash
	for _, ptx := range txsA {
		signedTx, err := ptx.Signed.Eval(ctx)
		require.NoError(s.t, err, "failed to sign transaction for chain A")
		rawBytes, err := signedTx.MarshalBinary()
		require.NoError(s.t, err, "failed to marshal transaction for chain A")
		rawTxsA = append(rawTxsA, rawBytes)
		txHashesA = append(txHashesA, signedTx.Hash())
	}

	// Extract signed transaction bytes for chain B
	var rawTxsB [][]byte
	var txHashesB []common.Hash
	for _, ptx := range txsB {
		signedTx, err := ptx.Signed.Eval(ctx)
		require.NoError(s.t, err, "failed to sign transaction for chain B")
		rawBytes, err := signedTx.MarshalBinary()
		require.NoError(s.t, err, "failed to marshal transaction for chain B")
		rawTxsB = append(rawTxsB, rawBytes)
		txHashesB = append(txHashesB, signedTx.Hash())
	}

	// Build blocks at deterministic timestamps using TestSequencer
	// Block timestamp will be parent.Time + blockTime = NextTimestamp
	s.TestSequencer.SequenceBlockWithTxs(s.t, chainIDA, parentA.Hash, rawTxsA)
	s.TestSequencer.SequenceBlockWithTxs(s.t, chainIDB, parentB.Hash, rawTxsB)

	// Get block refs by looking up the tx receipts
	var blockA, blockB eth.L2BlockRef
	for _, txHash := range txHashesA {
		receipt := s.L2ELA.WaitForReceipt(txHash)
		blockA = s.L2ELA.BlockRefByHash(receipt.BlockHash)
	}
	for _, txHash := range txHashesB {
		receipt := s.L2ELB.WaitForReceipt(txHash)
		blockB = s.L2ELB.BlockRefByHash(receipt.BlockHash)
	}

	// Verify same-timestamp property: both blocks at expected timestamp
	require.Equal(s.t, s.NextTimestamp, blockA.Time,
		"Chain A block must be at the precomputed NextTimestamp (init message identifier uses this)")
	require.Equal(s.t, s.NextTimestamp, blockB.Time,
		"Chain B block must be at the precomputed NextTimestamp (exec references init at this timestamp)")
	require.Equal(s.t, blockA.Time, blockB.Time, "blocks must be at same timestamp")

	// Resume interop and wait for validation
	s.Supernode.ResumeInterop()
	s.Supernode.AwaitValidatedTimestamp(blockA.Time)

	// Check reorg expectations
	currentA := s.L2ELA.BlockRefByNumber(blockA.Number)
	currentB := s.L2ELB.BlockRefByNumber(blockB.Number)

	if expectReplacedA {
		require.NotEqual(s.t, blockA.Hash, currentA.Hash, "Chain A should be replaced")
	} else {
		require.Equal(s.t, blockA.Hash, currentA.Hash, "Chain A should NOT be replaced")
	}

	if expectReplacedB {
		require.NotEqual(s.t, blockB.Hash, currentB.Hash, "Chain B should be replaced")
	} else {
		require.Equal(s.t, blockB.Hash, currentB.Hash, "Chain B should NOT be replaced")
	}
}

// synchronizeChainsToSameTimestamp ensures both chains are at the same timestamp.
func synchronizeChainsToSameTimestamp(t devtest.T, sys *TwoL2SupernodeInterop, unsafeA, unsafeB eth.L2BlockRef) (eth.L2BlockRef, eth.L2BlockRef) {
	for i := 0; i < 10; i++ {
		if unsafeA.Time == unsafeB.Time {
			return unsafeA, unsafeB
		}
		if unsafeA.Time < unsafeB.Time {
			sys.L2ACL.StartSequencer()
			sys.L2ELA.WaitForTime(unsafeB.Time)
			sys.L2ACL.StopSequencer()
			unsafeA = sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		} else {
			sys.L2BCL.StartSequencer()
			sys.L2ELB.WaitForTime(unsafeA.Time)
			sys.L2BCL.StopSequencer()
			unsafeB = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		}
	}
	require.Equal(t, unsafeA.Time, unsafeB.Time, "failed to synchronize chains")
	return unsafeA, unsafeB
}
