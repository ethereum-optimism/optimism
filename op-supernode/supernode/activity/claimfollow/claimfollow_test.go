package claimfollow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

var registryAddr = common.HexToAddress("0x4200000000000000000000000000000000000777")

// The fake rendering chain uses an epoch of 6 L2 blocks per L1 block, so that origins and sequence
// numbers both vary across the heights the tests read. Those two fields are the ones a hand-built
// ref cannot fake and the consumer compares, which is the whole reason completion reads a real
// payload instead of assembling a struct.
const epochBlocks = 6

func l1Hash(num uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("l1-%d", num)))
}

// renderHash names a rendering block: same number, different fork tag is a reorg.
func renderHash(num uint64, fork string) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("rendering-%s-%d", fork, num)))
}

func renderTime(num uint64) uint64 { return 1_000_000 + num*2 }

// privHash and privParent are what the CLAIM publishes: the private chain's identity at a height,
// which no public block carries.
func privHash(num uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("private-%d", num)))
}

// wantRef is the ref the module must serve for a claim ending at num: two fields from the claim,
// four from the rendering block at the same height.
func wantRef(num uint64) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           privHash(num),
		ParentHash:     privHash(num - 1),
		Number:         num,
		Time:           renderTime(num),
		L1Origin:       eth.BlockID{Hash: l1Hash(num / epochBlocks), Number: num / epochBlocks},
		SequenceNumber: num % epochBlocks,
	}
}

func testRollupCfg() *rollup.Config {
	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: l1Hash(0), Number: 0},
			L2:     eth.BlockID{Hash: renderHash(0, "a"), Number: 0},
			L2Time: renderTime(0),
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.HexToAddress("0x1111"),
				Overhead:    eth.Bytes32{},
				Scalar:      eth.Bytes32{},
				GasLimit:    30_000_000,
			},
		},
		BlockTime:              2,
		SeqWindowSize:          3600,
		MaxSequencerDrift:      600,
		L1ChainID:              big.NewInt(900),
		L2ChainID:              big.NewInt(424243),
		BatchInboxAddress:      common.HexToAddress("0x4242"),
		DepositContractAddress: common.HexToAddress("0x4243"),
		L1SystemConfigAddress:  common.HexToAddress("0x4244"),
	}
}

// ---------------------------------------------------------------------------
// The fake rendering chain. It is the ONLY thing the module reads.
// ---------------------------------------------------------------------------

type fakeBlock struct {
	env      *eth.ExecutionPayloadEnvelope
	userTxs  types.Transactions
	reverted map[common.Hash]bool
}

type fakeRendering struct {
	t *testing.T

	safe      uint64
	finalized uint64
	currentL1 eth.L1BlockRef
	statusErr error

	blocks     map[uint64]*fakeBlock
	payloadErr map[uint64]error
}

var _ Rendering = (*fakeRendering)(nil)

func newFakeRendering(t *testing.T) *fakeRendering {
	f := &fakeRendering{t: t, blocks: map[uint64]*fakeBlock{}, payloadErr: map[uint64]error{}}
	f.set(0, "a", 0)
	return f
}

// set writes a rendering block at num on the named fork. Its parent is the same fork's previous
// block, except at or below forkAt, where it links to the original chain. The block always leads
// with a REAL L1-attributes deposit, because that is where the origin and sequence number the
// module copies actually live.
func (f *fakeRendering) set(num uint64, fork string, forkAt uint64, txs ...*types.Transaction) {
	f.t.Helper()
	parentFork := fork
	if num <= forkAt {
		parentFork = "a"
	}
	cfg := testRollupCfg()
	l1Info := &testutils.MockBlockInfo{
		InfoParentHash:  l1Hash(num/epochBlocks - 1),
		InfoNum:         num / epochBlocks,
		InfoTime:        renderTime(num) - (num%epochBlocks)*2,
		InfoHash:        l1Hash(num / epochBlocks),
		InfoBaseFee:     big.NewInt(1),
		InfoBlobBaseFee: big.NewInt(1),
		InfoReceiptRoot: types.EmptyRootHash,
		InfoRoot:        common.Hash{0x99},
		InfoGasUsed:     21000,
	}
	l1Bytes, err := derive.L1InfoDepositBytes(cfg, params.MergedTestChainConfig, cfg.Genesis.SystemConfig,
		num%epochBlocks, l1Info, renderTime(num))
	require.NoError(f.t, err)

	data := []eth.Data{l1Bytes}
	for _, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(f.t, err)
		data = append(data, raw)
	}
	f.blocks[num] = &fakeBlock{
		env: &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:   renderHash(num-1, parentFork),
			BlockNumber:  eth.Uint64Quantity(num),
			Timestamp:    eth.Uint64Quantity(renderTime(num)),
			BlockHash:    renderHash(num, fork),
			Transactions: data,
		}},
		userTxs:  txs,
		reverted: map[common.Hash]bool{},
	}
}

// fill writes plain blocks over an inclusive height range.
func (f *fakeRendering) fill(from, to uint64, fork string, forkAt uint64) {
	for n := from; n <= to; n++ {
		f.set(n, fork, forkAt)
	}
}

func (f *fakeRendering) revert(num uint64, tx *types.Transaction) {
	f.blocks[num].reverted[tx.Hash()] = true
}

func (f *fakeRendering) SyncStatus(context.Context) (*eth.SyncStatus, error) {
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return &eth.SyncStatus{
		SafeL2:      eth.L2BlockRef{Number: f.safe},
		FinalizedL2: eth.L2BlockRef{Number: f.finalized},
		CurrentL1:   f.currentL1,
	}, nil
}

func (f *fakeRendering) PayloadByNumber(_ context.Context, num uint64) (*eth.ExecutionPayloadEnvelope, error) {
	if err := f.payloadErr[num]; err != nil {
		return nil, err
	}
	b, ok := f.blocks[num]
	if !ok {
		return nil, fmt.Errorf("no rendering block at %d", num)
	}
	return b.env, nil
}

func (f *fakeRendering) FetchReceipts(_ context.Context, id eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	b, ok := f.blocks[id.Number]
	if !ok || b.env.ExecutionPayload.BlockHash != id.Hash {
		return nil, nil, fmt.Errorf("no rendering block with hash %s at %d", id.Hash, id.Number)
	}
	var out optypes.Receipts
	for _, tx := range b.userTxs {
		status := types.ReceiptStatusSuccessful
		if b.reverted[tx.Hash()] {
			status = types.ReceiptStatusFailed
		}
		r := &optypes.Receipt{}
		r.TxHash = tx.Hash()
		r.Status = status
		out = append(out, r)
	}
	return nil, out, nil
}

// ---------------------------------------------------------------------------
// Metrics recorder
// ---------------------------------------------------------------------------

type countingMetrics struct {
	claims    int
	rejected  map[string]int
	reorgs    int
	safe      uint64
	finalized uint64
}

func newCountingMetrics() *countingMetrics { return &countingMetrics{rejected: map[string]int{}} }

func (m *countingMetrics) RecordClaim()                      { m.claims++ }
func (m *countingMetrics) RecordRejectedClaim(reason string) { m.rejected[reason]++ }
func (m *countingMetrics) RecordRenderingReorg()             { m.reorgs++ }
func (m *countingMetrics) RecordSafe(h uint64)               { m.safe = h }
func (m *countingMetrics) RecordFinalized(h uint64)          { m.finalized = h }

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type harness struct {
	t *testing.T
	r *fakeRendering
	m *countingMetrics
	f *Module

	// highestSafe and highestFinal track what has EVER been served, so that every read asserts the
	// monotonicity a sequencing follower depends on.
	highestSafe  uint64
	highestFinal uint64
}

// privateGenesisHash is derived from the operator's private-chain genesis: the one field of
// the private genesis ref that no public data carries.
func privateGenesisHash() common.Hash { return privHash(0) }

// wantGenesisRef is the not-yet state's ref: one configured hash and five fields the module derives
// from the rendering's rollup config and the definition of a genesis block.
func wantGenesisRef() eth.L2BlockRef {
	cfg := testRollupCfg()
	return eth.L2BlockRef{
		Hash:           privateGenesisHash(),
		ParentHash:     common.Hash{},
		Number:         cfg.Genesis.L2.Number,
		Time:           cfg.Genesis.L2Time,
		L1Origin:       cfg.Genesis.L1,
		SequenceNumber: 0,
	}
}

func newHarness(t *testing.T) *harness {
	return newHarnessWithConfig(t, Config{Registry: registryAddr, GenesisHash: privateGenesisHash()})
}

func newHarnessWithConfig(t *testing.T, cfg Config) *harness {
	r := newFakeRendering(t)
	m := newCountingMetrics()
	f := New(cfg, testRollupCfg(), testlog.Logger(t, gethlog.LevelInfo), m)
	f.Attach(r)
	return &harness{t: t, r: r, m: m, f: f}
}

func (h *harness) step() error { return h.f.Step(context.Background()) }

// status reads the served state and asserts, on EVERY read, the invariants the follow consumer
// checks and the monotonicity a sequencing follower depends on.
func (h *harness) status() *eth.SyncStatus {
	h.t.Helper()
	st, err := h.f.SyncStatus()
	require.NoError(h.t, err)
	require.Equal(h.t, st.LocalSafeL2, st.SafeL2, "this design serves safe == local_safe")
	require.LessOrEqual(h.t, st.FinalizedL2.Number, st.SafeL2.Number, "finalized must not exceed safe")
	require.LessOrEqual(h.t, st.SafeL2.Number, st.LocalSafeL2.Number, "safe must not exceed local safe")
	require.GreaterOrEqual(h.t, st.SafeL2.Number, h.highestSafe, "served safe must never regress")
	require.GreaterOrEqual(h.t, st.FinalizedL2.Number, h.highestFinal, "served finalized must never regress")
	h.highestSafe, h.highestFinal = st.SafeL2.Number, st.FinalizedL2.Number

	// Everything the follow consumer does not read stays at its zero value, so no consumer can come
	// to depend on a field this module has no honest answer for.
	require.Equal(h.t, eth.L1BlockRef{}, st.HeadL1)
	require.Equal(h.t, eth.L1BlockRef{}, st.SafeL1)
	require.Equal(h.t, eth.L1BlockRef{}, st.FinalizedL1)
	require.Equal(h.t, eth.L2BlockRef{}, st.UnsafeL2)
	require.Equal(h.t, eth.L2BlockRef{}, st.PendingSafeL2)
	return st
}

// claimTx builds a postClaim transaction, the way the batcher's terminal seam does. The two hashes
// it publishes are the private chain's identity at lastBlock.
func claimTx(t *testing.T, nonce, first, last uint64) *types.Transaction {
	t.Helper()
	return claimTxWithTerminal(t, nonce, first, last, privHash(last), privHash(last-1))
}

func claimTxWithTerminal(t *testing.T, nonce, first, last uint64, terminal, parent common.Hash) *types.Transaction {
	t.Helper()
	data, err := render.EncodePostClaim(&codec.RangeClaim{
		FirstBlock:                first,
		LastBlock:                 last,
		PrivateTerminalBlockHash:  terminal,
		PrivateTerminalParentHash: parent,
		L1Head:                    crypto.Keccak256Hash([]byte("l1head")),
		RollupConfigHash:          crypto.Keccak256Hash([]byte("rollupcfg")),
		DepSetHash:                crypto.Keccak256Hash([]byte("depset")),
		PrivateDataHash:           crypto.Keccak256Hash([]byte(fmt.Sprintf("data-%d-%d", first, last))),
	})
	require.NoError(t, err)
	return rawTx(nonce, registryAddr, data)
}

func rawTx(nonce uint64, to common.Address, data []byte) *types.Transaction {
	return types.NewTx(&types.LegacyTx{Nonce: nonce, To: &to, Gas: 1_000_000, GasPrice: big.NewInt(1), Data: data})
}

// ---------------------------------------------------------------------------
// Gates
// ---------------------------------------------------------------------------

// THE NOT-YET STATE. Before any claim the module serves the private chain's GENESIS ref for all
// three L2 labels — a complete, true status from the module's very first tick.
//
// Serving one rather than erroring is what keeps the pair able to bootstrap at all: everything
// downstream of the op-node this feeds inherits its silence, and the operator's own batcher will not
// load a block until it is told a non-zero current_l1. See the package comment.
func TestGenesisRefBeforeAnyClaim(t *testing.T) {
	h := newHarness(t)

	// True before a single poll: the not-yet state is seeded at construction, not discovered.
	st := h.status()
	require.Equal(t, wantGenesisRef(), st.LocalSafeL2)
	require.Equal(t, wantGenesisRef(), st.SafeL2)
	require.Equal(t, wantGenesisRef(), st.FinalizedL2)

	h.r.fill(1, 4, "a", 0)
	h.r.safe, h.r.finalized = 4, 4
	require.NoError(t, h.step())

	st = h.status()
	require.Equal(t, wantGenesisRef(), st.SafeL2, "no claim, no movement")
	require.Zero(t, h.m.claims)
}

// The not-yet status must pass the checks the consumer actually makes, or it takes the whole status
// down instead of standing in for it.
//
// The driver hash-checks the L1 ORIGIN of all three served refs against real L1
// (op-node/rollup/driver/driver.go), which is why a zero-valued or synthesised ref would fail: L1
// block 0's real hash is not the zero hash. The genesis ref passes because its origin IS the chain's
// real genesis anchor, taken from the rollup config rather than invented.
func TestTheNotYetStatusPassesTheConsumersChecks(t *testing.T) {
	h := newHarness(t)
	st := h.status() // status() itself asserts finalized <= safe <= local_safe on every read.

	cfg := testRollupCfg()
	for name, ref := range map[string]eth.L2BlockRef{
		"local_safe": st.LocalSafeL2, "safe": st.SafeL2, "finalized": st.FinalizedL2,
	} {
		require.Equal(t, cfg.Genesis.L1, ref.L1Origin,
			"%s must carry the chain's REAL genesis L1 origin; the driver hash-checks it against L1", name)
		require.NotEqual(t, eth.BlockID{}, ref.L1Origin, "%s must not carry a zero origin", name)
		require.NotEqual(t, common.Hash{}, ref.Hash, "%s must name a real block", name)
	}
	// A genesis block's parent hash and sequence number are zero by definition, which is exactly what
	// derive.PayloadToBlockRef produces on its own genesis branch — so the served ref compares equal
	// to the one the private EL derives for itself.
	require.Equal(t, common.Hash{}, st.SafeL2.ParentHash)
	require.Zero(t, st.SafeL2.SequenceNumber)
	require.Equal(t, cfg.Genesis.L2Time, st.SafeL2.Time)
}

// The transition out of the not-yet state is a step forward and never a step sideways: genesis, then
// the first claim, with the ordering invariants and monotonicity holding across the boundary.
func TestTheGenesisToFirstClaimTransitionIsMonotone(t *testing.T) {
	h := newHarness(t)
	require.Equal(t, wantGenesisRef(), h.status().SafeL2)

	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)

	// Walk the rendering forward one height at a time. h.status() asserts on EVERY read that neither
	// label ever regresses, so the whole walk is the monotonicity gate.
	for n := uint64(1); n <= 8; n++ {
		h.r.safe, h.r.finalized = n, n
		require.NoError(t, h.step())
		st := h.status()
		if n < 8 {
			require.Equal(t, wantGenesisRef(), st.SafeL2, "still genesis at rendering height %d", n)
		}
	}

	st := h.status()
	require.Equal(t, wantRef(8), st.SafeL2, "the first claim replaces genesis in one step")
	require.Equal(t, wantRef(8), st.FinalizedL2)
	require.Greater(t, st.SafeL2.Number, wantGenesisRef().Number)
}

// The headline behaviour: a ref built from two claim fields and four public ones, equal field for
// field to what the private EL would have reported.
func TestRefIsCompletedFromPublicData(t *testing.T) {
	h := newHarness(t)
	// The claim LEADS its range: it rides in block 1 and describes blocks 1..8, so its ref lives at
	// height 8, seven blocks after the transaction that announced it.
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())

	st := h.status()
	require.Equal(t, wantRef(8), st.LocalSafeL2)
	require.Equal(t, wantRef(8), st.SafeL2)
	require.Equal(t, uint64(8), st.SafeL2.Number)
	// The two claim-borne fields are the private chain's, not the rendering's.
	require.NotEqual(t, renderHash(8, "a"), st.SafeL2.Hash)
	// The four borrowed ones are the rendering's, and the sequence number is non-trivial.
	require.Equal(t, uint64(8%epochBlocks), st.SafeL2.SequenceNumber)
	require.Equal(t, uint64(8/epochBlocks), st.SafeL2.L1Origin.Number)
	require.Equal(t, 1, h.m.claims)
}

// A claim cannot be served until the rendering has derived the block its ref is read from. That is
// not an implementation quirk: the claim leads its range, so the terminal block is a whole cadence
// above the block that carried the claim.
func TestCompletionWaitsForTheTerminalBlock(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 5, "a", 0)
	h.r.safe = 5
	require.NoError(t, h.step())
	require.Equal(t, wantGenesisRef(), h.status().SafeL2, "the claim is read, but its ref does not exist publicly yet")
	require.Equal(t, 1, h.m.claims, "the claim itself was read on the first pass and is not re-read")

	h.r.fill(6, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)
	require.Equal(t, 1, h.m.claims)
}

// Finality lags safety, and its gate is the claim's TERMINAL block.
func TestFinalityLagsBehindSafety(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.set(9, "a", 0, claimTx(t, 1, 9, 16))
	h.r.fill(10, 16, "a", 0)
	h.r.safe, h.r.finalized = 16, 0
	require.NoError(t, h.step())

	st := h.status()
	require.Equal(t, wantRef(16), st.SafeL2)
	require.Equal(t, wantGenesisRef(), st.FinalizedL2, "nothing beyond genesis is final while nothing on the rendering is")

	// Finalizing through the first claim's terminal block (rendering block 8) finalizes that claim
	// only.
	h.r.finalized = 8
	require.NoError(t, h.step())
	st = h.status()
	require.Equal(t, wantRef(8), st.FinalizedL2)
	require.Equal(t, wantRef(16), st.SafeL2)

	h.r.finalized = 16
	require.NoError(t, h.step())
	require.Equal(t, wantRef(16), h.status().FinalizedL2)
	require.Equal(t, uint64(16), h.m.finalized)
}

// A finalized ref must be a PURE FUNCTION OF FINALIZED-DEPTH INPUTS, which is why the gate is the
// claim's terminal block and not the block that carried it.
//
// Four of a served ref's six fields are borrowed from the rendering block at lastBlock, a whole
// cadence above the carrier. Gating on the carrier would publish a ref two of whose fields came
// from a finalized block and four from a merely-safe one, so an L1 reorg above the finalized height
// could change the borrowed four while the claim-borne hash stayed put — a changed finalized ref at
// an unchanged height. The consumer treats a finalized head as immutable at a height
// (engine_controller.applyFinalizedHeadCacheChecks returns the stale cached value on a matching ID
// and PANICS on a same-height hash conflict), so this is the module's job to make impossible.
func TestFinalizedNeverBorrowsFromABlockAboveTheFinalizedView(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8

	// The carrier (rendering block 1) is finalized; the terminal block (8) is only safe. Under the
	// carrier rule this claim would be finalized here, borrowing four fields from block 8.
	h.r.finalized = 1
	require.NoError(t, h.step())
	st := h.status()
	require.Equal(t, wantRef(8), st.SafeL2, "safe is fine: it only needs the terminal block derived")
	require.Equal(t, wantGenesisRef(), st.FinalizedL2, "finalized waits for the block its fields come from")
	require.Zero(t, h.m.finalized)

	// Walk the finalized view up to one block short of the terminal block: still not final.
	for n := uint64(2); n < 8; n++ {
		h.r.finalized = n
		require.NoError(t, h.step())
		require.Equal(t, wantGenesisRef(), h.status().FinalizedL2, "finalized view %d", n)
	}

	// The moment the finalized view reaches lastBlock, every one of the ref's six fields is a
	// finalized-depth fact, and it is published.
	h.r.finalized = 8
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().FinalizedL2)
}

// The contract with a sequencing follower: a rendering reorg re-derives what is above the rewind
// point, and never unsays what was already said.
func TestServedSafeIsMonotoneAcrossAReorg(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe, h.r.finalized = 8, 1
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)

	// Blocks 2..8 are replaced on a new fork. Everything at or below the finalized height (1) is
	// untouched, which is what makes the claim itself survive.
	for n := uint64(2); n <= 8; n++ {
		h.r.set(n, "b", 1)
	}
	h.r.safe = 8
	require.NoError(t, h.step())

	st := h.status()
	require.Equal(t, wantRef(8), st.SafeL2, "the served ref did not move and did not regress")
	require.Positive(t, h.m.reorgs)
	// The rewind target is the chain's own FINALIZED height, so a claim carried at or below it is
	// kept rather than re-read: an L1 reorg cannot reach it, and re-reading it could only produce
	// the same answer.
	require.Equal(t, 1, h.m.claims)
}

// Even a rendering that reorgs into a chain with NO claim at all cannot lower the served head.
func TestAReorgThatErasesAClaimStillCannotRegress(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe, h.r.finalized = 8, 0
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)

	// A deep reorg replaces block 1 itself, claim and all.
	h.r.fill(1, 8, "b", 0)
	h.r.safe = 8
	require.NoError(t, h.step())
	require.NoError(t, h.step())

	st := h.status()
	require.Equal(t, wantRef(8), st.SafeL2, "monotone: a served ref is never withdrawn")
	require.Positive(t, h.m.reorgs)
}

// A REVERTED postClaim never entered the registry's record, so there is nothing to follow. Under
// snap-to-commitment that is a skip with a metric, NOT the sidecar's fail-stop latch: the scan
// keeps going and a later, accepted claim is served normally.
func TestRevertedClaimIsSkippedAndCounted(t *testing.T) {
	h := newHarness(t)
	bad := claimTx(t, 0, 1, 8)
	h.r.set(1, "a", 0, bad)
	h.r.revert(1, bad)
	h.r.fill(2, 8, "a", 0)
	h.r.set(9, "a", 0, claimTx(t, 1, 9, 16))
	h.r.fill(10, 16, "a", 0)
	h.r.safe, h.r.finalized = 16, 16
	require.NoError(t, h.step())

	require.Equal(t, 1, h.m.rejected["reverted"])
	require.Equal(t, 1, h.m.claims, "only the accepted claim counted")
	st := h.status()
	require.Equal(t, wantRef(16), st.SafeL2, "the scan advanced past the reverted claim rather than latching")
	require.Equal(t, wantRef(16), st.FinalizedL2)
}

// Anyone may send bytes to a contract address. A registry-addressed transaction that is not a
// canonically-encoded postClaim is a loud log and a skip, never a crash and never a lenient read —
// otherwise one transaction is a denial-of-service switch.
func TestMalformedRegistryTransactionsAreSkipped(t *testing.T) {
	valid := claimTx(t, 3, 1, 8)
	wrongSelector := rawTx(0, registryAddr, append([]byte{0xde, 0xad, 0xbe, 0xef}, make([]byte, codec.EncodedSizeEmptyProof)...))
	tooShort := rawTx(1, registryAddr, []byte{0x01, 0x02})
	elsewhere := rawTx(4, common.HexToAddress("0xbeef"), valid.Data())

	body, err := render.EncodePostClaim(&codec.RangeClaim{FirstBlock: 1, LastBlock: 8})
	require.NoError(t, err)
	nonCanonical := rawTx(2, registryAddr, append(body, 0x00))

	h := newHarness(t)
	h.r.set(1, "a", 0, wrongSelector, tooShort, nonCanonical, valid, elsewhere)
	h.r.fill(2, 8, "a", 0)
	h.r.safe, h.r.finalized = 8, 8
	require.NoError(t, h.step())

	require.Equal(t, 2, h.m.rejected["selector"], "a wrong selector and a stub are both selector rejections")
	require.Equal(t, 1, h.m.rejected["decode"], "trailing bytes are not canonical form")
	require.Equal(t, 1, h.m.claims)
	require.Equal(t, wantRef(8), h.status().SafeL2)
}

// Claims are read only at or below the chain's own SAFE view, so L1 reorg handling is inherited
// from the supernode's derivation rather than reimplemented here.
func TestOnlyScansAtOrBelowSafe(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 0
	require.NoError(t, h.step())
	require.Equal(t, wantGenesisRef(), h.status().SafeL2, "a claim above the safe view is not read")

	h.r.safe = 8
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)
}

// current_l1 is forwarded from the chain's own view, and never regressed: a consumer that saw it go
// backwards would read it as a sequencer restart, and the value is a view rather than a commitment.
func TestCurrentL1IsForwardedAndNeverRegresses(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())
	require.Equal(t, eth.L1BlockRef{}, h.status().CurrentL1, "zero is legal, and it is what an unstarted chain reports")

	h.r.currentL1 = eth.L1BlockRef{Hash: l1Hash(5), Number: 5}
	require.NoError(t, h.step())
	require.Equal(t, uint64(5), h.status().CurrentL1.Number)

	h.r.currentL1 = eth.L1BlockRef{Hash: l1Hash(3), Number: 3}
	require.NoError(t, h.step())
	require.Equal(t, uint64(5), h.status().CurrentL1.Number, "held, not served backwards")

	h.r.currentL1 = eth.L1BlockRef{}
	require.NoError(t, h.step())
	require.Equal(t, uint64(5), h.status().CurrentL1.Number, "a silent chain is not forwarded as silence")
}

// Every failure is a skipped tick: the module reports the error to its caller, changes nothing, and
// tries again. Nothing here is worth taking a supernode down for.
func TestFailuresAreSkippedTicks(t *testing.T) {
	h := newHarness(t)
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe, h.r.finalized = 8, 8
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)

	h.r.statusErr = errors.New("chain is restarting")
	require.Error(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2, "the served state is untouched by a failed poll")

	h.r.statusErr = nil
	h.r.set(9, "a", 0, claimTx(t, 1, 9, 16))
	h.r.fill(10, 16, "a", 0)
	h.r.safe, h.r.finalized = 16, 16
	h.r.payloadErr[16] = errors.New("engine is busy")
	require.Error(t, h.step(), "the claim is read but its ref cannot be completed yet")
	require.Equal(t, wantRef(8), h.status().SafeL2)

	delete(h.r.payloadErr, 16)
	require.NoError(t, h.step())
	st := h.status()
	require.Equal(t, wantRef(16), st.SafeL2)
	require.Equal(t, wantRef(16), st.FinalizedL2)
}

// An unattached module is a skipped tick too, rather than a nil dereference: New and Attach are two
// steps because the chain container and the module are necessarily built in that order. It still
// serves its genesis ref, because that answer needs no data source at all.
func TestUnattachedModuleIsASkippedTick(t *testing.T) {
	m := New(Config{Registry: registryAddr, GenesisHash: privateGenesisHash()},
		testRollupCfg(), testlog.Logger(t, gethlog.LevelInfo), nil)
	require.Error(t, m.Step(context.Background()))
	st, err := m.SyncStatus()
	require.NoError(t, err)
	require.Equal(t, wantGenesisRef(), st.SafeL2)
}

// A module built with NO genesis hash is the one state with nothing true to say, and it says so.
// An operator cannot reach it -- CLIConfig.Check refuses an enabled group without the flag -- so this
// pins the library-caller path rather than an operator one.
func TestModuleWithoutAGenesisHashErrorsUntilTheFirstClaim(t *testing.T) {
	h := newHarnessWithConfig(t, Config{Registry: registryAddr})
	_, err := h.f.SyncStatus()
	require.ErrorIs(t, err, ErrNoGenesisRef)

	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())

	st, err := h.f.SyncStatus()
	require.NoError(t, err, "a claim gives it something true to say")
	require.Equal(t, wantRef(8), st.SafeL2)
}

// A claim whose terminal hash is anything at all is served VERBATIM. There is no private EL to
// check it against and, under snap-to-commitment, no check to make: the claim is the operator's
// binding statement, and a consumer that cannot find the block fail-stops itself.
func TestClaimsAreServedVerbatim(t *testing.T) {
	h := newHarness(t)
	odd := common.HexToHash("0xdead")
	oddParent := common.HexToHash("0xbeef")
	h.r.set(1, "a", 0, claimTxWithTerminal(t, 0, 1, 8, odd, oddParent))
	h.r.fill(2, 8, "a", 0)
	h.r.safe = 8
	require.NoError(t, h.step())

	st := h.status()
	require.Equal(t, odd, st.SafeL2.Hash)
	require.Equal(t, oddParent, st.SafeL2.ParentHash)
	require.Equal(t, uint64(8), st.SafeL2.Number)
}

// A scan bigger than one poll's budget makes progress incrementally rather than in one stall.
//
// Note what completion does NOT wait for: the ref for a claim read at block 1 is read straight from
// block 8, because the chain's safe head already vouches for block 8. The cursor's job is to find
// claims, not to walk to their refs.
func TestScanIsBounded(t *testing.T) {
	h := newHarnessWithConfig(t, Config{Registry: registryAddr, GenesisHash: privateGenesisHash(), MaxBlocksPerPoll: 3})
	h.r.set(1, "a", 0, claimTx(t, 0, 1, 8))
	h.r.fill(2, 8, "a", 0)
	h.r.set(9, "a", 0, claimTx(t, 1, 9, 16))
	h.r.fill(10, 16, "a", 0)
	h.r.safe = 16

	require.Equal(t, wantGenesisRef(), h.status().SafeL2)
	// Blocks 0,1,2: the first claim is found and completed at once.
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)

	// Blocks 3-5 and 6-8: still nothing new, because the second claim is at block 9.
	require.NoError(t, h.step())
	require.NoError(t, h.step())
	require.Equal(t, wantRef(8), h.status().SafeL2)

	// Blocks 9-11: the second claim is reached.
	require.NoError(t, h.step())
	require.Equal(t, wantRef(16), h.status().SafeL2)
}
