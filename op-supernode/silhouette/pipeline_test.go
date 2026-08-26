package silhouette

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// Gate 2: a REAL op-node derivation pipeline, fed by the injected source, deriving attributes.
//
// This is not a pipeline-shaped test double. It is derive.NewDerivationPipeline with the actual
// stage list — L1Traversal, L1Retrieval, FrameQueue, ChannelMux, ChannelInReader, BatchMux,
// AttributesQueue and the FetchingAttributesBuilder — with exactly one thing replaced: the L1 data
// source. Everything the frames pass through on their way to payload attributes is stock code
// running for real, which is the whole architectural claim.
//
// The one thing stood in for is the ENGINE, and that is the G3 boundary rather than a shortcut: the
// shim EL does not exist yet, so the harness plays its part by serving each derived block back from
// the proof-committed facts, which is precisely what the shim will do. That makes this harness a
// small integration of source + fact store + stock derivation, and it is why the assertions can
// check that op-node's L2 chain carries P's REAL block hashes.

// factsL2 stands in for the shim EL. It answers the L2 queries derivation makes, serving each block
// from the fact store — real hash, real timestamp, rendered origin — and nothing else.
//
// It never executes anything, and it holds no state root it did not read off the wire. That is the
// fail-stop shape DR-1 asks of the real shim: a block it has no fact for is a block it refuses to
// describe.
type factsL2 struct {
	env *testEnv
}

func (f *factsL2) genesisRef() eth.L2BlockRef {
	g := f.env.rollup.Genesis
	return eth.L2BlockRef{
		Hash:           g.L2.Hash,
		Number:         g.L2.Number,
		ParentHash:     common.Hash{},
		Time:           g.L2Time,
		L1Origin:       g.L1,
		SequenceNumber: 0,
	}
}

func (f *factsL2) refOf(number uint64) (eth.L2BlockRef, error) {
	if number == f.env.rollup.Genesis.L2.Number {
		return f.genesisRef(), nil
	}
	fact, ok := f.env.facts.ByNumber(number)
	if !ok {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	parent := f.env.rollup.Genesis.L2.Hash
	if number > f.env.rollup.Genesis.L2.Number+1 {
		p, ok := f.env.facts.ByNumber(number - 1)
		if !ok {
			return eth.L2BlockRef{}, ethereum.NotFound
		}
		parent = p.Hash
	}
	return eth.L2BlockRef{
		Hash:           fact.Hash,
		Number:         fact.Number,
		ParentHash:     parent,
		Time:           fact.Timestamp,
		L1Origin:       fact.L1Origin,
		SequenceNumber: fact.SeqNumber,
	}, nil
}

func (f *factsL2) L2BlockRefByLabel(_ context.Context, _ eth.BlockLabel) (eth.L2BlockRef, error) {
	return f.genesisRef(), nil
}

func (f *factsL2) L2BlockRefByHash(_ context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	if hash == f.env.rollup.Genesis.L2.Hash {
		return f.genesisRef(), nil
	}
	fact, ok := f.env.facts.ByHash(hash)
	if !ok {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return f.refOf(fact.Number)
}

func (f *factsL2) L2BlockRefByNumber(_ context.Context, num uint64) (eth.L2BlockRef, error) {
	return f.refOf(num)
}

// SystemConfigL2Fetcher: P's SystemConfig is FROZEN, so every block reports the genesis one. On a
// real deployment this comes back out of the parent header's extraData and L1-info transaction, and
// TestForcedBlockExtraDataRoundTripsThroughOpNode is what checks that path agrees with this answer.
func (f *factsL2) SystemConfigByL2Hash(_ context.Context, _ common.Hash) (eth.SystemConfig, error) {
	return f.env.sysCfg, nil
}

func (f *factsL2) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	ref, err := f.L2BlockRefByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return f.payload(ref), nil
}

func (f *factsL2) PayloadByNumber(ctx context.Context, num uint64) (*eth.ExecutionPayloadEnvelope, error) {
	ref, err := f.refOf(num)
	if err != nil {
		return nil, err
	}
	return f.payload(ref), nil
}

func (f *factsL2) payload(ref eth.L2BlockRef) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:  ref.ParentHash,
		BlockNumber: hexutil.Uint64(ref.Number),
		BlockHash:   ref.Hash,
		Timestamp:   hexutil.Uint64(ref.Time),
		GasLimit:    hexutil.Uint64(f.env.sysCfg.GasLimit),
	}}
}

// staticDepSet is the whole DependencySet interface: derivation asks only how many chains are in it,
// to decide whether the interop activation bundle includes the multi-chain wrappers.
type staticDepSet struct{ chains []eth.ChainID }

func (s staticDepSet) Chains() []eth.ChainID { return s.chains }

// FetchReceipts completes derive.L1Fetcher. A silhouette chain's origins are deposit-free by
// construction (DR-2, gated portal), so this returns none — and that is the property the whole
// no-deposit design rests on, exercised here rather than asserted.
func (f *fakeL1) FetchReceipts(_ context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	num, ok := f.byHash(hash)
	if !ok {
		return nil, nil, ethereum.NotFound
	}
	return f.info(num), optypes.Receipts{}, nil
}

func (f *fakeL1) L1BlockRefByLabel(_ context.Context, _ eth.BlockLabel) (eth.L1BlockRef, error) {
	return f.ref(f.head), nil
}

// realPipeline assembles the stock pipeline with only the data source replaced.
func (e *testEnv) realPipeline(t *testing.T) (*derive.DerivationPipeline, *factsL2) {
	l2 := &factsL2{env: e}
	pipeline := derive.NewDerivationPipeline(
		testlog.Logger(t, 3),
		e.rollup,
		staticDepSet{chains: []eth.ChainID{eth.ChainIDFromBig(e.rollup.L2ChainID)}},
		e.l1,
		e.blobs,
		altda.NewAltDA(testlog.Logger(t, 3), altda.CLIConfig{}, altda.Config{}, &altda.NoopMetrics{}),
		l2,
		metrics.NoopMetrics,
		sepoliaChainConfig(),
		derive.WithDataSource(e.src),
	)
	pipeline.ConfirmEngineReset()
	return pipeline, l2
}

// runPipeline drives the pipeline until it has produced `want` sets of attributes or gone idle,
// advancing the safe head from the fact store as each set comes out — which is the part the shim EL
// will do for real.
func runPipeline(t *testing.T, e *testEnv, pipeline *derive.DerivationPipeline, l2 *factsL2, want int) []*derive.AttributesWithParent {
	t.Helper()
	var out []*derive.AttributesWithParent
	safe := l2.genesisRef()
	for step := 0; step < 4000 && len(out) < want; step++ {
		attrib, err := pipeline.Step(context.Background(), safe)
		if err != nil {
			// io.EOF means "blocked on new L1 data", and a temporary/reset error is the pipeline asking
			// to be driven again — neither is a failure of this harness.
			if errors.Is(err, io.EOF) || errors.Is(err, derive.ErrTemporary) || errors.Is(err, derive.ErrReset) {
				continue
			}
			require.NoError(t, err, "pipeline step %d", step)
		}
		if attrib == nil {
			continue
		}
		out = append(out, attrib)
		// Play the engine: the block the attributes describe now exists, with the REAL hash the proof
		// committed to. Feeding that back is what makes the next batch's parent-hash check meaningful.
		next, err := l2.refOf(uint64(attrib.Attributes.Timestamp-hexutil.Uint64(e.rollup.Genesis.L2Time))/e.rollup.BlockTime + e.rollup.Genesis.L2.Number)
		if err == nil {
			safe = next
		}
	}
	return out
}

// TestStockPipelineDerivesFromASyntheticEnvelope is gate 2(a): a synthetic proof-batch envelope,
// through the real stage list, out as payload attributes.
func TestStockPipelineDerivesFromASyntheticEnvelope(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	pipeline, l2 := e.realPipeline(t)
	attribs := runPipeline(t, e, pipeline, l2, 3)
	require.Len(t, attribs, 3, "the stock pipeline must derive one block per proven block")

	for i, a := range attribs {
		attr := a.Attributes
		blk := batch.Blocks[i]

		// The derived block is the proven block: same number, same timestamp.
		require.Equal(t, blk.Timestamp, uint64(attr.Timestamp), "block %d timestamp", i)

		// The parent is the previous PROVEN block's real wire hash (genesis for the first). This is
		// the assertion that matters: stock derivation, with nothing re-hashing a payload, is
		// carrying P's real identity.
		wantParent := e.rollup.Genesis.L2.Hash
		if i > 0 {
			wantParent = batch.Blocks[i-1].Hash
		}
		require.Equal(t, wantParent, a.Parent.Hash, "block %d parent hash", i)

		// Exactly one transaction, and it is the stock L1-info deposit: no user deposits exist
		// (DR-2, gated portal) and the batch carried no transactions, so this is the whole block.
		require.Len(t, attr.Transactions, 1, "block %d must be single-transaction", i)
		require.Equal(t, byte(optypes.DepositTxType), attr.Transactions[0][0], "block %d tx 0 must be a deposit", i)

		// The L1-info transaction must parse back with the stock parser, and name the origin this
		// source rendered.
		fact, ok := e.facts.ByNumber(blk.Number)
		require.True(t, ok)
		info, err := derive.L1BlockInfoFromBytes(e.rollup, uint64(attr.Timestamp), depositData(t, attr.Transactions[0]))
		require.NoError(t, err, "block %d L1-info tx must parse with the stock parser", i)
		require.Equal(t, fact.L1Origin.Number, info.Number, "block %d origin number", i)
		require.Equal(t, fact.L1Origin.Hash, info.BlockHash, "block %d origin hash", i)
		require.Equal(t, fact.SeqNumber, info.SequenceNumber, "block %d sequence number", i)

		// Stock attribute fields that must be right for the chain to be a real OP chain.
		require.Equal(t, eth.Bytes32(sepoliaRandao(fact.L1Origin.Number)), attr.PrevRandao, "block %d prevRandao", i)
		require.NotNil(t, attr.GasLimit)
		require.Equal(t, e.sysCfg.GasLimit, uint64(*attr.GasLimit), "block %d gas limit", i)
		require.NotNil(t, attr.Withdrawals)
		require.Empty(t, *attr.Withdrawals, "Canyon+ requires an empty withdrawals list")
		require.True(t, attr.NoTxPool, "a derived block never opens the tx pool")
	}
}

// TestStockPipelineRejectsATamperedEnvelopeAndContinues is gate 3 at pipeline level: a bad batch
// must not stop the real pipeline, and the good batch behind it must still derive.
func TestStockPipelineRejectsATamperedEnvelopeAndContinues(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)

	bad := e.goodBatch()
	bad.proof = []byte{0xba, 0xd0} // attested mode is "no proof", never "any proof"
	e.plantSpec(bad)

	good := e.goodBatch()
	good.carrier = l1GenesisNum + 6
	batch := e.buildBatch(good)
	e.plant(batch, good)

	pipeline, l2 := e.realPipeline(t)
	attribs := runPipeline(t, e, pipeline, l2, 3)
	require.Len(t, attribs, 3, "the pipeline must derive past a rejected batch")
	require.Equal(t, batch.Blocks[0].Timestamp, uint64(attribs[0].Attributes.Timestamp))
}

// depositData pulls the calldata out of an opaque deposit transaction.
func depositData(t *testing.T, opaque hexutil.Bytes) []byte {
	t.Helper()
	dep, err := optypes.UnmarshalDepositTx(opaque)
	require.NoError(t, err)
	return dep.Data
}
