package interop

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
)

// These tests pin the difference between the two ways verification can fail, at the layer where it
// decides what to DO about them.
//
// A proof-carried chain lags by a whole proof interval by design, so a driven chain's block can
// legitimately reference an initiating message the verifier has not received yet. If the judge
// reads that as "invalid" it deny-lists and replaces a perfectly good block, and the node forks off
// the canonical chain permanently — over data that was simply late. The correct behaviour is a
// future reference waits, a conflicting one invalidates.
//
// Because a proof-carried chain is a full MEMBER of the dependency set, the verifier's own
// readiness check ALSO gates every round on that chain having reached the round's timestamp, so a
// round that would have waited at the judge often never runs at all. The judge's rule is still
// there and still correct; it has simply gone latent for this chain in the common case.
// TestFrontierReadinessAlsoProvidesTheWait is the proof of that claim, while the rest of this file
// keeps the rule itself under test — it is an upstream asset and it is what makes the stack correct
// under races, not only under the happy ordering.

const (
	waitTestActivation = uint64(1000)
	waitTestBlockTime  = uint64(1)
	// waitTestDrivenBlock is the driven chain block carrying the executing message.
	waitTestDrivenBlock = uint64(5)
	// waitTestInitBlock / waitTestInitLogIdx locate the initiating message on the proven chain.
	waitTestInitBlock  = uint64(1)
	waitTestInitLogIdx = uint32(0)
)

// provenOrigin is the contract on P that emitted the initiating message.
var provenOrigin = common.Address{0xaa, 0xbb}

// encodeExecutingMessageLog builds a real CrossL2Inbox ExecutingMessage log — the inverse of
// messages.Message.DecodeEvent — so the fixture's executing message is decoded by production code
// rather than injected as a struct.
func encodeExecutingMessageLog(id messages.Identifier, payloadHash common.Hash) *types.Log {
	data := make([]byte, 0, 32*5)
	data = append(data, make([]byte, 12)...)
	data = append(data, id.Origin.Bytes()...)
	data = append(data, make([]byte, 24)...)
	data = binary.BigEndian.AppendUint64(data, id.BlockNumber)
	data = append(data, make([]byte, 28)...)
	data = binary.BigEndian.AppendUint32(data, id.LogIndex)
	data = append(data, make([]byte, 24)...)
	data = binary.BigEndian.AppendUint64(data, id.Timestamp)
	chainID := id.ChainID.Bytes32()
	data = append(data, chainID[:]...)
	return &types.Log{
		Address: predeploys.CrossL2InboxAddr,
		Topics:  []common.Hash{messages.ExecutingMessageEventTopic, payloadHash},
		Data:    data,
	}
}

// fakeProvenChain is a proof-carried chain container: its blocks become known only when a proof
// batch is accepted, and its logs are sealed by the REAL silhouette log sink rather than by
// anything that reads receipts.
//
// It exists to make the wait real. A mock that simply returned ErrFuture would test the switch
// statement; this drives the actual database through the actual ingestion path, so what the judge
// sees is what a live verifier sees between proof batches.
type fakeProvenChain struct {
	*algoMockChain
	sink *silhouette.LogSink
	// blocks maps block number -> the wire facts a proof committed to, populated by AcceptBatch.
	blocks map[uint64]proofbatch.BlockExport
	// carriers maps block number -> the L1 block that carried its proof.
	carriers map[uint64]eth.BlockID
	// imports maps block number -> the executing messages the wire declared for it, mirroring
	// silhouette.Fact.ExecMsgs. This is the G7 import list as the judge consumes it.
	imports map[uint64][]messages.ExecutingMessage
	// importsOnWire is the wire version's answer to "does this chain declare its imports at all".
	// True is v3 and above; false reproduces the retired pre-v3 posture in which the chain's
	// dependencies were proof-trusted, which is worth keeping testable because the rotation runs
	// both.
	importsOnWire bool
	receiptsCall  int
	invalidations int
}

func newFakeProvenChain(t *testing.T, chainID eth.ChainID, store silhouette.LogStore) *fakeProvenChain {
	t.Helper()
	return &fakeProvenChain{
		algoMockChain: &algoMockChain{id: chainID, blockTimeOverride: waitTestBlockTime},
		sink:          silhouette.NewLogSink(testlog.Logger(t, gethlog.LevelInfo), store),
		blocks:        make(map[uint64]proofbatch.BlockExport),
		carriers:      make(map[uint64]eth.BlockID),
		imports:       make(map[uint64][]messages.ExecutingMessage),
		importsOnWire: true,
	}
}

// ProvenExecMsgs is the G7 capability: the import list, keyed by the wire's SET ORDINAL.
func (c *fakeProvenChain) ProvenExecMsgs(blockNum uint64) (map[uint32]*messages.ExecutingMessage, bool, error) {
	if _, ok := c.blocks[blockNum]; !ok {
		// Fail closed, exactly as the production container does: no facts means this node cannot say
		// what the block imported, which is a different statement from "it imported nothing".
		return nil, false, fmt.Errorf("chain %s: no facts for block %d", c.id, blockNum)
	}
	if !c.importsOnWire {
		return nil, false, nil
	}
	list := c.imports[blockNum]
	out := make(map[uint32]*messages.ExecutingMessage, len(list))
	for i := range list {
		msg := list[i]
		out[uint32(i)] = &msg
	}
	return out, true, nil
}

func (c *fakeProvenChain) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32, parentPayload *eth.ExecutionPayloadEnvelope) (bool, error) {
	c.invalidations++
	return false, nil
}

func (c *fakeProvenChain) Invalidations() int { return c.invalidations }

// OutputV0AtBlockNumber answers from the WIRE FACTS, which is what the production container does: a
// proof-carried chain's roots are what a proof committed to, never a re-execution. The judge needs
// it to build an InvalidHead, so under G7 it is on a path a test can actually reach.
func (c *fakeProvenChain) OutputV0AtBlockNumber(ctx context.Context, num uint64) (*eth.OutputV0, error) {
	blk, ok := c.blocks[num]
	if !ok {
		return nil, ethereum.NotFound
	}
	return &eth.OutputV0{
		StateRoot:                eth.Bytes32(blk.StateRoot),
		MessagePasserStorageRoot: eth.Bytes32(blk.MessagePasserStorageRoot),
		BlockHash:                blk.Hash,
	}, nil
}

// IngestionSource is the capability that routes the interop activity around this chain's receipts.
func (c *fakeProvenChain) IngestionSource() cc.IngestionSource { return cc.IngestionProven }

// FetchReceipts refuses, and counts. Receipts are execution output; a proof commits to the message
// hashes a block exported and to nothing else about its transactions. Every path that would reach
// this is gated on IngestionSource, so a non-zero count is a test failure and the refusal turns
// "no path reaches it" from an argument into an assertion.
func (c *fakeProvenChain) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	c.receiptsCall++
	return nil, nil, ethereum.NotFound
}

func (c *fakeProvenChain) ReceiptsAsked() int { return c.receiptsCall }

// OptimisticAt is what the verifier's readiness check asks. A proof-carried chain answers only for
// timestamps a proof covers, and its L1 inclusion is the L1 block that carried that proof.
func (c *fakeProvenChain) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	for num, blk := range c.blocks {
		if blk.Timestamp == ts {
			return eth.BlockID{Number: num, Hash: blk.Hash}, c.carriers[num], nil
		}
	}
	return eth.BlockID{}, eth.BlockID{}, ethereum.NotFound
}

func (c *fakeProvenChain) blockHash(num uint64) common.Hash { return c.blocks[num].Hash }

// PayloadHash is the payload the initiating log at (block, logIdx) hashes from. The driven chain's
// executing message carries it, and the proven chain's exported log hash derives from it, so the
// two agree by construction rather than by a copied constant.
func (c *fakeProvenChain) PayloadHash(block uint64, logIdx uint32) common.Hash {
	return crypto.Keccak256Hash([]byte("initiating-message"))
}

// AcceptBatch is "the proof batch covering P block `num` lands on L1": it seals the block's
// exported logs through the production sink, under the block's real wire hash.
func (c *fakeProvenChain) AcceptBatch(t *testing.T, num uint64, carrier eth.BlockID) {
	t.Helper()
	c.AcceptBatchWithImports(t, num, carrier)
}

// AcceptBatchWithImports is the same event for a block that CONSUMED cross-chain messages: the
// import list travels on the wire beside the export list, and the judge validates it.
//
// The batch is put through the real proofbatch.CheckStructure before it is accepted, so a test
// cannot hand the judge an import list the codec would have refused — which is the whole basis for
// the claim that a proven chain contributes no same-timestamp executing messages.
func (c *fakeProvenChain) AcceptBatchWithImports(t *testing.T, num uint64, carrier eth.BlockID, imports ...proofbatch.ExecMsg) {
	t.Helper()
	logHash := messages.PayloadHashToLogHash(c.PayloadHash(num, waitTestInitLogIdx), provenOrigin)
	blk := proofbatch.BlockExport{
		Number:    num,
		Timestamp: waitTestActivation + num*waitTestBlockTime,
		Hash:      common.BytesToHash([]byte{byte(num), 0x9e, 0x11}),
		Logs:      []proofbatch.LogExport{{Index: waitTestInitLogIdx, Hash: logHash}},
		ExecMsgs:  imports,
	}
	batch := &proofbatch.ProofBatch{Blocks: []proofbatch.BlockExport{blk}}
	require.NoError(t, batch.CheckStructure(), "the test built a batch the wire codec would refuse")
	require.NoError(t, batch.CheckNoSameTimestampImports(),
		"the test built a batch a verifier would refuse at acceptance")
	parent := common.HexToHash("0x9e00")
	if prev, ok := c.blocks[num-1]; ok {
		parent = prev.Hash
	}
	require.NoError(t, c.sink.Accept([]proofbatch.BlockExport{blk}, parent))
	c.blocks[num] = blk
	c.carriers[num] = carrier
	list := make([]messages.ExecutingMessage, len(imports))
	for i := range imports {
		list[i] = *imports[i].Executing()
	}
	c.imports[num] = list
}

// drivenChain is chain A: an ordinary, healthy, executed chain. Its receipts are real, because the
// frontier verification view builds a driven chain's executing messages out of them — the view is
// the path a full round actually takes for A, as distinct from the LogsDB path a decided block
// takes later.
type drivenChain struct {
	*algoMockChain
	info *mockBlockInfo
	logs []*types.Log
}

func (d *drivenChain) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	return d.info, optypes.Receipts{{Receipt: types.Receipt{Logs: d.logs}}}, nil
}

// laggingSourceFixture is a driven chain A whose block references an initiating message on a
// proof-carried chain P, with P's proof cadence under the test's control.
type laggingSourceFixture struct {
	t        *testing.T
	interop  *Interop
	proven   *fakeProvenChain
	drivenID eth.ChainID
	provenID eth.ChainID
	blocks   blockPerChain
	l1Heads  blockPerChain
	ts       uint64
}

// newLaggingSourceFixture seals a real CrossL2Inbox executing-message log into A's LogsDB through
// the production processBlockLogs path, referencing P block 1 log 0. P's LogsDB starts EMPTY: no
// proof batch has landed yet, which is exactly the live-edge state a verifier is in between
// batches.
func newLaggingSourceFixture(t *testing.T, payloadHash common.Hash) *laggingSourceFixture {
	t.Helper()
	provenID := eth.ChainIDFromUInt64(424247)
	drivenID := eth.ChainIDFromUInt64(424246)

	provenDB, err := openLogsDB(testLogger(), provenID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = provenDB.Close() })
	chain := newFakeProvenChain(t, provenID, provenDB)

	drivenDB, err := openLogsDB(testLogger(), drivenID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = drivenDB.Close() })

	initTimestamp := waitTestActivation + waitTestBlockTime
	if payloadHash == (common.Hash{}) {
		payloadHash = chain.PayloadHash(waitTestInitBlock, waitTestInitLogIdx)
	}
	logEntry := encodeExecutingMessageLog(messages.Identifier{
		Origin:      provenOrigin,
		BlockNumber: waitTestInitBlock,
		LogIndex:    waitTestInitLogIdx,
		Timestamp:   initTimestamp,
		ChainID:     provenID,
	}, payloadHash)

	drivenHash := common.HexToHash("0xd0e5")
	l1Block := eth.BlockID{Number: 40, Hash: common.HexToHash("0x1")}
	drivenMock := newMockChainWithL1(drivenID, l1Block, eth.BlockID{Number: waitTestDrivenBlock, Hash: drivenHash})
	drivenMock.optimisticL2 = eth.BlockID{Number: waitTestDrivenBlock, Hash: drivenHash}
	drivenInfo := &mockBlockInfo{
		hash:       drivenHash,
		parentHash: common.HexToHash("0xd0e4"),
		number:     waitTestDrivenBlock,
		timestamp:  initTimestamp,
	}
	drivenChain := &drivenChain{algoMockChain: drivenMock, info: drivenInfo, logs: []*types.Log{logEntry}}

	verifiedDB, err := OpenVerifiedDB(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = verifiedDB.Close() })

	i := &Interop{
		log:                 testLogger(),
		ctx:                 context.Background(),
		activationTimestamp: waitTestActivation,
		messageExpiryWindow: defaultMessageExpiryWindow,
		logsDBs:             map[eth.ChainID]LogsDB{drivenID: drivenDB, provenID: provenDB},
		chains:              map[eth.ChainID]cc.InteropChain{drivenID: drivenChain, provenID: chain},
		verifiedDB:          verifiedDB,
		l1Checker:           noopL1Checker{},
		metrics:             resources.NewSupernodeMetrics(),

		verificationStartTimestamp: initTimestamp,
	}
	// New() installs these; the fixture builds the struct directly, and the full-round tests below
	// go through verify() which dispatches on them.
	i.verifyFn = i.verifyInteropMessages
	i.cycleVerifyFn = i.verifyCycleMessages
	i.initialized.Store(true)
	require.NoError(t, i.processBlockLogs(drivenDB, drivenInfo,
		types.Receipts{&types.Receipt{Logs: []*types.Log{logEntry}}}))

	// The log must really decode to one executing message, or a "wait" below would be the vacuous
	// kind that an empty block also produces.
	_, _, execMsgs, err := drivenDB.OpenBlock(waitTestDrivenBlock)
	require.NoError(t, err)
	require.Len(t, execMsgs, 1, "driven block must carry exactly one decoded executing message")

	blocks := blockPerChain{drivenID: {Number: waitTestDrivenBlock, Hash: drivenHash}}
	// Built directly rather than via l1HeadsFromMocks, which type-asserts on the bare mock and so
	// cannot see through A's receipts-serving wrapper.
	l1Heads := blockPerChain{drivenID: l1Block}
	return &laggingSourceFixture{
		t: t, interop: i, proven: chain,
		drivenID: drivenID, provenID: provenID,
		blocks: blocks, l1Heads: l1Heads, ts: initTimestamp,
	}
}

// round runs verification and the pure decision function, exactly as progressInterop does once
// readiness has already passed.
func (f *laggingSourceFixture) round() (StepOutput, Result) {
	f.t.Helper()
	result, err := f.interop.verifyInteropMessages(f.ts, f.blocks, f.l1Heads, nil)
	require.NoError(f.t, err)
	return decideVerifiedResult(RoundObservation{}, result), result
}

// prove is "the proof batch covering P block 1 lands on L1".
func (f *laggingSourceFixture) prove() {
	f.t.Helper()
	f.proven.AcceptBatch(f.t, waitTestInitBlock, eth.BlockID{Number: 41, Hash: common.HexToHash("0xc0")})
}

// TestWaitsForLaggingSourceThenAdvances is the live-edge scenario at the judge. A's block
// references a P log no proof has delivered; the round must WAIT, not invalidate. When the covering
// batch arrives the very same block verifies and the frontier advances.
func TestWaitsForLaggingSourceThenAdvances(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})

	out, result := f.round()
	require.Equal(t, DecisionWait, out.Decision,
		"a reference the source chain has not reached yet must wait, never invalidate")
	require.Empty(t, result.InvalidHeads, "nothing may be deny-listed over a late message")
	require.Contains(t, result.NotReady, f.drivenID)
	require.True(t, result.IsValid(), "not-yet-decidable is the absence of a verdict, not a bad one")
	require.False(t, result.IsReady())

	// A wait is not representable as a transition, so nothing reaches the WAL and the verified
	// frontier cannot move: this is the concrete sense in which the round is a no-op.
	_, err := f.interop.buildPendingTransition(out, RoundObservation{})
	require.ErrorContains(t, err, "unsupported transition decision")

	f.prove()

	out, result = f.round()
	require.Equal(t, DecisionAdvance, out.Decision,
		"once the source chain reaches the referenced block, the same block verifies")
	require.Empty(t, result.InvalidHeads)
	require.Empty(t, result.NotReady)
	require.Equal(t, f.blocks[f.drivenID], result.L2Heads[f.drivenID])
}

// TestConflictStillInvalidates keeps the other half of the rule: once the source chain HAS the
// referenced block, a reference that does not match it is a verdict, and the invalidate path is
// unchanged.
func TestConflictStillInvalidates(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.HexToHash("0xbadbad"))
	f.prove()

	out, result := f.round()
	require.Equal(t, DecisionInvalidate, out.Decision,
		"a checksum mismatch against proven history is a conflict, not a wait")
	require.Contains(t, result.InvalidHeads, f.drivenID)
	require.Empty(t, result.NotReady, "a decided block is not pending")
	require.False(t, result.IsValid())
	require.True(t, result.IsReady())
}

// TestHaltedSourceStallsWithoutInvalidating is the failure mode that matters operationally: the
// proof stream stops for good. Cross-safe progression for the referencing chain must stall at that
// timestamp — which is recoverable the moment proofs resume — and must never turn into an
// invalidation, which is not.
func TestHaltedSourceStallsWithoutInvalidating(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})

	for round := 0; round < 25; round++ {
		out, result := f.round()
		require.Equalf(t, DecisionWait, out.Decision, "round %d must still wait", round)
		require.Emptyf(t, result.InvalidHeads, "round %d must not deny-list anything", round)
		require.Containsf(t, result.NotReady, f.drivenID, "round %d must still be pending", round)
	}

	f.prove()
	out, _ := f.round()
	require.Equal(t, DecisionAdvance, out.Decision, "the stall resolves as soon as proofs resume")
}

// TestFrontierReadinessAlsoProvidesTheWait is the claim that the judge's wait has gone latent for a
// proof-carried chain in the common case, as a test rather than as an assertion in a comment.
//
// Because the chain is a MEMBER of the dependency set, the verifier's readiness check consults it:
// it asks for the round's timestamp, gets ethereum.NotFound because no proof covers it, and the
// round never runs. Both halves are asserted, because only the pair is the claim: not-ready BEFORE
// the proof, ready and decidable AFTER it, for the identical timestamp and the identical block.
func TestFrontierReadinessAlsoProvidesTheWait(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})

	_, err := f.interop.checkChainsReady(f.ts)
	require.ErrorIs(t, err, ethereum.NotFound,
		"with no proof covering the timestamp, readiness must report a retry-safe not-found")

	// A not-found from readiness is a WAIT, not a failed round: observeRound converts exactly this
	// error into ChainsReady=false, and checkPreconditions turns that into DecisionWait before
	// verification is even attempted.
	obs, err := f.interop.observeRound()
	require.NoError(t, err, "a chain that has not proven the timestamp yet must not fail the round")
	require.False(t, obs.ChainsReady)
	early := checkPreconditions(obs)
	require.NotNil(t, early)
	require.Equal(t, DecisionWait, early.Decision)

	f.prove()

	ready, err := f.interop.checkChainsReady(f.ts)
	require.NoError(t, err, "once the covering proof lands the chain is ready at that timestamp")
	require.Contains(t, ready.blocks, f.provenID)
	require.Equal(t, waitTestInitBlock, ready.blocks[f.provenID].Number)
	require.Equal(t, uint64(41), ready.l1Heads[f.provenID].Number,
		"a proven chain's L1 inclusion is the L1 block that carried its proof")

	// And the round that now runs is decidable rather than pending.
	out, result := f.round()
	require.Equal(t, DecisionAdvance, out.Decision)
	require.Empty(t, result.NotReady)

	require.Zero(t, f.proven.ReceiptsAsked(),
		"no interop path may reach a proof-carried chain's receipts")
}

// TestFrozenProvenChainFreezesTheClusterUntilItIsLabelled is hazard-3 at the cluster level, run
// through the real round rather than argued.
//
// The cross-safety round gates on EVERY chain answering for the round's timestamp. So a P that
// never advances a public label does not merely stall itself — it stalls chain A's cross-safe, and
// every other chain's, for as long as it is frozen. A is perfectly healthy throughout; nothing is
// wrong with it.
//
// CONTROL: P unlabelled. Twenty rounds, no progress, and the verified frontier never moves.
// TREATMENT: P labelled from its proven head. The very next round advances.
//
// This is the cluster half of the sequencer-side label source. The other half — that a silhouette
// Container in the sequencer posture supplies exactly this label from the proven head, where the
// chain's own container supplies none — is TestSequencerLabelSourceFollowsTheProvenHead in the
// silhouette package. Together they are the claim: without the label source A's cross-safe stalls,
// with it A's cross-safe advances.
func TestFrozenProvenChainFreezesTheClusterUntilItIsLabelled(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})

	_, initialised := f.interop.verifiedDB.LastTimestamp()
	require.False(t, initialised, "the frontier starts unset")

	// CONTROL. P answers nothing, because no proof has labelled it.
	for round := 0; round < 20; round++ {
		out, _, err := f.interop.progressInterop()
		require.NoError(t, err, "round %d must not fail: a chain that has not proven yet is not an error", round)
		require.Equalf(t, DecisionWait, out.Decision,
			"round %d: an unlabelled peer chain freezes the round", round)

		_, moved := f.interop.verifiedDB.LastTimestamp()
		require.Falsef(t, moved, "round %d: the cross-safe frontier must not have advanced", round)
	}

	// TREATMENT. The proof lands, which is what gives P a public label at this timestamp.
	f.prove()

	out, _, err := f.interop.progressInterop()
	require.NoError(t, err)
	require.Equal(t, DecisionAdvance, out.Decision,
		"a labelled peer chain unfreezes the cluster: A's cross-safe advances again")
	require.Equal(t, f.ts, out.Result.Timestamp)
	require.Contains(t, out.Result.L2Heads, f.drivenID, "and A is what advanced")
	require.Empty(t, out.Result.InvalidHeads, "with zero invalidations throughout")
}

// TestFrontierViewSkipsProvenChains is the first of the three capability-gated skips, under test.
//
// The frontier verification view exists because a DRIVEN chain's frontier block is not in its
// message database yet. A proof-carried chain's is — sealed when the proof landed — so it is left
// out of the view and every question about it goes to the database instead. The assertion that
// matters is the pair: no view entry, and no attempt to read receipts that do not exist.
func TestFrontierViewSkipsProvenChains(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})
	f.prove()

	view, err := f.interop.resolveFrontierVerificationView(blockPerChain{
		f.provenID: {Number: waitTestInitBlock, Hash: f.proven.blockHash(waitTestInitBlock)},
	})
	require.NoError(t, err, "a proof-carried chain must not fail frontier-view resolution")
	_, ok := view.block(f.provenID)
	require.False(t, ok, "a proof-carried chain contributes no frontier view")
	require.Zero(t, f.proven.ReceiptsAsked(), "and its receipts are never requested")

	// The database path yields exactly what the view would have: the real wire hash, no executing
	// messages, and containment answered from the sealed logs.
	ref, _, execMsgs, err := f.interop.logsDBs[f.provenID].OpenBlock(waitTestInitBlock)
	require.NoError(t, err)
	require.Equal(t, f.proven.blockHash(waitTestInitBlock), ref.Hash, "ref is the real wire hash")
	require.Empty(t, execMsgs, "a proof-carried chain's own executing messages are proof-trusted")

	logHash := messages.PayloadHashToLogHash(
		f.proven.PayloadHash(waitTestInitBlock, waitTestInitLogIdx), provenOrigin)
	seal, err := f.interop.logsDBs[f.provenID].Contains(messages.ChecksumArgs{
		BlockNumber: waitTestInitBlock,
		LogIndex:    waitTestInitLogIdx,
		Timestamp:   f.ts,
		ChainID:     f.provenID,
		LogHash:     logHash,
	}.Query())
	require.NoError(t, err, "containment is answered from the sealed logs")
	require.Equal(t, f.proven.blockHash(waitTestInitBlock), seal.Hash)
}

// TestFrontierLogPersistSkipsProvenChains is the second capability-gated skip: advancing the
// frontier must not try to re-seal a chain whose blocks its own sink already sealed.
func TestFrontierLogPersistSkipsProvenChains(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})
	f.prove()

	before, ok := f.interop.logsDBs[f.provenID].LatestSealedBlock()
	require.True(t, ok)
	require.NoError(t, f.interop.persistFrontierLogs(f.ts, blockPerChain{
		f.provenID: {Number: waitTestInitBlock, Hash: f.proven.blockHash(waitTestInitBlock)},
	}))
	after, ok := f.interop.logsDBs[f.provenID].LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, before, after, "the frontier persist must be a no-op for a proof-carried chain")
	require.Zero(t, f.proven.ReceiptsAsked())
}

// TestColdStartBackfillSkipsProvenChains is the third: backfill reads receipts block by block, and
// a proof-carried chain re-derives its history from L1 instead.
func TestColdStartBackfillSkipsProvenChains(t *testing.T) {
	t.Parallel()
	f := newLaggingSourceFixture(t, common.Hash{})
	f.prove()
	f.interop.logBackfillDepth = time.Hour

	// Only the proven chain is configured, so any receipt fetch at all would be its own.
	f.interop.chains = map[eth.ChainID]cc.InteropChain{f.provenID: f.proven}
	require.NoError(t, f.interop.runColdStartBackfill(f.ts+1))
	require.Zero(t, f.proven.ReceiptsAsked(),
		"cold-start backfill must not walk a proof-carried chain's blocks")
}

// TestIngestionSourceDefaultsToExecuted keeps the capability seam a pure addition: a container that
// does not declare a source is treated exactly as before, which is what lets every existing chain
// and every existing test go untouched.
func TestIngestionSourceDefaultsToExecuted(t *testing.T) {
	t.Parallel()
	plain := &algoMockChain{id: eth.ChainIDFromUInt64(10)}
	require.Equal(t, cc.IngestionExecuted, cc.IngestionSourceOf(plain))

	store, err := openLogsDB(testLogger(), eth.ChainIDFromUInt64(424247), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.Equal(t, cc.IngestionProven,
		cc.IngestionSourceOf(newFakeProvenChain(t, eth.ChainIDFromUInt64(424247), store)))
}
