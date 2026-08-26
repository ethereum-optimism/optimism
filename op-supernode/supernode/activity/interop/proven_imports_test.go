package interop

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	coreinterop "github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
)

// THE JUDGE FLIP (G7), in the direction wait_on_lagging_source_test.go does not cover.
//
// That file runs A-executes-P's-message: the proven chain is the SOURCE, and what is under test is
// that a verifier waits for a proof rather than invalidating the block that referenced it.
//
// This file runs the reverse and the harder half — P EXECUTES A'S MESSAGE. Before G7 the verifier had
// nothing to say about it: a proven chain's executing messages were empty on the wire and its
// cross-chain reads were trusted because a proof had been verified over them. Now the import list is
// on the wire, and P's dependencies go through the STOCK machinery: the same checksum lookup, the
// same activation and expiry invariants, the same wait-versus-conflict split.
//
// The shape of every test here is therefore: build a proof batch for a P block that declares an
// import, run the REAL round functions over it, and assert what the judge concluded — including, in
// the negative cases, that it concluded something at all. A judge that validated nothing would pass
// every "it cross-safed" assertion in this file, so each one is paired with a control.

const (
	// The forward-direction fixture's clock. A's initiating message sits at initTS, P's consuming
	// block one block later, because the wire refuses an import stamped at its own block's timestamp
	// (G7G D2).
	importsActivation = uint64(1000)
	importsBlockTime  = uint64(1)
	// importsInitBlock / importsInitLogIdx locate A's initiating message.
	importsInitBlock  = uint64(1)
	importsInitLogIdx = uint32(0)
	// importsProvenBlock is P's block that consumes it.
	importsProvenBlock = uint64(3)
)

// initiatingOrigin is the contract on A that emitted the initiating message.
var initiatingOrigin = common.Address{0xa1, 0xa2, 0xa3}

// initiatingChain is chain A: driven, healthy, and serving real receipts per block.
//
// It differs from wait_on_lagging_source_test.go's drivenChain in holding a MAP of blocks rather than
// one, because this fixture needs two: the block that emitted the initiating message, and the
// frontier block the round is deciding.
type initiatingChain struct {
	*algoMockChain
	infos map[uint64]*mockBlockInfo
	logs  map[uint64][]*types.Log
}

func (a *initiatingChain) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	info, ok := a.infos[blockID.Number]
	if !ok {
		return nil, nil, coreinterop.ErrFuture
	}
	return info, optypes.Receipts{{Receipt: types.Receipt{Logs: a.logs[blockID.Number]}}}, nil
}

// initiatingLog is the log A emitted: an ordinary contract event, not a CrossL2Inbox one. An
// initiating message is just a log; what makes it referenceable is its hash and its position.
func initiatingLog() *types.Log {
	return &types.Log{
		Address: initiatingOrigin,
		Topics:  []common.Hash{common.HexToHash("0xfeed")},
		Data:    []byte("hello from A"),
	}
}

// provenImportsFixture is chain A driven and chain P proof-carried, with P importing A's messages.
type provenImportsFixture struct {
	t        *testing.T
	interop  *Interop
	proven   *fakeProvenChain
	driven   *initiatingChain
	drivenDB LogsDB
	drivenID eth.ChainID
	provenID eth.ChainID
	// initTS is the timestamp of A's initiating block; provenTS is P's consuming block's.
	initTS   uint64
	provenTS uint64
}

type provenImportsOptions struct {
	// expiryWindow overrides the message expiry window. Zero means the stock default.
	expiryWindow uint64
	// sealDrivenBlocks are A's blocks to seal into A's LogsDB up front. Empty means block 1 only.
	sealDrivenBlocks []uint64
}

// newProvenImportsFixture builds the two-chain system. A's blocks 1 and 2 exist and each carries one
// initiating log; which of them are SEALED into A's message database is the test's choice, because
// that is the difference between "the dependency is there" and "the dependency is not there yet".
func newProvenImportsFixture(t *testing.T, opts provenImportsOptions) *provenImportsFixture {
	t.Helper()
	provenID := eth.ChainIDFromUInt64(424247)
	drivenID := eth.ChainIDFromUInt64(424246)

	provenDB, err := openLogsDB(testLogger(), provenID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = provenDB.Close() })
	proven := newFakeProvenChain(t, provenID, provenDB)

	drivenDB, err := openLogsDB(testLogger(), drivenID, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = drivenDB.Close() })

	initTS := importsActivation + importsInitBlock*importsBlockTime
	provenTS := importsActivation + importsProvenBlock*importsBlockTime

	// A has three blocks: 1 and 2 each emit one initiating log, and 3 is the frontier block the
	// round decides alongside P's.
	infos := map[uint64]*mockBlockInfo{}
	logs := map[uint64][]*types.Log{}
	parent := common.HexToHash("0xa000")
	for num := uint64(1); num <= 3; num++ {
		hash := common.BytesToHash([]byte{byte(num), 0xa0, 0x0d})
		infos[num] = &mockBlockInfo{
			hash:       hash,
			parentHash: parent,
			number:     num,
			timestamp:  importsActivation + num*importsBlockTime,
		}
		parent = hash
		if num < 3 {
			logs[num] = []*types.Log{initiatingLog()}
		}
	}

	drivenMock := newMockChainWithL1(drivenID,
		eth.BlockID{Number: 40, Hash: common.HexToHash("0x1")},
		eth.BlockID{Number: 3, Hash: infos[3].hash})
	drivenMock.blockTimeOverride = importsBlockTime
	drivenMock.optimisticL2 = eth.BlockID{Number: 3, Hash: infos[3].hash}
	driven := &initiatingChain{algoMockChain: drivenMock, infos: infos, logs: logs}

	verifiedDB, err := OpenVerifiedDB(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = verifiedDB.Close() })

	expiry := opts.expiryWindow
	if expiry == 0 {
		expiry = defaultMessageExpiryWindow
	}

	i := &Interop{
		log:                 testLogger(),
		ctx:                 context.Background(),
		activationTimestamp: importsActivation,
		messageExpiryWindow: expiry,
		logsDBs:             map[eth.ChainID]LogsDB{drivenID: drivenDB, provenID: provenDB},
		chains:              map[eth.ChainID]cc.InteropChain{drivenID: driven, provenID: proven},
		verifiedDB:          verifiedDB,
		l1Checker:           noopL1Checker{},
		metrics:             resources.NewSupernodeMetrics(),

		verificationStartTimestamp: initTS,
	}
	i.verifyFn = i.verifyInteropMessages
	i.cycleVerifyFn = i.verifyCycleMessages
	i.initialized.Store(true)

	f := &provenImportsFixture{
		t: t, interop: i, proven: proven, driven: driven, drivenDB: drivenDB,
		drivenID: drivenID, provenID: provenID, initTS: initTS, provenTS: provenTS,
	}
	toSeal := opts.sealDrivenBlocks
	if toSeal == nil {
		toSeal = []uint64{importsInitBlock}
	}
	for _, num := range toSeal {
		f.sealDrivenBlock(num)
	}
	return f
}

// sealDrivenBlock puts one of A's blocks into A's message database through the production
// processBlockLogs path — the same path a driven chain's logs really take.
func (f *provenImportsFixture) sealDrivenBlock(num uint64) {
	f.t.Helper()
	info := f.driven.infos[num]
	require.NotNil(f.t, info, "A has no block %d", num)
	require.NoError(f.t, f.interop.processBlockLogs(f.drivenDB, info,
		types.Receipts{&types.Receipt{Logs: f.driven.logs[num]}}))
}

// importOf is the ExecMsg P's wire would carry for A's initiating log at block `num`, index 0.
//
// It is derived from the LOG rather than written out, so a checksum mismatch in this test can only
// come from the code under test and never from a stale constant.
func (f *provenImportsFixture) importOf(num uint64) proofbatch.ExecMsg {
	f.t.Helper()
	log := initiatingLog()
	return proofbatch.ExecMsg{Message: messages.Message{
		Identifier: messages.Identifier{
			Origin:      log.Address,
			BlockNumber: num,
			LogIndex:    importsInitLogIdx,
			Timestamp:   importsActivation + num*importsBlockTime,
			ChainID:     f.drivenID,
		},
		PayloadHash: crypto.Keccak256Hash(messages.LogToMessagePayload(log)),
	}}
}

// acceptProvenBlock is "the proof batch covering P's consuming block lands on L1".
func (f *provenImportsFixture) acceptProvenBlock(imports ...proofbatch.ExecMsg) {
	f.t.Helper()
	carrier := eth.BlockID{Number: 40, Hash: common.HexToHash("0x1")}
	for num := uint64(1); num < importsProvenBlock; num++ {
		f.proven.AcceptBatch(f.t, num, carrier)
	}
	f.proven.AcceptBatchWithImports(f.t, importsProvenBlock, carrier, imports...)
}

// round runs the real verification over the frontier timestamp: A's block 3 and P's block 3, both at
// provenTS, exactly as progressInterop does once readiness has passed.
func (f *provenImportsFixture) round() (StepOutput, Result) {
	f.t.Helper()
	blocks := blockPerChain{
		f.drivenID: {Number: 3, Hash: f.driven.infos[3].hash},
		f.provenID: {Number: importsProvenBlock, Hash: f.proven.blockHash(importsProvenBlock)},
	}
	l1Heads := blockPerChain{
		f.drivenID: {Number: 40, Hash: common.HexToHash("0x1")},
		f.provenID: {Number: 40, Hash: common.HexToHash("0x1")},
	}
	// Through verify(), not verifyInteropMessages directly, so the frontier view is really resolved
	// and the cycle check really runs — the two places a proven chain's imports are handled.
	result, err := f.interop.verify(f.provenTS, blocks, l1Heads)
	require.NoError(f.t, err)
	return decideVerifiedResult(RoundObservation{}, result), result
}

// TestProvenChainDependencyIsVerifiedAndCrossSafes is the happy path, and the one that closes
// ruling-7: a P block that consumed a REAL message on A is validated by the stock judge and the round
// advances.
//
// The control is what makes it worth anything. `importsOnWire = false` reproduces the retired
// pre-v3 posture on the identical fixture with the identical (wrong) import list, and it ADVANCES —
// because a verifier that is told nothing about imports checks nothing. So the assertion below is
// "the check ran and passed", not merely "the round advanced".
func TestProvenChainDependencyIsVerifiedAndCrossSafes(t *testing.T) {
	f := newProvenImportsFixture(t, provenImportsOptions{})
	f.acceptProvenBlock(f.importOf(importsInitBlock))

	// The import really is one message, and it really does resolve against A's database. Asserted
	// directly first, so a later "advance" cannot be the vacuous kind an empty list produces.
	msgs, onWire, err := cc.ProvenExecMsgsOf(f.interop.chains[f.provenID], importsProvenBlock)
	require.NoError(t, err)
	require.True(t, onWire, "wire v3 must declare the import list")
	require.Len(t, msgs, 1, "P's block must declare exactly one import")
	require.Equal(t, f.drivenID, msgs[0].ChainID)
	require.Equal(t, importsInitBlock, msgs[0].BlockNum)

	out, result := f.round()
	require.Equal(t, DecisionAdvance, out.Decision)
	require.Empty(t, result.InvalidHeads)
	require.Empty(t, result.NotReady)
	require.Zero(t, f.proven.Invalidations())
	// P's receipts were never touched: the dependency was checked from the wire, not from execution
	// data this node does not have.
	require.Zero(t, f.proven.ReceiptsAsked())
}

// TestProvenChainDependencyCheckIsNotVacuous is the negative control for the test above, and it is
// the assertion that the FLIP happened rather than that the round is healthy.
//
// Same fixture, same P block, but the import names a message A never emitted — a real log position
// with a wrong message hash, which is the exact shape of a prover that lied about what it consumed.
// A verifier still running the pre-v3 posture would advance; this one must not.
func TestProvenChainDependencyCheckIsNotVacuous(t *testing.T) {
	f := newProvenImportsFixture(t, provenImportsOptions{})
	bogus := f.importOf(importsInitBlock)
	bogus.PayloadHash = common.HexToHash("0xdeadbeef") // A's log hashes to something else
	f.acceptProvenBlock(bogus)

	out, result := f.round()
	require.Equal(t, DecisionInvalidate, out.Decision,
		"a P block referencing a message A never emitted must not reach cross-safe")
	require.Contains(t, result.InvalidHeads, f.provenID)
	require.Equal(t, importsProvenBlock, result.InvalidHeads[f.provenID].BlockID.Number)

	// ...and the same fixture under the RETIRED posture advances, which is what proves the assertion
	// above is about the flip and not about something else in the round.
	f.proven.importsOnWire = false
	outLegacy, resultLegacy := f.round()
	require.Equal(t, DecisionAdvance, outLegacy.Decision,
		"the pre-v3 posture is supposed to be the weaker one; if it also refuses, this test proves nothing")
	require.Empty(t, resultLegacy.InvalidHeads)
}

// TestProvenChainDependencyWaitsThenAdvances is the wait-fix discipline applied to the new direction:
// NOT-YET-INDEXED IS NOT INVALID.
//
// P's block imports A's block 2, and A's message database is sealed only through block 1. That is not
// a conflict — it is the absence of a verdict. So the round must WAIT, invalidate nothing, and then
// advance on the SAME block once A catches up.
func TestProvenChainDependencyWaitsThenAdvances(t *testing.T) {
	f := newProvenImportsFixture(t, provenImportsOptions{sealDrivenBlocks: []uint64{importsInitBlock}})
	f.acceptProvenBlock(f.importOf(2))

	// The reference really is beyond A's head, or the wait below would be an accident.
	head, ok := f.drivenDB.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, importsInitBlock, head.Number, "A must be sealed only through the earlier block")

	out, result := f.round()
	require.Equal(t, DecisionWait, out.Decision, "a dependency A has not indexed yet must wait")
	require.Empty(t, result.InvalidHeads, "a late dependency is not a bad one")
	require.Contains(t, result.NotReady, f.provenID)
	require.Zero(t, f.proven.Invalidations())

	// The wait must also TERMINATE. A's block 2 is sealed and the same round, on the same P block,
	// now advances — which is the half of the wait-fix that a test asserting only the pin would miss.
	f.sealDrivenBlock(2)
	out, result = f.round()
	require.Equal(t, DecisionAdvance, out.Decision)
	require.Empty(t, result.InvalidHeads)
	require.Empty(t, result.NotReady)
	require.Zero(t, f.proven.Invalidations())
}

// TestProvenChainDependencyThatCanNeverExist covers the third class: a reference that STOCK semantics
// can decide against immediately and permanently. Each case is a different rule, and the point is
// that the rule is the stock one — none of this code is silhouette-specific.
//
// The DOCUMENTED OUTCOME, uniform across all of them: the judge returns InvalidHead for P and the
// round decides DecisionInvalidate. The sequencing container applies the stock deposits-only
// replacement; a verifier waits for the corrected proof rather than synthesising a different block.
func TestProvenChainDependencyThatCanNeverExist(t *testing.T) {
	t.Run("checksum conflict against a sealed block", func(t *testing.T) {
		f := newProvenImportsFixture(t, provenImportsOptions{})
		bad := f.importOf(importsInitBlock)
		bad.PayloadHash = common.HexToHash("0x1234") // right position, wrong content
		f.acceptProvenBlock(bad)

		out, result := f.round()
		require.Equal(t, DecisionInvalidate, out.Decision)
		require.Contains(t, result.InvalidHeads, f.provenID)
	})

	t.Run("log index A's block does not have", func(t *testing.T) {
		f := newProvenImportsFixture(t, provenImportsOptions{})
		bad := f.importOf(importsInitBlock)
		bad.Identifier.LogIndex = 99 // A's block 1 emitted one log
		f.acceptProvenBlock(bad)

		out, result := f.round()
		require.Equal(t, DecisionInvalidate, out.Decision)
		require.Contains(t, result.InvalidHeads, f.provenID)
	})

	t.Run("a chain that is not in the dependency set", func(t *testing.T) {
		f := newProvenImportsFixture(t, provenImportsOptions{})
		bad := f.importOf(importsInitBlock)
		bad.Identifier.ChainID = eth.ChainIDFromUInt64(999999)
		f.acceptProvenBlock(bad)

		out, result := f.round()
		require.Equal(t, DecisionInvalidate, out.Decision)
		require.Contains(t, result.InvalidHeads, f.provenID)
	})

	t.Run("an expired message", func(t *testing.T) {
		// The expiry window is squeezed rather than the clock stretched, because the fixture's block
		// numbering and its timestamps are the same arithmetic and a far-future P block would need a
		// far-future A too. The rule under test is the stock one either way.
		f := newProvenImportsFixture(t, provenImportsOptions{expiryWindow: 1})
		f.acceptProvenBlock(f.importOf(importsInitBlock))

		out, result := f.round()
		require.Equal(t, DecisionInvalidate, out.Decision,
			"a message older than the expiry window must not be consumable, on a proven chain either")
		require.Contains(t, result.InvalidHeads, f.provenID)

		// The control: the same message inside the window is fine, so the refusal is the expiry rule
		// and not the fixture.
		wide := newProvenImportsFixture(t, provenImportsOptions{})
		wide.acceptProvenBlock(wide.importOf(importsInitBlock))
		outWide, _ := wide.round()
		require.Equal(t, DecisionAdvance, outWide.Decision)
	})
}

func TestProvenChainInvalidationReachesContainer(t *testing.T) {
	f := newProvenImportsFixture(t, provenImportsOptions{})
	bad := f.importOf(importsInitBlock)
	bad.PayloadHash = common.HexToHash("0xbad")
	f.acceptProvenBlock(bad)

	out, result := f.round()
	require.Equal(t, DecisionInvalidate, out.Decision)
	invalid := result.InvalidHeads[f.provenID]
	require.Equal(t, importsProvenBlock, invalid.BlockID.Number)

	// The generic interop layer does not special-case proof-carried chains. Their container owns the
	// replacement policy, just as an ordinary container owns its engine rewind.
	rewound, err := f.interop.invalidateBlock(f.provenID, invalid.BlockID, f.provenTS,
		invalid.StateRoot, invalid.MessagePasserStorageRoot, nil)
	require.NoError(t, err)
	require.False(t, rewound)
	require.Equal(t, 1, f.proven.Invalidations())

	// A driven chain still follows the same generic path.
	rewound, err = f.interop.invalidateBlock(f.drivenID, eth.BlockID{Number: 3, Hash: f.driven.infos[3].hash},
		f.provenTS, eth.Bytes32{}, eth.Bytes32{}, nil)
	require.NoError(t, err)
	require.False(t, rewound)
}

// TestProvenChainSameTimestampImportIsUnreachable is the assertion behind the ordinal key.
//
// A proven chain's imports carry no position, and the same-timestamp dependency graph orders
// executing messages BY position. The resolution (G7G D2) is that the codec refuses a batch whose
// import is stamped at its own block's timestamp, so the class never reaches acceptance — and the
// cycle path asserts that rather than assuming it.
//
// Both halves are checked here, because either one alone is worth little: the codec's refusal is
// what makes the situation impossible, and the cycle path's error is what makes a REGRESSION in that
// refusal loud instead of a silent soundness hole.
func TestProvenChainSameTimestampImportIsUnreachable(t *testing.T) {
	f := newProvenImportsFixture(t, provenImportsOptions{})

	sameTS := f.importOf(importsInitBlock)
	sameTS.Identifier.Timestamp = f.provenTS

	t.Run("acceptance refuses it", func(t *testing.T) {
		blk := proofbatch.BlockExport{
			Number:    importsProvenBlock,
			Timestamp: f.provenTS,
			Hash:      common.HexToHash("0x9e03"),
			ExecMsgs:  []proofbatch.ExecMsg{sameTS},
		}
		batch := &proofbatch.ProofBatch{Blocks: []proofbatch.BlockExport{blk}}
		// Well-formed as a wire object — the format permits it — and refused by the verifier, which is
		// the layer that owns the rule.
		require.NoError(t, batch.CheckStructure())
		require.ErrorIs(t, batch.CheckNoSameTimestampImports(), proofbatch.ErrSameTimestampImport)
	})

	t.Run("and the cycle path refuses it too, if it ever got there", func(t *testing.T) {
		// Injected past the codec, which is the only way to reach this: the fake's import table is
		// written directly, simulating a codec regression rather than a batch a prover could post.
		f.acceptProvenBlock()
		f.proven.imports[importsProvenBlock] = []messages.ExecutingMessage{*sameTS.Executing()}

		blocks := blockPerChain{
			f.provenID: {Number: importsProvenBlock, Hash: f.proven.blockHash(importsProvenBlock)},
		}
		_, err := f.interop.verifyCycleMessages(f.provenTS, blocks, nil)
		require.Error(t, err, "a same-timestamp import on a proven chain must stall the round loudly")
		require.Contains(t, err.Error(), "carry no position")
	})
}

// TestProvenChainImportsFailClosed is the fail-open guard, and it is the most important test in this
// file because the failure it prevents is invisible.
//
// The fact table is a WINDOW. A judge asking about a block whose facts have been pruned, or about a
// container that does not implement the capability at all, must get an ERROR — never an empty import
// list, which reads identically to "this block consumed nothing" and would leave a verifier
// validating no dependencies while every log line and every metric looked healthy.
func TestProvenChainImportsFailClosed(t *testing.T) {
	t.Run("a block with no facts", func(t *testing.T) {
		f := newProvenImportsFixture(t, provenImportsOptions{})
		// No batch accepted at all: the round asks about a block this node cannot describe.
		blocks := blockPerChain{
			f.provenID: {Number: importsProvenBlock, Hash: common.HexToHash("0x9e03")},
		}
		l1Heads := blockPerChain{f.provenID: {Number: 40, Hash: common.HexToHash("0x1")}}
		_, err := f.interop.verifyInteropMessages(f.provenTS, blocks, l1Heads, nil)
		require.Error(t, err, "absent facts must not be read as an absent import list")
	})

	t.Run("a container that does not implement the capability", func(t *testing.T) {
		// A bare mock that claims proven ingestion and nothing else — the wiring mistake this guards
		// against. It must be an error rather than a chain whose dependencies nobody checks.
		_, _, err := cc.ProvenExecMsgsOf(&provenWithoutImports{algoMockChain: &algoMockChain{
			id: eth.ChainIDFromUInt64(1),
		}}, 1)
		require.ErrorContains(t, err, "cannot report the executing messages")
	})
}

// provenWithoutImports declares proven ingestion and does not implement ProvenMessageImports.
type provenWithoutImports struct {
	*algoMockChain
}

func (p *provenWithoutImports) IngestionSource() cc.IngestionSource { return cc.IngestionProven }
