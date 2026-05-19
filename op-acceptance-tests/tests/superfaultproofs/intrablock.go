package superfaultproofs

import (
	"bytes"
	"math/rand"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// IntraBlockTestCase describes a single intra-block scenario.
type IntraBlockTestCase struct {
	Name     string
	BuildTxs func(s *intraBlockSetup) (txsA, txsB []*txplan.PlannedTx)
}

// intraBlockSetup holds shared state for building same-timestamp transactions.
type intraBlockSetup struct {
	rng               *rand.Rand
	alice             *dsl.EOA // funded on chain A
	bob               *dsl.EOA // funded on chain B
	eventLoggerA      common.Address
	eventLoggerB      common.Address
	expectedBlockNumA uint64
	expectedBlockNumB uint64
	nextTimestamp     uint64
}

func (s *intraBlockSetup) prepareInitA(logIdx uint32) *dsl.SameTimestampPair {
	return s.alice.PrepareSameTimestampInit(s.rng, s.eventLoggerA, s.expectedBlockNumA, logIdx, s.nextTimestamp)
}

func (s *intraBlockSetup) prepareInitB(logIdx uint32) *dsl.SameTimestampPair {
	return s.bob.PrepareSameTimestampInit(s.rng, s.eventLoggerB, s.expectedBlockNumB, logIdx, s.nextTimestamp)
}

// RunIntraBlockConsolidationTest verifies that the consolidation step in the
// super-root transition correctly handles intra-block cross-chain messages.
//
// It builds one block per chain at the same timestamp with the given transaction
// set, waits for the supernode to validate and replace invalid blocks, then
// verifies the consolidation transition via kona and the challenger trace provider.
func RunIntraBlockConsolidationTest(t devtest.T, sys *presets.SimpleInterop, tc *IntraBlockTestCase) {
	t.Require().NotNil(sys.SuperRoots, "supernode is required for this test")
	t.Require().NotNil(sys.TestSequencer, "test sequencer is required for same-timestamp block building")

	rng := rand.New(rand.NewSource(98765))
	chains := orderedChains(sys)
	t.Require().Len(chains, 2, "expected exactly 2 interop chains")

	// --- Create funded EOAs and deploy event loggers -------------------------
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()

	// --- Sync chains and prepare for same-timestamp block building -----------
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)
	sys.L2ChainA.CatchUpTo(sys.L2ChainB)
	sys.SuperRoots.EnsureInteropPaused(sys.L2CLA, sys.L2CLB, 10)

	sys.L2CLA.StopSequencer()
	sys.L2CLB.StopSequencer()

	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	// Synchronize chains to the same timestamp
	for range 10 {
		if unsafeA.Time == unsafeB.Time {
			break
		}
		if unsafeA.Time < unsafeB.Time {
			sys.L2CLA.StartSequencer()
			sys.L2ELA.WaitForTime(unsafeB.Time)
			sys.L2CLA.StopSequencer()
			unsafeA = sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		} else {
			sys.L2CLB.StartSequencer()
			sys.L2ELB.WaitForTime(unsafeA.Time)
			sys.L2CLB.StopSequencer()
			unsafeB = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		}
	}
	t.Require().Equal(unsafeA.Time, unsafeB.Time, "chains must be at same timestamp")

	blockTime := sys.L2ChainA.Escape().RollupConfig().BlockTime
	nextTimestamp := unsafeA.Time + blockTime

	setup := &intraBlockSetup{
		rng:               rng,
		alice:             alice,
		bob:               bob,
		eventLoggerA:      eventLoggerA,
		eventLoggerB:      eventLoggerB,
		expectedBlockNumA: unsafeA.Number + 1,
		expectedBlockNumB: unsafeB.Number + 1,
		nextTimestamp:     nextTimestamp,
	}

	// --- Build and include transactions in same-timestamp blocks -------------
	txsA, txsB := tc.BuildTxs(setup)

	// Assign deterministic nonces
	baseNonceA := alice.PendingNonce()
	for i, ptx := range txsA {
		txplan.WithStaticNonce(baseNonceA + uint64(i))(ptx)
	}
	baseNonceB := bob.PendingNonce()
	for i, ptx := range txsB {
		txplan.WithStaticNonce(baseNonceB + uint64(i))(ptx)
	}

	ctx := t.Ctx()
	var rawTxsA, rawTxsB [][]byte
	for _, ptx := range txsA {
		signedTx, err := ptx.Signed.Eval(ctx)
		t.Require().NoError(err, "failed to sign tx for chain A")
		rawBytes, err := signedTx.MarshalBinary()
		t.Require().NoError(err, "failed to marshal tx for chain A")
		rawTxsA = append(rawTxsA, rawBytes)
	}
	for _, ptx := range txsB {
		signedTx, err := ptx.Signed.Eval(ctx)
		t.Require().NoError(err, "failed to sign tx for chain B")
		rawBytes, err := signedTx.MarshalBinary()
		t.Require().NoError(err, "failed to marshal tx for chain B")
		rawTxsB = append(rawTxsB, rawBytes)
	}

	sys.TestSequencer.SequenceBlockWithTxs(t, sys.L2ChainA.ChainID(), unsafeA.Hash, rawTxsA)
	sys.TestSequencer.SequenceBlockWithTxs(t, sys.L2ChainB.ChainID(), unsafeB.Hash, rawTxsB)

	// --- Resume interop and wait for validation ------------------------------
	sys.SuperRoots.ResumeInterop()
	sys.SuperRoots.AwaitValidatedTimestamp(nextTimestamp)

	endTimestamp := nextTimestamp
	startTimestamp := endTimestamp - 1

	// --- Capture optimistic and cross-safe super roots -----------------------
	queryAPI := sys.SuperRoots.QueryAPI()
	firstOptimistic := optimisticBlockAtTimestamp(t, queryAPI, chains[0].ID, endTimestamp)
	secondOptimistic := optimisticBlockAtTimestamp(t, queryAPI, chains[1].ID, endTimestamp)

	start := superRootAtTimestamp(t, chains, startTimestamp)
	preConsolidation := marshalTransition(start, consolidateStep, firstOptimistic, secondOptimistic)

	crossSafeEnd := superRootAtTimestamp(t, chains, endTimestamp)
	optimisticEnd := eth.NewSuperV1(endTimestamp,
		eth.ChainIDAndOutput{ChainID: chains[0].ID, Output: firstOptimistic.OutputRoot},
		eth.ChainIDAndOutput{ChainID: chains[1].ID, Output: secondOptimistic.OutputRoot},
	)
	optimisticIsCrossSafe := bytes.Equal(optimisticEnd.Marshal(), crossSafeEnd.Marshal())

	l1HeadCurrent := latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp))

	// --- Build and run consolidation transition tests ------------------------
	tests := []*transitionTest{
		{
			Name:               "Consolidate",
			AgreedClaim:        preConsolidation,
			DisputedClaim:      crossSafeEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		},
		{
			Name:               "Consolidate-InvalidNoChange",
			AgreedClaim:        preConsolidation,
			DisputedClaim:      preConsolidation,
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		},
	}
	if !optimisticIsCrossSafe {
		tests = append(tests, &transitionTest{
			Name:               "Consolidate-ExpectInvalidPendingBlock",
			AgreedClaim:        preConsolidation,
			DisputedClaim:      optimisticEnd.Marshal(),
			DisputedTraceIndex: consolidateStep,
			L1Head:             l1HeadCurrent,
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        false,
		})
	}

	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	for _, test := range tests {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, queryAPI, gameDepth, startTimestamp, test.ClaimTimestamp, test)
		})
	}
}

// IntraBlockCases returns all intra-block test scenarios.
// CyclicDependencyInvalid uses a fabricated pending message approach to create a true
// exec→exec cycle, exercising kona's cycle detection codepath during consolidation.
func IntraBlockCases() []*IntraBlockTestCase {
	return []*IntraBlockTestCase{
		{
			// Init(A) + valid Exec(B→A) on chain A,
			// Init(B) + invalid Exec(A→B) on chain B.
			// Chain B is invalid (bad exec), chain A is transitively invalid.
			Name: "CascadeInvalid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pairA := s.prepareInitA(0)
				pairB := s.prepareInitB(0)
				return []*txplan.PlannedTx{pairA.SubmitInit(), pairB.SubmitExecTo(s.alice)},
					[]*txplan.PlannedTx{pairB.SubmitInit(), pairA.SubmitInvalidExecTo(s.bob)}
			},
		},
		{
			// Same as CascadeInvalid but with chains swapped.
			// Init(A) + invalid Exec(B→A) on chain A,
			// Init(B) + valid Exec(A→B) on chain B.
			Name: "SwapCascadeInvalid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pairA := s.prepareInitA(0)
				pairB := s.prepareInitB(0)
				return []*txplan.PlannedTx{pairA.SubmitInit(), pairB.SubmitInvalidExecTo(s.alice)},
					[]*txplan.PlannedTx{pairB.SubmitInit(), pairA.SubmitExecTo(s.bob)}
			},
		},
		{
			// Valid cyclic cross-chain dependency:
			// Init(A) + Exec(B→A) on chain A,
			// Init(B) + Exec(A→B) on chain B.
			// Both blocks survive.
			Name: "CyclicDependencyValid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pairA := s.prepareInitA(0)
				pairB := s.prepareInitB(0)
				return []*txplan.PlannedTx{pairA.SubmitInit(), pairB.SubmitExecTo(s.alice)},
					[]*txplan.PlannedTx{pairB.SubmitInit(), pairA.SubmitExecTo(s.bob)}
			},
		},
		{
			// True exec→exec cycle using fabricated pending messages.
			// ExecA references a fabricated init at B's position (wrong origin),
			// ExecB references ExecA's ExecutingMessage event.
			// Both are invalid: ExecA references wrong origin, ExecB depends on invalid ExecA.
			Name: "CyclicDependencyInvalid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				// Fabricate a pending init message at ExecB's expected position.
				// ExecA will reference this, but the actual log at (chainB, blockNumB, logIdx=0)
				// will be ExecB's ExecutingMessage event (wrong origin), making ExecA invalid.
				topic := crypto.Keccak256Hash([]byte("DataEmitted(bytes)"))
				msgHash := crypto.Keccak256Hash([]byte("fabricated cyclic msg"))
				var fabricatedPayload []byte
				fabricatedPayload = append(fabricatedPayload, topic.Bytes()...)
				fabricatedPayload = append(fabricatedPayload, msgHash.Bytes()...)

				fabricatedBMsg := messages.Message{
					Identifier: messages.Identifier{
						Origin:      s.eventLoggerB,
						BlockNumber: s.expectedBlockNumB,
						LogIndex:    0,
						Timestamp:   s.nextTimestamp,
						ChainID:     s.bob.ChainID(),
					},
					PayloadHash: crypto.Keccak256Hash(fabricatedPayload),
				}

				// Precompute ExecA's ExecutingMessage event message
				execAEventMsg := dsl.PrecomputeExecEventMessage(
					fabricatedBMsg, s.alice.ChainID(),
					s.expectedBlockNumA, 0, s.nextTimestamp,
				)

				// ExecA references fabricated message from B
				// ExecB references ExecA's event
				return []*txplan.PlannedTx{dsl.SubmitExecForMessage(fabricatedBMsg, s.alice)},
					[]*txplan.PlannedTx{dsl.SubmitExecForMessage(execAEventMsg, s.bob)}
			},
		},
		{
			// Depth-10 exec→exec dependency chain alternating between chains A and B.
			// Starts with an init on chain A, then 10 execs alternating B, A, B, A, ...
			// All valid.
			Name: "LongDependencyChainValid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pairA := s.prepareInitA(0)

				var txsA, txsB []*txplan.PlannedTx
				txsA = append(txsA, pairA.SubmitInit())

				currentMsg := pairA.Message
				logIdxA := uint32(1) // init occupies logIdx 0
				logIdxB := uint32(0)

				const depth = 10
				for i := 0; i < depth; i++ {
					if i%2 == 0 {
						// Exec on chain B referencing currentMsg
						execEventMsg := dsl.PrecomputeExecEventMessage(currentMsg, s.bob.ChainID(), s.expectedBlockNumB, logIdxB, s.nextTimestamp)
						txsB = append(txsB, dsl.SubmitExecForMessage(currentMsg, s.bob))
						currentMsg = execEventMsg
						logIdxB++
					} else {
						// Exec on chain A referencing currentMsg
						execEventMsg := dsl.PrecomputeExecEventMessage(currentMsg, s.alice.ChainID(), s.expectedBlockNumA, logIdxA, s.nextTimestamp)
						txsA = append(txsA, dsl.SubmitExecForMessage(currentMsg, s.alice))
						currentMsg = execEventMsg
						logIdxA++
					}
				}

				return txsA, txsB
			},
		},
		{
			// Same-chain valid: Init + Exec on chain A, empty chain B.
			Name: "SameChainValid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pair := s.prepareInitA(0)
				return []*txplan.PlannedTx{pair.SubmitInit(), pair.SubmitExecTo(s.alice)},
					nil
			},
		},
		{
			// Same-chain invalid: Init + invalid Exec on chain A, empty chain B.
			Name: "SameChainInvalid",
			BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
				pair := s.prepareInitA(0)
				return []*txplan.PlannedTx{pair.SubmitInit(), pair.SubmitInvalidExecTo(s.alice)},
					nil
			},
		},
	}
}
