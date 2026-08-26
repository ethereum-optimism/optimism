package silhouette

import (
	"context"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// THE ONE-HOUR BOMB, BOTH DIRECTIONS.
//
// This is the failure mode the sequencer posture exists to disarm, driven through the REAL stock
// derivation stage rather than described. It is not an end-to-end test on purpose: at cluster scale
// the bomb is a timer, and a test that waited out a sequencing window would be slow, flaky, and
// would prove the same thing less clearly. What has to be shown is a mechanism, and the mechanism is
// two exported calls.
//
// The shape of the deployment, reproduced exactly:
//
//   - the chain has NO BATCHER, so the batch inbox is empty forever. That is the fake provider
//     below returning io.EOF: not "no data yet", but "no data, ever".
//   - the chain's SAFE HEAD therefore never advances. That is the parent staying put.
//   - L1 keeps moving. That is the origin advancing.
//
// Under the committed window the stage eventually manufactures an empty batch for an epoch it has no
// data for, and consolidation compares that against the real block the sequencer produced at that
// height and reorgs the real one out. Under the posture's window it never does.

// emptyInbox is a NextBatchProvider for a chain nobody batches: it has an L1 origin, it advances,
// and it never has a batch.
//
// io.EOF rather than an error, because that is what an empty inbox really looks like to the stage —
// every upstream stage returned EOF, the pipeline advanced the origin, and the batch stage is asked
// again at the new origin. An error here would exercise a different path.
type emptyInbox struct{ origin eth.L1BlockRef }

func (e *emptyInbox) Origin() eth.L1BlockRef { return e.origin }
func (e *emptyInbox) NextBatch(context.Context) (derive.Batch, error) {
	return nil, io.EOF
}
func (e *emptyInbox) FlushChannel() {}

// frozenSafeHead is the SafeBlockFetcher for a chain whose safe head does not move. The batch stage
// only reads it on the paths a real batch takes, so answering is enough.
type frozenSafeHead struct{ safe eth.L2BlockRef }

func (f *frozenSafeHead) L2BlockRefByNumber(_ context.Context, n uint64) (eth.L2BlockRef, error) {
	if n != f.safe.Number {
		return eth.L2BlockRef{}, ethNotFound
	}
	return f.safe, nil
}

func (f *frozenSafeHead) PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, ethNotFound
}

var ethNotFound = io.EOF

// runUntilForcedBatch drives the real post-Holocene batch stage over an advancing L1 with an empty
// inbox, and reports the first batch it manufactures, if any.
//
// `blocks` is how many L1 blocks past the parent's own origin to walk. The window predicate is
// `epoch + seqWindow <= origin`, so a walk longer than the window is what arms it.
func runUntilForcedBatch(t *testing.T, seqWindow uint64, blocks uint64) *derive.SingularBatch {
	t.Helper()
	cfg := silhouetteRollupConfig()
	cfg.SeqWindowSize = seqWindow

	// The parent is the chain's frozen safe head: genesis, at the rollup config's genesis L1 origin.
	parent := eth.L2BlockRef{
		Hash:     cfg.Genesis.L2.Hash,
		Number:   cfg.Genesis.L2.Number,
		Time:     cfg.Genesis.L2Time,
		L1Origin: cfg.Genesis.L1,
	}
	inbox := &emptyInbox{origin: l1Ref(cfg.Genesis.L1.Number)}
	stage := derive.NewBatchStage(testlog.Logger(t, log.LevelError), cfg, inbox, &frozenSafeHead{safe: parent})
	require.ErrorIs(t, stage.Reset(context.Background(), inbox.origin, cfg.Genesis.SystemConfig), io.EOF)

	for i := uint64(0); i <= blocks; i++ {
		inbox.origin = l1Ref(cfg.Genesis.L1.Number + i)
		batch, _, err := stage.NextBatch(context.Background(), parent)
		if err == nil {
			require.NotNil(t, batch)
			return batch
		}
		require.ErrorIs(t, err, io.EOF, "the stage failed for a reason other than having no data")
	}
	return nil
}

func l1Ref(num uint64) eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l1Hash(num),
		Number:     num,
		ParentHash: l1Hash(num - 1),
		Time:       l1Time(num),
	}
}

// TestTheEmptyBatchBombIsArmedByTheCommittedWindow is the CONTROL, and without it the treatment
// below is a test that passes because nothing was ever going to happen.
//
// It asserts the failure the checkpoint log describes: with the committed window, an empty inbox and
// a frozen safe head, stock derivation manufactures a batch with no transactions in it for an epoch
// it holds no data for. On a verifier that is DR-2's designed liveness. On the node that sequences
// the chain it is a claim about history that contradicts the blocks in its own execution client.
func TestTheEmptyBatchBombIsArmedByTheCommittedWindow(t *testing.T) {
	t.Parallel()
	// Two windows' worth of L1, so the expiry is passed rather than grazed.
	forced := runUntilForcedBatch(t, seqWindow, 2*seqWindow)

	require.NotNil(t, forced,
		"the control failed: stock derivation did not force a batch, so the treatment proves nothing")
	require.Empty(t, forced.Transactions,
		"the forced batch is supposed to be empty — that is what makes it contradict the real chain")
	require.Equal(t, silhouetteRollupConfig().Genesis.L2.Hash, forced.ParentHash)
	t.Logf("committed window %d: derivation forced an empty batch at epoch %d, timestamp %d",
		seqWindow, forced.EpochNum, forced.Timestamp)
}

// TestTheSequencerPostureWindowDisarmsIt is the treatment: the same stage, the same empty inbox, the
// same frozen safe head, an L1 walked far past where the committed window expired — and no batch.
//
// The walk is deliberately longer than the control's: this must not pass merely because the test did
// not wait long enough, which is the one way a disarm test can lie.
func TestTheSequencerPostureWindowDisarmsIt(t *testing.T) {
	t.Parallel()
	forced := runUntilForcedBatch(t, SequencerPostureSeqWindow, 4*seqWindow)

	require.Nil(t, forced,
		"the posture's window did not disarm the bomb: derivation still manufactured a batch")
}

// TestAForcedEmptyBatchWouldReorgTheRealChain closes the loop from "derivation forced a batch" to
// "the real chain is gone", which is the step that makes the control a catastrophe rather than an
// oddity.
//
// The forced batch carries no transactions. The block the sequencer really produced at that height
// carries at least the L1-info transaction and, on a live chain, user transactions. Consolidation
// compares the two and, on a mismatch, rewinds the unsafe chain onto the derived attributes
// (op-node/rollup/attributes, reorgOutUnsafeChain). This asserts the comparison directly, because
// that comparison IS the reorg trigger — and it asserts the matching case too, so that "these
// mismatch" is a statement about their contents rather than about the comparison refusing everything.
func TestAForcedEmptyBatchWouldReorgTheRealChain(t *testing.T) {
	t.Parallel()
	cfg := silhouetteRollupConfig()
	logger := testlog.Logger(t, log.LevelError)

	parentHash := cfg.Genesis.L2.Hash
	// One transaction, standing in for the L1-info transaction every real block begins with. Its
	// bytes do not have to be a valid transaction for the count comparison that fires first.
	realTx := eth.Data(common.FromHex("0x7ef8f8a0deadbeef"))
	real := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:   parentHash,
		BlockNumber:  1,
		Timestamp:    hexutil.Uint64(cfg.Genesis.L2Time + cfg.BlockTime),
		Transactions: []eth.Data{realTx},
	}}

	// What derivation builds from a forced empty batch: no transactions at all.
	empty := &eth.PayloadAttributes{
		Timestamp:    real.ExecutionPayload.Timestamp,
		Transactions: nil,
	}
	err := attributes.AttributesMatchBlock(cfg, empty, parentHash, real, logger)
	require.Error(t, err, "an empty derived block matched a real one; the reorg trigger is not this")
	require.ErrorContains(t, err, "transaction count does not match")
	t.Logf("consolidation verdict on a forced empty batch against the real block: %v", err)

	// The control. Attributes that DO carry the block's transaction get PAST the transaction checks
	// and fail on the next thing this bare fixture does not fill in — which is the point: it shows
	// that the verdict above was reached on the emptiness rather than by a comparison that refuses
	// everything it is handed.
	matching := &eth.PayloadAttributes{
		Timestamp:    real.ExecutionPayload.Timestamp,
		Transactions: []eth.Data{realTx},
	}
	err = attributes.AttributesMatchBlock(cfg, matching, parentHash, real, logger)
	require.Error(t, err, "the fixture is too complete to be a control; make it barer")
	require.NotContains(t, err.Error(), "transaction",
		"a matching transaction list must clear the transaction checks")
}

// TestTheDisarmIsPostureScoped is the fence around the fix. The window that disarms the bomb is also
// the window the forced-extension convention is measured in, and that convention is a PUBLIC rule —
// so the disarm must reach the sequencer's pipeline and nothing else. Assemble's `committed` copy is
// what makes that true; this asserts it at the level the bomb lives at.
func TestTheDisarmIsPostureScoped(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+10, LabelsFromProvenHead)

	// The pipeline the bomb would fire in: disarmed.
	require.Equal(t, SequencerPostureSeqWindow, e.vncfg.Rollup.SeqWindowSize)
	// The rule every verifier and the prover also compute: untouched.
	require.Equal(t, seqWindow, e.a.Source.rollup.SeqWindowSize)
	// And the untouched one is still the one that arms the generator, which is the property that makes
	// the forced extension a liveness backstop rather than a dead branch.
	require.NotNil(t, runUntilForcedBatch(t, e.a.Source.rollup.SeqWindowSize, 2*seqWindow))
}
