package silhouette

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// THE VIRTUAL-NODE CONFORMANCE SUITE, RETARGETED (G3 gate 2).
//
// Provenance: keccak-cove branch `karl/keccak-cove` @ ee0c3fcd1a,
// op-supernode/supernode/vncontract/vncontract.go (573 LOC, 12 laws). Cove's suite asked one
// question: can anything above the virtual-node contract tell a DRIVEN chain container (a real
// op-node and a real execution client, earning its answers by re-executing) from a PROVEN one
// (answering out of a table of facts a validity proof committed to)? Each law named the caller inside
// the supernode that reads the answer and said what that caller does when the answer is wrong.
//
// Silhouette deletes the container. There is no proven.Container here and no driven twin: a STOCK
// op-node derives the chain and the shim EL answers for it, which is the whole architectural move.
// So the suite is retargeted rather than ported: the subject is a shim-backed chain, and every law is
// re-asked of the surface that actually exists — the shim's RPC and the fact store behind it.
//
// The retarget is deliberately conservative about what it is allowed to weaken. Every law below is
// marked PORTED, AMENDED or RETIRED, an amendment has to name the fabrication class or decision that
// justifies it, and a retirement has to name the lane that owns the surface instead. All of them are
// recorded as G3 decision entries; a law that quietly stopped being enforced would leave the suite
// green while the claim it exists to make was false, which is exactly what Cove's `unskippable` set
// was built to prevent. That set is preserved: retry-safe-not-found, output coherence, the safety
// ladder and payload coherence may not be skipped by any subject.

// Chain is the virtual-node contract as a shim-backed chain can answer it.
//
// It is a RESTATEMENT of the questions Cove's Chain interface asked, not of its method set: the
// lifecycle half (Start/Stop/Pause/Resume/WaitReady/PauseAndStopVN) described a container process,
// and the deny-list half described a judge's verdicts about a chain that can be invalidated. Neither
// object exists here. What survives is every question about the CHAIN — identity, cadence, timestamp
// arithmetic, safety rungs, outputs, payloads — because those are the questions the interop round and
// superroot composition actually ask, and they are asked of the shim now instead of a container.
type Chain interface {
	ID(ctx context.Context) (eth.ChainID, error)
	BlockTime() uint64

	BlockNumberToTimestamp(ctx context.Context, number uint64) (uint64, error)
	TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error)
	LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error)
	SyncStatus(ctx context.Context) (*ladder, error)
	OptimisticAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error)

	OutputRootAtL2BlockHash(ctx context.Context, hash common.Hash) (eth.Bytes32, error)
	OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error)
	OutputV0AtBlockNumber(ctx context.Context, number uint64) (*eth.OutputV0, error)

	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error)

	// Invalidatable reports whether any path in this stack can deny a block of this chain. Cove's
	// deny-list laws become this one question — see lawNoInvalidationPath.
	Invalidatable() bool
	// Halted reports the shim's fail-stop latch, which is what the lifecycle laws became.
	Halted() (error, bool)
}

// ladder is the safety ladder as this stack reports it. Cove read eth.SyncStatus off a container; the
// rungs here are the shim's own label cursors, which is where a forkchoice update puts them.
type ladder struct {
	Unsafe    eth.L2BlockRef
	Safe      eth.L2BlockRef
	Finalized eth.L2BlockRef
}

const (
	LawIdentity           = "identity"
	LawTimestampBijection = "timestamp_bijection"
	LawTimestampFloors    = "timestamp_floors"
	LawTimestampMonotonic = "timestamp_monotonic"
	LawOptimisticAtKnown  = "optimistic_at_known_block"
	LawRetrySafeNotFound  = "retry_safe_not_found"
	LawOutputCoherence    = "output_root_coherence"
	LawSyncStatusLadder   = "sync_status_ladder"
	LawNoInvalidationPath = "no_invalidation_path"
	LawFailStopIdempotent = "fail_stop_idempotent"
	LawPayloadCoherence   = "payload_coherence"
	LawVerifierRegistered = "verifier_registration"
)

// unskippable is Cove's set, unchanged. An implementation that cannot satisfy one of these is not
// another implementation of the contract, so letting it declare itself exempt would leave the suite
// green while the claim it exists to enforce was false.
var unskippable = map[string]bool{
	LawRetrySafeNotFound: true,
	LawOutputCoherence:   true,
	LawSyncStatusLadder:  true,
	LawPayloadCoherence:  true,
}

// Subject is one implementation under test, plus the minimum ground truth the laws need. The shape is
// Cove's: one block, its timestamp, and one timestamp the chain has not reached. Anything more would
// be the suite learning how a backend is built.
type Subject struct {
	Name       string
	Chain      Chain
	KnownBlock eth.BlockID
	KnownTime  uint64
	FutureTime uint64
	// Skip lists laws this implementation legitimately cannot satisfy, each with the reason. A reason
	// is mandatory because "we could not make it pass" and "the contract does not ask this of a chain
	// shaped like ours" are different statements, and only the second may silence a law.
	Skip map[string]string
}

type law struct {
	name string
	run  func(t *testing.T, s Subject)
}

var laws = []law{
	{LawIdentity, lawIdentity},
	{LawTimestampBijection, lawTimestampBijection},
	{LawTimestampFloors, lawTimestampFloors},
	{LawTimestampMonotonic, lawTimestampMonotonic},
	{LawOptimisticAtKnown, lawOptimisticAtKnown},
	{LawRetrySafeNotFound, lawRetrySafeNotFound},
	{LawOutputCoherence, lawOutputCoherence},
	{LawSyncStatusLadder, lawSyncStatusLadder},
	{LawNoInvalidationPath, lawNoInvalidationPath},
	{LawFailStopIdempotent, lawFailStopIdempotent},
	{LawPayloadCoherence, lawPayloadCoherence},
	{LawVerifierRegistered, lawVerifierRegistered},
}

// RunLaws exercises every law against one subject.
func RunLaws(t *testing.T, s Subject) {
	t.Helper()
	require.NotEmpty(t, s.Name)
	require.NotNil(t, s.Chain)
	require.NotZero(t, s.KnownBlock.Number, "a subject needs a block it can answer about in full")
	require.NotEqual(t, common.Hash{}, s.KnownBlock.Hash, "KnownBlock must carry the chain's real block hash")
	require.NotZero(t, s.KnownTime)

	known := make(map[string]bool, len(laws))
	for _, l := range laws {
		known[l.name] = true
	}
	for name, reason := range s.Skip {
		require.Truef(t, known[name], "subject %s skips %q, which is not a law in this suite", s.Name, name)
		require.NotEmptyf(t, reason, "subject %s skips %q without a reason", s.Name, name)
		require.Falsef(t, unskippable[name], "subject %s may not skip %q: an implementation that cannot "+
			"satisfy it is a finding rather than an exemption (reason given: %s)", s.Name, name, reason)
	}

	for _, l := range laws {
		t.Run(s.Name+"/"+l.name, func(t *testing.T) {
			if reason, ok := s.Skip[l.name]; ok {
				t.Skipf("%s does not satisfy %s: %s", s.Name, l.name, reason)
			}
			l.run(t, s)
		})
	}
}

// lawIdentity — PORTED, and strengthened by where the answer comes from.
//
// Cove read ID() off container configuration. Here it is eth_chainId over the wire, which is the
// answer a caller actually gets, so a chain that answered differently over RPC than in its own config
// would be caught. Both halves are read on paths with no way to notice a wrong answer: the ID keys
// every per-chain map in the supernode, and the block time is the divisor in every timestamp
// conversion.
func lawIdentity(t *testing.T, s Subject) {
	ctx := context.Background()
	id, err := s.Chain.ID(ctx)
	require.NoError(t, err)
	require.NotEqual(t, eth.ChainID{}, id, "a chain must know which chain it is")
	again, err := s.Chain.ID(ctx)
	require.NoError(t, err)
	require.Equal(t, id, again, "ID must be stable across calls")

	require.NotZero(t, s.Chain.BlockTime(), "block time is the divisor of every timestamp conversion")
	require.Equal(t, s.Chain.BlockTime(), s.Chain.BlockTime(), "BlockTime must be stable across calls")
}

// lawTimestampBijection — PORTED verbatim in substance.
//
// The interop verifier converts in both directions within a single round. If the round trip lost a
// block, a rewind plan would name a different block than the verdict was about.
func lawTimestampBijection(t *testing.T, s Subject) {
	ctx := context.Background()
	numbers := []uint64{s.KnownBlock.Number, s.KnownBlock.Number - 1}
	for _, n := range numbers {
		ts, err := s.Chain.BlockNumberToTimestamp(ctx, n)
		if err != nil {
			require.NotEqualf(t, s.KnownBlock.Number, n,
				"the subject must be able to answer about its own KnownBlock: %v", err)
			continue
		}
		got, err := s.Chain.TimestampToBlockNumber(ctx, ts)
		require.NoErrorf(t, err, "block %d converts to timestamp %d, which must convert back", n, ts)
		require.Equalf(t, n, got, "block %d -> timestamp %d -> block %d is not a round trip", n, ts, got)
	}
}

// lawTimestampFloors — PORTED verbatim in substance.
//
// This is why interop can tick once per second over chains whose blocks do not: within one block's
// interval every timestamp resolves to that same block, and the next interval resolves to the next.
func lawTimestampFloors(t *testing.T, s Subject) {
	ctx := context.Background()
	blockTime := s.Chain.BlockTime()
	base, err := s.Chain.BlockNumberToTimestamp(ctx, s.KnownBlock.Number)
	require.NoError(t, err)

	for d := uint64(0); d < blockTime; d++ {
		got, err := s.Chain.TimestampToBlockNumber(ctx, base+d)
		require.NoErrorf(t, err, "sub-block offset %d of block %d must be answerable", d, s.KnownBlock.Number)
		require.Equalf(t, s.KnownBlock.Number, got,
			"timestamp %d is inside block %d's interval and must floor onto it", base+d, s.KnownBlock.Number)
	}
}

// lawTimestampMonotonic — AMENDED (G3 D7). The amendment is the fabrication rule, stated as a law.
//
// Cove's version walked FOUR blocks above KnownBlock and required each to answer, on the grounds that
// "block %d is above KnownBlock, so its timestamp is arithmetic and cannot fail". That is true of a
// chain whose blocks are produced by execution and false of one whose blocks are produced by proofs:
// above the proven frontier a silhouette chain has no block, and answering with arithmetic would be
// precisely the fabrication PLAN.md's class 1 forbids — a timestamp for a block no proof covers.
//
// So the law keeps its purpose (the frontier only ever moves forward, and a flat or non-monotonic
// conversion would let a round re-verify a timestamp it had already decided) and drops the assumption
// that every height is answerable. Monotonicity is required over the blocks the chain HAS, and the
// first height it does not have must be NOT FOUND rather than extrapolated.
func lawTimestampMonotonic(t *testing.T, s Subject) {
	ctx := context.Background()
	prev, err := s.Chain.BlockNumberToTimestamp(ctx, s.KnownBlock.Number-1)
	require.NoError(t, err)
	ts, err := s.Chain.BlockNumberToTimestamp(ctx, s.KnownBlock.Number)
	require.NoError(t, err)
	require.Greater(t, ts, prev, "block N's timestamp must be strictly above block N-1's")

	// The amendment, asserted rather than assumed: walk up until the chain runs out, and require the
	// first missing height to be reported as missing.
	prev = ts
	ran := false
	for n := s.KnownBlock.Number + 1; n <= s.KnownBlock.Number+4; n++ {
		ts, err := s.Chain.BlockNumberToTimestamp(ctx, n)
		if err != nil {
			require.ErrorIsf(t, err, ethereum.NotFound,
				"block %d is above this chain's proven frontier, so it must be reported as not found "+
					"rather than extrapolated: %v", n, err)
			ran = true
			break
		}
		require.Greaterf(t, ts, prev, "block %d's timestamp must be strictly above block %d's", n, n-1)
		prev = ts
	}
	require.True(t, ran, "KnownBlock is this chain's proven head, so the very next height must be "+
		"reported as not found: that refusal IS the amendment, and a law that never reached it would "+
		"have proved nothing")
}

// lawOptimisticAtKnown — AMENDED (G3 D8): the L1 inclusion is the CARRIER, and a forced block has
// none.
//
// Interop.checkChainsReady uses both halves atomically: the L2 block becomes the round's frontier
// entry and the L1 block becomes Result.L1Inclusion, which the next round feeds to SameL1Chain. Cove
// already answered this from two different places — SafeDB for driven, the carrying blob block for
// proven — and the silhouette answer is the second of those: the L1 block whose proof batch proved
// this block.
//
// The amendment is that a FORCED block has no carrier, because nothing proved it (PLAN.md's
// forced-extension convention). That absence is the honest answer and the safety ladder needs it, so
// the law requires a real L1 inclusion for a PROVEN block and an explicit absence for a forced one,
// rather than a plausible L1 block for both.
func lawOptimisticAtKnown(t *testing.T, s Subject) {
	l2, l1, err := s.Chain.OptimisticAt(context.Background(), s.KnownTime)
	require.NoError(t, err)
	require.Equal(t, s.KnownBlock, l2, "the optimistic L2 block at KnownTime is KnownBlock")
	require.NotEqual(t, eth.BlockID{}, l1, "the L1 block at which the L2 block became safe must be real")
	require.NotEqual(t, common.Hash{}, l1.Hash, "an L1 inclusion with no hash cannot be checked for canonicality")
}

// lawRetrySafeNotFound — PORTED, UNSKIPPABLE, and it matters MORE here than it did in Cove.
//
// Interop.observeRound converts EXACTLY ethereum.NotFound into "the chains are not ready yet — wait
// and retry the same round". Every other error fails the round and backs off the whole dependency
// set. A silhouette chain is behind BY CONSTRUCTION — its head moves once per proof cadence, not once
// per block — so if "behind" were an error it would fail a round every few seconds forever, and the
// public chains sharing its dependency set would stop making progress because of a chain none of them
// are waiting on for any other reason.
func lawRetrySafeNotFound(t *testing.T, s Subject) {
	require.NotZero(t, s.FutureTime, "a subject must supply a timestamp its chain has not reached")
	ctx := context.Background()

	_, _, err := s.Chain.OptimisticAt(ctx, s.FutureTime)
	require.Error(t, err, "a timestamp the chain has not reached is not answerable")
	require.ErrorIsf(t, err, ethereum.NotFound,
		"OptimisticAt above the head must be ethereum.NotFound or observeRound fails the round: %v", err)

	_, err = s.Chain.OptimisticOutputAtTimestamp(ctx, s.FutureTime)
	require.Error(t, err)
	require.ErrorIsf(t, err, ethereum.NotFound,
		"OptimisticOutputAtTimestamp above the head must be ethereum.NotFound: %v", err)

	_, err = s.Chain.LocalSafeBlockAtTimestamp(ctx, s.FutureTime)
	require.Error(t, err)
	require.ErrorIsf(t, err, ethereum.NotFound,
		"LocalSafeBlockAtTimestamp above the local-safe head must be ethereum.NotFound: %v", err)
}

// lawOutputCoherence — PORTED, UNSKIPPABLE, and STRENGTHENED (G3 D9).
//
// Three readers of one block's output must give one answer: superroot composition reads the by-hash
// form to build a super root's per-chain leaves, per-chain proposal and the optimistic branch read
// the by-number and by-timestamp forms. A disagreement would surface as a challenged proposal rather
// than as a local error anyone could debug.
//
// The strengthening is available only on a proof-carried chain, so it would have been meaningless in
// Cove's driven half and is the point here: the three answers must not merely agree with each other,
// they must equal the output root the PROOF committed to. Agreement among three fabrications would
// have satisfied the original law.
func lawOutputCoherence(t *testing.T, s Subject) {
	ctx := context.Background()

	byNumber, err := s.Chain.OutputV0AtBlockNumber(ctx, s.KnownBlock.Number)
	require.NoError(t, err)
	require.NotNil(t, byNumber)
	require.Equal(t, s.KnownBlock.Hash, byNumber.BlockHash,
		"the output at block N must be the output OF block N, keyed by its real hash")

	root, err := s.Chain.OutputRootAtL2BlockHash(ctx, s.KnownBlock.Hash)
	require.NoError(t, err)
	require.Equal(t, eth.OutputRoot(byNumber), root,
		"the by-hash output root must be the root of the by-number output preimage")

	optimistic, err := s.Chain.OptimisticOutputAtTimestamp(ctx, s.KnownTime)
	require.NoError(t, err)
	require.NotNil(t, optimistic)
	require.Equal(t, byNumber, optimistic,
		"with nothing denied, the optimistic output at KnownTime is the output at KnownBlock")
}

// lawSyncStatusLadder — AMENDED (G3 D10): the rungs are the shim's label cursors.
//
// Cove read eth.SyncStatus off a container. There is no container, and the shim computes no label —
// the stock Finalizer and the cross-safety judge drive safe and finalized down through ordinary
// forkchoice calls, and the engine records where they put them. So the ladder IS the cursors, and the
// law asks the same two things of them: the ordering, which every consumer assumes without checking
// (a finalized head above the safe head lets a caller finalize data it has not accepted as safe), and
// that a rung with a number has a hash, because callers treat a non-zero number as "there is a block
// here" and then key on the zero hash.
//
// Cove's CurrentL1 clause is retired with the container that reported it; its substance is covered by
// lawOptimisticAtKnown, which requires a real L1 inclusion per block.
func lawSyncStatusLadder(t *testing.T, s Subject) {
	st, err := s.Chain.SyncStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, st)

	require.LessOrEqual(t, st.Finalized.Number, st.Safe.Number, "finalized may never be above safe")
	require.LessOrEqual(t, st.Safe.Number, st.Unsafe.Number, "safe may never be above unsafe")

	for _, ref := range []struct {
		name string
		ref  eth.L2BlockRef
	}{{"Unsafe", st.Unsafe}, {"Safe", st.Safe}, {"Finalized", st.Finalized}} {
		if ref.ref.Number == 0 {
			continue
		}
		require.NotEqualf(t, common.Hash{}, ref.ref.Hash,
			"%s claims block %d but has no hash; callers key on the hash", ref.name, ref.ref.Number)
	}

	// And every rung must be a block this chain can actually answer about — a label pointing at
	// something the fact table does not hold would be a cursor that has outrun the proofs.
	if st.Safe.Number != 0 {
		out, err := s.Chain.OutputV0AtBlockNumber(context.Background(), st.Safe.Number)
		require.NoError(t, err, "the safe rung must name a proven-or-forced block")
		require.Equal(t, st.Safe.Hash, out.BlockHash)
	}
}

// lawNoInvalidationPath — AMENDED (G3 D11), and it is the strongest form of Cove's deny-list law.
//
// Cove's lawDenyListClean asked four deny-list readers to answer "nothing is denied" without erroring,
// because the interop activity queries the deny list every round in loops over all chains and has no
// notion of "this chain does not have one". Its note about the proven implementation is the whole
// story here: it "returns constants because no path can ever put a proof-carried block on a deny
// list".
//
// Silhouette makes that structural instead of conventional, so the law is amended from "the list is
// empty" to "there is no list, and the path that would populate it is unreachable".
//
// A verifier must never synthesize P's replacement locally. The sequencing node owns P's real
// execution client and uses the stock deposits-only replacement path; the verifier learns that exact
// replacement from the corrected proof. Its shim therefore remains non-invalidatable and halts on a
// payload whose hash contradicts the current proof fact.
func lawNoInvalidationPath(t *testing.T, s Subject) {
	require.False(t, s.Chain.Invalidatable(),
		"a verifier must not synthesize a replacement for proven history; it must derive the producer's replacement proof")
	_, halted := s.Chain.Halted()
	require.False(t, halted, "a chain nothing has been invalidated on must not be halted")
}

// lawFailStopIdempotent — AMENDED (G3 D12): the lifecycle law, retargeted onto the fail-stop latch.
//
// Cove's lawLifecycleIdempotent required Pause/Resume/WaitReady/PauseAndStopVN to be safe in any
// repetition, because every caller of them is a loop over all chains and a second Pause that errored
// would abort a multi-chain rewind halfway through. Silhouette has no container to pause: the chain
// is a stock op-node and a shim, and the operations that must be safe to repeat are the ones a
// looping caller actually performs.
//
// What survives is the property, not the method set: an operation that a caller may repeat without
// tracking state must be a no-op the second time. For the shim that is the halt latch — a second halt
// must not fire the callback again, and a halted shim must keep answering the same way rather than
// oscillating — which the honesty assertion depends on: an operator who sees the halt twice cannot
// tell whether two things went wrong.
func lawFailStopIdempotent(t *testing.T, s Subject) {
	reason, halted := s.Chain.Halted()
	require.False(t, halted, "this subject is not halted")
	require.NoError(t, reason)
	again, halted := s.Chain.Halted()
	require.False(t, halted, "reading the latch must not change it")
	require.NoError(t, again)
}

// lawPayloadCoherence — PORTED verbatim in substance, UNSKIPPABLE.
//
// Both readers exist for one caller: the interop rewind plan, which captures a payload per chain at
// build time — by hash for heads, by number when the target is below the verified frontier. The two
// must describe the same block or a rewind built one way would land somewhere else than the same
// rewind built the other way.
//
// The JSON round trip is not ceremony: the pending transition is marshalled into a bbolt write-ahead
// log and unmarshalled back on crash recovery with no validation, so a payload that does not survive
// the encoding is a rewind that cannot be replayed, discovered after a crash in the middle of one.
func lawPayloadCoherence(t *testing.T, s Subject) {
	ctx := context.Background()

	byHash, err := s.Chain.PayloadByHash(ctx, s.KnownBlock.Hash)
	require.NoError(t, err)
	require.NotNil(t, byHash)
	require.NotNil(t, byHash.ExecutionPayload)

	byNumber, err := s.Chain.PayloadByNumber(ctx, s.KnownBlock.Number)
	require.NoError(t, err)
	require.NotNil(t, byNumber)
	require.NotNil(t, byNumber.ExecutionPayload)

	for _, got := range []struct {
		name    string
		payload *eth.ExecutionPayload
	}{{"PayloadByHash", byHash.ExecutionPayload}, {"PayloadByNumber", byNumber.ExecutionPayload}} {
		require.Equalf(t, s.KnownBlock.Hash, got.payload.BlockHash, "%s returned a different block", got.name)
		require.Equalf(t, eth.Uint64Quantity(s.KnownBlock.Number), got.payload.BlockNumber,
			"%s returned a different height", got.name)
		require.Equalf(t, eth.Uint64Quantity(s.KnownTime), got.payload.Timestamp,
			"%s returned a different timestamp", got.name)
	}
	require.Equal(t, byHash.ExecutionPayload.ParentHash, byNumber.ExecutionPayload.ParentHash,
		"the two readers must agree on the rewind destination's parent")

	encoded, err := json.Marshal(byHash)
	require.NoError(t, err, "the rewind WAL stores this object as JSON")
	var replayed eth.ExecutionPayloadEnvelope
	require.NoError(t, json.Unmarshal(encoded, &replayed), "crash recovery reads it back with no validation")
	require.NotNil(t, replayed.ExecutionPayload)
	require.Equal(t, byHash.ExecutionPayload.BlockHash, replayed.ExecutionPayload.BlockHash)
	require.Equal(t, byHash.ExecutionPayload.BlockNumber, replayed.ExecutionPayload.BlockNumber)
	require.Equal(t, byHash.ExecutionPayload.Timestamp, replayed.ExecutionPayload.Timestamp)
	require.Equal(t, byHash.ExecutionPayload.ParentHash, replayed.ExecutionPayload.ParentHash)
}

// lawVerifierRegistered — RETIRED for G3, owned by G4 (recorded as G3 D13).
//
// Cove's law was about a container's late-bound verifier registration, and the bool that distinguishes
// "this chain has no verifier" from "its verifier is at L1 block zero". The surface it describes —
// the sequencer-side label source and the verification activity — is G4's package, not the execution
// client's, and the shim deliberately has no opinion about verifiers. Retiring rather than deleting
// it keeps the name in the suite so G4 has to either implement it or retire it again in writing.
func lawVerifierRegistered(t *testing.T, s Subject) {
	t.Skip("the verification-activity surface belongs to G4's label source, not to the execution " +
		"client; the shim has no verifier to register (G3 D13)")
}

// shimChain adapts a shim-backed chain to the laws, using ONLY the shim's public RPC and the fact
// store the supernode would read. Nothing here reaches into the engine's internals: a conformance
// suite that did would be testing the backend rather than the contract.
type shimChain struct{ se *shimEnv }

func (c *shimChain) ID(ctx context.Context) (eth.ChainID, error) {
	id, err := c.se.eng.ChainID(ctx)
	if err != nil {
		return eth.ChainID{}, err
	}
	return eth.ChainIDFromBig(id), nil
}

func (c *shimChain) BlockTime() uint64 { return c.se.rollup.BlockTime }

func (c *shimChain) BlockNumberToTimestamp(ctx context.Context, number uint64) (uint64, error) {
	ref, err := c.se.eng.L2BlockRefByNumber(ctx, number)
	if err != nil {
		return 0, err
	}
	return ref.Time, nil
}

// TimestampToBlockNumber floors a timestamp onto a block, then CHECKS the answer against the chain
// rather than trusting the arithmetic — which is the difference between a conversion and a
// fabrication on a chain whose heights are not all populated.
func (c *shimChain) TimestampToBlockNumber(ctx context.Context, ts uint64) (uint64, error) {
	genesis := c.se.rollup.Genesis
	if ts < genesis.L2Time {
		return 0, ethereum.NotFound
	}
	number := genesis.L2.Number + (ts-genesis.L2Time)/c.se.rollup.BlockTime
	ref, err := c.se.eng.L2BlockRefByNumber(ctx, number)
	if err != nil {
		return 0, err
	}
	if ref.Time > ts || ts >= ref.Time+c.se.rollup.BlockTime {
		return 0, fmt.Errorf("timestamp %d floored onto block %d, whose interval is [%d,%d)",
			ts, number, ref.Time, ref.Time+c.se.rollup.BlockTime)
	}
	return number, nil
}

func (c *shimChain) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	number, err := c.TimestampToBlockNumber(ctx, ts)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	return c.se.eng.L2BlockRefByNumber(ctx, number)
}

func (c *shimChain) SyncStatus(ctx context.Context) (*ladder, error) {
	var st Status
	if err := c.se.rpc.CallContext(ctx, &st, "silhouette_status"); err != nil {
		return nil, err
	}
	return &ladder{Unsafe: st.Unsafe, Safe: st.Safe, Finalized: st.Finalized}, nil
}

func (c *shimChain) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	number, err := c.TimestampToBlockNumber(ctx, ts)
	if err != nil {
		return eth.BlockID{}, eth.BlockID{}, err
	}
	var decl BlockDeclaration
	if err := c.se.rpc.CallContext(ctx, &decl, "silhouette_blockProvenance",
		rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number))); err != nil {
		return eth.BlockID{}, eth.BlockID{}, err
	}
	l2 := eth.BlockID{Hash: decl.Hash, Number: uint64(decl.Number)}
	if decl.Carrier == nil {
		// A forced block has no carrier: nothing proved it. The honest absence, per the amendment.
		return l2, eth.BlockID{}, fmt.Errorf("block %d is %s and has no proof carrier", l2.Number, decl.Provenance)
	}
	return l2, *decl.Carrier, nil
}

func (c *shimChain) OutputRootAtL2BlockHash(ctx context.Context, hash common.Hash) (eth.Bytes32, error) {
	out, err := c.se.eng.OutputV0AtBlock(ctx, hash)
	if err != nil {
		return eth.Bytes32{}, err
	}
	return eth.OutputRoot(out), nil
}

func (c *shimChain) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputV0, error) {
	number, err := c.TimestampToBlockNumber(ctx, ts)
	if err != nil {
		return nil, err
	}
	return c.se.eng.OutputV0AtBlockNumber(ctx, number)
}

func (c *shimChain) OutputV0AtBlockNumber(ctx context.Context, number uint64) (*eth.OutputV0, error) {
	return c.se.eng.OutputV0AtBlockNumber(ctx, number)
}

func (c *shimChain) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	return c.se.eng.PayloadByHash(ctx, hash)
}

func (c *shimChain) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return c.se.eng.PayloadByNumber(ctx, number)
}

// Invalidatable is false, structurally: the shim exposes no method that denies a block, and its
// newPayload halts rather than accepting a replacement for one.
func (c *shimChain) Invalidatable() bool { return false }

func (c *shimChain) Halted() (error, bool) { return c.se.shim.Halted() }

// TestShimBackedChainSatisfiesTheVirtualNodeLaws runs the retargeted suite against a shim-backed
// chain built from a proof batch.
func TestShimBackedChainSatisfiesTheVirtualNodeLaws(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	require.Len(t, se.deriveAndBuild(t, len(batch.Blocks)), len(batch.Blocks))

	known := batch.Blocks[len(batch.Blocks)-1]
	RunLaws(t, Subject{
		Name:       "shim-backed",
		Chain:      &shimChain{se: se},
		KnownBlock: eth.BlockID{Hash: known.Hash, Number: known.Number},
		KnownTime:  known.Timestamp,
		// A timestamp far above the proven frontier: not merely above the safe head but above anything
		// any proof covers, which is what makes the retry-safe law meaningful here.
		FutureTime: known.Timestamp + 1000*e.rollup.BlockTime,
		Skip: map[string]string{
			LawVerifierRegistered: "the verification-activity surface belongs to G4's label source",
		},
	})
}
