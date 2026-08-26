package silhouette

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// THE COMPLIANCE ORACLE.
//
// The shim's contract is not "implement the engine API spec"; it is "answer what op-service/sources
// sends". So every gate in this file drives the shim through the STOCK clients — `sources.EngineClient`
// for the build dance and its embedded `sources.L2Client` for the query surface — over a real
// JSON-RPC round trip (an in-process geth pipe: the whole RPC with the socket removed). A shim that
// satisfied a hand-written expectation but not those clients would be useless, and one that satisfies
// those clients is by construction what op-node needs.

// shimL1 is the L1 access a shim-backed node needs: the pipeline's L1Fetcher, plus the headers-only
// reads the forced-extension convention makes. The synthetic ladder and sources.L1Client both satisfy
// it, which is what lets the same harness run against a fixture and against real Sepolia.
type shimL1 interface {
	derive.L1Fetcher
	L1Headers
}

// shimEnv is a shim over a fact store, served over RPC, with the stock op-node engine client on the
// other end.
type shimEnv struct {
	t      *testing.T
	rollup *rollup.Config
	sysCfg eth.SystemConfig
	l1     shimL1
	blobs  derive.L1BlobsFetcher
	src    derive.DataAvailabilitySource
	facts  *FactStore

	shim *Shim
	srv  *rpc.Server
	rpc  client.RPC
	eng  *sources.EngineClient
}

func (e *testEnv) newShim(t *testing.T) *shimEnv {
	t.Helper()
	return newShimEnv(t, e.rollup, e.sysCfg, e.l1, e.blobs, e.src, e.facts)
}

func newShimEnv(t *testing.T, rollupCfg *rollup.Config, sysCfg eth.SystemConfig, l1 shimL1,
	blobs derive.L1BlobsFetcher, src derive.DataAvailabilitySource, facts *FactStore,
) *shimEnv {
	t.Helper()
	logger := testlog.Logger(t, 3)
	shim := NewShim(logger, rollupCfg, sepoliaChainConfig(), sysCfg, l1, facts)
	cl, srv, err := shim.InProc()
	require.NoError(t, err)
	t.Cleanup(srv.Stop)

	// The engine client's own default config, unmodified: this is where trustCache=true comes from
	// ("engine is trusted, no need to recompute responses", sources/engine_client.go:21-26) and it is
	// the configuration a real op-node uses for its L2 engine.
	eng, err := sources.NewEngineClient(cl, logger, nil, sources.EngineClientDefaultConfig(rollupCfg))
	require.NoError(t, err)
	return &shimEnv{
		t: t, rollup: rollupCfg, sysCfg: sysCfg, l1: l1, blobs: blobs, src: src, facts: facts,
		shim: shim, srv: srv, rpc: cl, eng: eng,
	}
}

// genesisRef is the chain's starting point as the shim reports it.
func (se *shimEnv) genesisRef(t *testing.T) eth.L2BlockRef {
	t.Helper()
	ref, err := se.eng.L2BlockRefByLabel(context.Background(), eth.Unsafe)
	require.NoError(t, err)
	return ref
}

// buildOne runs the STOCK build dance for one block, exactly in the order op-node runs it
// (op-node/rollup/engine/build_sealed.go:29-39): fcU(attrs) → getPayload → newPayload → fcU. Every
// call goes through sources.EngineClient, so the versions (V3/V4/V5) are chosen by op-node's own fork
// gating rather than by this test.
func (se *shimEnv) buildOne(t *testing.T, parent eth.L2BlockRef, attrs *eth.PayloadAttributes) (eth.L2BlockRef, *eth.ExecutionPayloadEnvelope) {
	t.Helper()
	ctx := context.Background()
	fc := eth.ForkchoiceState{HeadBlockHash: parent.Hash, SafeBlockHash: parent.Hash, FinalizedBlockHash: parent.Hash}

	res, err := se.eng.ForkchoiceUpdate(ctx, &fc, attrs)
	require.NoError(t, err, "fcU with attributes")
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
	require.NotNil(t, res.PayloadID, "attributes must open a build job")

	env, err := se.eng.GetPayload(ctx, eth.PayloadInfo{ID: *res.PayloadID, Timestamp: uint64(attrs.Timestamp)})
	require.NoError(t, err, "getPayload")

	// The CL's own sanity check on a sealed payload, and its own ref decode: if the shim's envelope
	// did not parse here, nothing downstream would work.
	ref, err := derive.PayloadToBlockRef(se.rollup, env.ExecutionPayload)
	require.NoError(t, err, "the sealed payload must decode to an L2BlockRef with the stock parser")

	status, err := se.eng.NewPayload(ctx, env.ExecutionPayload, env.ParentBeaconBlockRoot)
	require.NoError(t, err, "newPayload")
	require.Equal(t, eth.ExecutionValid, status.Status, "newPayload validation error: %v", status.ValidationError)
	require.NotNil(t, status.LatestValidHash)
	require.Equal(t, ref.Hash, *status.LatestValidHash)

	fc = eth.ForkchoiceState{HeadBlockHash: ref.Hash, SafeBlockHash: ref.Hash, FinalizedBlockHash: parent.Hash}
	res, err = se.eng.ForkchoiceUpdate(ctx, &fc, nil)
	require.NoError(t, err, "fcU without attributes")
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
	require.Nil(t, res.PayloadID, "a forkchoice update with no attributes opens no build job")
	return ref, env
}

// deriveAndBuild runs the REAL derivation pipeline with the shim as its L2 source, and builds every
// block it derives through the real build dance. This is the harness swap in its simplest form: G2's
// `factsL2` stub is gone and the stock engine client over the shim's RPC stands where it stood.
func (se *shimEnv) deriveAndBuild(t *testing.T, want int) []eth.L2BlockRef {
	t.Helper()
	pipeline := se.pipelineOverShim(t)
	safe := se.genesisRef(t)
	var built []eth.L2BlockRef
	for step := 0; step < 4000 && len(built) < want; step++ {
		attrib, err := pipeline.Step(context.Background(), safe)
		if err != nil {
			if isPipelineIdle(err) {
				continue
			}
			require.NoError(t, err, "pipeline step %d", step)
		}
		if attrib == nil {
			continue
		}
		require.Equal(t, safe.Hash, attrib.Parent.Hash, "the pipeline must build on the head the shim served")
		ref, _ := se.buildOne(t, safe, attrib.Attributes)
		built = append(built, ref)
		safe = ref
	}
	return built
}

// TestShimServesTheComplianceOracle is the per-method gate: the stock clients' whole read surface,
// over the shim, on a chain of proven blocks built through the real dance.
func TestShimServesTheComplianceOracle(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	built := se.deriveAndBuild(t, 3)
	require.Len(t, built, 3)
	ctx := context.Background()

	// eth_chainId.
	chainID, err := se.eng.ChainID(ctx)
	require.NoError(t, err)
	require.Equal(t, e.rollup.L2ChainID, chainID)

	for i, blk := range batch.Blocks {
		fact, ok := e.facts.ByNumber(blk.Number)
		require.True(t, ok)

		// L2BlockRefByNumber / ByHash / ByLabel — the walk FindL2Heads runs on. The hash it reports is
		// the hash the PROOF committed to: real identity, carried by stock code.
		byNum, err := se.eng.L2BlockRefByNumber(ctx, blk.Number)
		require.NoError(t, err)
		require.Equal(t, blk.Hash, byNum.Hash, "block %d must report its real wire hash", blk.Number)
		require.Equal(t, blk.Timestamp, byNum.Time)
		require.Equal(t, fact.L1Origin, byNum.L1Origin, "the rendered origin must survive the round trip")
		require.Equal(t, fact.SeqNumber, byNum.SequenceNumber)
		wantParent := e.rollup.Genesis.L2.Hash
		if i > 0 {
			wantParent = batch.Blocks[i-1].Hash
		}
		require.Equal(t, wantParent, byNum.ParentHash)

		byHash, err := se.eng.L2BlockRefByHash(ctx, blk.Hash)
		require.NoError(t, err)
		require.Equal(t, byNum, byHash, "by-number and by-hash must agree")

		// SystemConfigByL2Hash — the reconstruction op-node does on EVERY block, out of the parent
		// header's extraData and the L1-info transaction (derive.PayloadToSystemConfig). This is the
		// path the dead in-header marker would have poisoned (G2 D8), so it is the assertion that
		// matters most for the header's shape.
		sysCfg, err := se.eng.SystemConfigByL2Hash(ctx, blk.Hash)
		require.NoError(t, err, "block %d must reconstruct a SystemConfig", blk.Number)
		require.Equal(t, e.sysCfg.GasLimit, sysCfg.GasLimit)
		require.Equal(t, e.sysCfg.EIP1559Params, sysCfg.EIP1559Params,
			"a served header must reconstruct the FROZEN eip-1559 parameters")
		require.Equal(t, e.sysCfg.MinBaseFee, sysCfg.MinBaseFee)
		require.Equal(t, e.sysCfg.Scalar, sysCfg.Scalar, "the fee scalars must round-trip exactly")
		require.Equal(t, e.sysCfg.BatcherAddr, sysCfg.BatcherAddr)

		// PayloadByNumber / ByHash — the full-tx RPCBlock decode, including the body the CL supplied
		// at build time coming back verbatim.
		payload, err := se.eng.PayloadByNumber(ctx, blk.Number)
		require.NoError(t, err)
		require.Equal(t, blk.Hash, payload.ExecutionPayload.BlockHash)
		require.Equal(t, eth.Bytes32(blk.StateRoot), payload.ExecutionPayload.StateRoot)
		require.NotNil(t, payload.ExecutionPayload.WithdrawalsRoot)
		require.Equal(t, blk.MessagePasserStorageRoot, *payload.ExecutionPayload.WithdrawalsRoot,
			"Isthmus puts the message-passer storage root in the header, and that is what the wire carries")
		require.Len(t, payload.ExecutionPayload.Transactions, 1,
			"a silhouette block's public body is the L1-info deposit and nothing else")
		require.NotNil(t, payload.ExecutionPayload.Withdrawals)
		require.Empty(t, *payload.ExecutionPayload.Withdrawals, "Canyon+ requires an empty withdrawals list")
		byHashPayload, err := se.eng.PayloadByHash(ctx, blk.Hash)
		require.NoError(t, err)
		require.NoError(t, payload.ExecutionPayload.CheckEqual(byHashPayload.ExecutionPayload))

		// The L1-info transaction parses back with the stock parser and names the rendered origin.
		info, err := derive.L1BlockInfoFromBytes(e.rollup, blk.Timestamp,
			depositData(t, hexutil.Bytes(payload.ExecutionPayload.Transactions[0])))
		require.NoError(t, err)
		require.Equal(t, fact.L1Origin.Number, info.Number)
		require.Equal(t, fact.L1Origin.Hash, info.BlockHash)
		require.Equal(t, fact.SeqNumber, info.SequenceNumber)

		// Header-only form (fullTx=false), the other of the two calls that carry the whole surface.
		hdr, err := se.eng.InfoByHash(ctx, blk.Hash)
		require.NoError(t, err)
		require.Equal(t, blk.Hash, hdr.Hash())
		require.Equal(t, blk.StateRoot, hdr.Root())
		require.Equal(t, uint64(0), hdr.GasUsed(), "nothing was executed")
	}
}

// TestShimServedHeadersProduceTheRealOutputRoots is the real-identity headline (gate 5).
//
// `OutputV0AtBlockNumber` is stock code with no idea any of this is happening: it fetches a header
// and, post-Isthmus, reads the message-passer storage root straight out of `withdrawalsRoot`
// (op-service/sources/l2_client.go:192-227). Over the shim's served headers it returns P's REAL
// output roots — the same values the proof batch committed to, the same values a settlement claim
// carries. No eth_getProof, no MPT, no P-awareness anywhere on the path.
func TestShimServedHeadersProduceTheRealOutputRoots(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	require.Len(t, se.deriveAndBuild(t, 3), 3)
	ctx := context.Background()

	for _, blk := range batch.Blocks {
		out, err := se.eng.OutputV0AtBlockNumber(ctx, blk.Number)
		require.NoError(t, err, "stock outputV0 over a served header, block %d", blk.Number)
		require.Equal(t, eth.Bytes32(blk.StateRoot), out.StateRoot)
		require.Equal(t, eth.Bytes32(blk.MessagePasserStorageRoot), out.MessagePasserStorageRoot)
		require.Equal(t, blk.Hash, out.BlockHash)
		require.Equal(t, blk.OutputRoot(), common.Hash(eth.OutputRoot(out)),
			"block %d: the output root stock code computes from the served header must be the one the "+
				"proof committed to, byte for byte", blk.Number)

		byHash, err := se.eng.OutputV0AtBlock(ctx, blk.Hash)
		require.NoError(t, err)
		require.Equal(t, out, byHash)
	}

	// And the batch's own claimed newOutputRoot — the value that goes on chain — comes back out of
	// the shim's last served header.
	last := batch.Blocks[len(batch.Blocks)-1]
	out, err := se.eng.OutputV0AtBlockNumber(ctx, last.Number)
	require.NoError(t, err)
	require.Equal(t, batch.NewOutputRoot, common.Hash(eth.OutputRoot(out)),
		"the settlement-facing value must be reproducible from the shim alone")
}

// TestShimHeadersDoNotReHash is the declared fabrication, asserted rather than glossed.
//
// A client told to VERIFY what the RPC gives it must reject these blocks, because the served header's
// interior describes a block nobody executed while the hash beside it is the real one. That is DR-1's
// honest price, and the test exists so that if a future change accidentally made headers re-hash, the
// change is noticed as a behaviour change rather than celebrated as a fix — the roots come off the
// wire, so a re-hashing header would mean a FABRICATED root somewhere.
func TestShimHeadersDoNotReHash(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	require.Len(t, se.deriveAndBuild(t, 1), 1)

	// The same client construction, with trustRPC flipped off: `L2ClientDefaultConfig(cfg, false)`.
	untrusting, err := sources.NewL2Client(se.rpc, testlog.Logger(t, 3), nil,
		sources.L2ClientDefaultConfig(e.rollup, false))
	require.NoError(t, err)
	_, err = untrusting.PayloadByNumber(context.Background(), batch.Blocks[0].Number)
	require.ErrorContains(t, err, "failed to verify block hash",
		"a verifying client must reject a proof-rendered header: the hash is real and the interior is a "+
			"rendering, and only trustCache makes that usable")
}

// TestShimFailsStopBeyondTheFacts is the fail-stop negative (gate 4): the engine refuses to describe
// a block no proof covers, and it refuses in the way the CL can recover from.
func TestShimFailsStopBeyondTheFacts(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	built := se.deriveAndBuild(t, 3)
	require.Len(t, built, 3)
	head := built[len(built)-1]
	ctx := context.Background()

	// Ask for one block past the proven head, with attributes that are perfectly well formed.
	attrs := se.attributesFor(t, head, l1GenesisNum+1)
	fc := eth.ForkchoiceState{HeadBlockHash: head.Hash, SafeBlockHash: head.Hash, FinalizedBlockHash: head.Hash}
	res, err := se.eng.ForkchoiceUpdate(ctx, &fc, attrs)
	require.NoError(t, err, "opening the job is fine: the fail-stop is a statement about the block, not the attributes")
	require.NotNil(t, res.PayloadID)

	_, err = se.eng.GetPayload(ctx, eth.PayloadInfo{ID: *res.PayloadID, Timestamp: uint64(attrs.Timestamp)})
	require.Error(t, err, "getPayload beyond the facts must ERROR, never fabricate")
	var rpcErr rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, int(eth.UnknownPayload), rpcErr.ErrorCode(),
		"the code must be UnknownPayload: op-node maps it to ErrSealExpired (build_seal.go), so the same "+
			"attributes can be re-attempted once the proof batch lands")
	require.ErrorContains(t, err, "forced block")

	// Nothing was recorded, and nothing was halted: a fail-stop is a refusal, not a corruption.
	_, ok := e.facts.ByNumber(head.Number + 1)
	require.False(t, ok, "a refused build must leave no fact behind")
	_, halted := se.shim.Halted()
	require.False(t, halted, "outrunning the facts is not the honesty assertion firing")
}

// TestShimNewPayloadRejectsAnUnknownHashAndHalts is the guarded edge (gate 4).
//
// rewind.go:195 on the Cove branch synthesises a replacement block by mutating ExtraData and
// re-hashing — the only code in the supernode that could hand this engine a hash outside the facts.
// It runs only on judge invalidation, and P is proof-trusted, so no path may reach it. E3's honesty
// assertion is kept in code exactly here: a payload at a known height, on a known parent, with a
// different hash, is refused AND halts the shim.
func TestShimNewPayloadRejectsAnUnknownHashAndHalts(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	var halts []error
	se.shim.OnHalt(func(err error) { halts = append(halts, err) })
	built := se.deriveAndBuild(t, 2)
	require.Len(t, built, 2)
	ctx := context.Background()

	// Take the payload the shim itself served for block 2 and mutate its ExtraData, exactly as the
	// replacement-block synthesiser does, then offer it back.
	payload, err := se.eng.PayloadByNumber(ctx, built[1].Number)
	require.NoError(t, err)
	replacement := *payload.ExecutionPayload
	replacement.ExtraData = eth.BytesMax32(append([]byte(nil), []byte("replaced")...))
	replacement.BlockHash = common.HexToHash("0xdeadbeef")

	status, err := se.eng.NewPayload(ctx, &replacement, payload.ParentBeaconBlockRoot)
	require.NoError(t, err, "the rejection is a STATUS, so the CL can act on it")
	require.Equal(t, eth.ExecutionInvalid, status.Status)
	require.NotNil(t, status.ValidationError)
	require.Contains(t, *status.ValidationError, "the proven-or-forced fact for that height is")

	reason, halted := se.shim.Halted()
	require.True(t, halted, "a replacement block on a known parent must halt the shim")
	require.ErrorContains(t, reason, "newPayload offered a replacement for block")
	require.Len(t, halts, 1, "the halt callback fires exactly once")

	// And a halted shim refuses everything, which is the point of halting.
	_, err = se.eng.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{HeadBlockHash: built[1].Hash}, nil)
	require.ErrorContains(t, err, "silhouette shim halted")
}

// TestShimRefusesAnUnknownForkchoiceHeadWithoutHalting: the CL is told to reset, not that a block is
// bad. `InvalidForkchoiceState` is the one answer that makes the engine controller re-run
// FindL2Heads (engine_controller.go:686-693) — and SYNCING, the stock EL's answer, is forbidden here.
func TestShimRefusesAnUnknownForkchoiceHeadWithoutHalting(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	se := e.newShim(t)

	_, err := se.eng.ForkchoiceUpdate(context.Background(),
		&eth.ForkchoiceState{HeadBlockHash: common.HexToHash("0xfeed")}, nil)
	require.Error(t, err)
	var rpcErr rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, int(eth.InvalidForkchoiceState), rpcErr.ErrorCode())
	_, halted := se.shim.Halted()
	require.False(t, halted, "an unknown head is this node's ignorance, not a bad block")
}

// TestShimForkchoiceRewindsCursorsAndReDerivesIdentically is the reorg gate (gate 4).
//
// A forkchoice update to an older head is what an L1 reorg looks like from the engine's side, after
// the pipeline has reset and the transcoder's chaining state has rewound with it (G2 D5). The
// cursors must retreat, the renderings above must be forgotten, and re-deriving must produce
// BYTE-IDENTICAL payloads — the shim is meant to be a pure function of (facts, attributes), which is
// what makes the dark-launch equality gate a comparison rather than an interpretation.
func TestShimForkchoiceRewindsCursorsAndReDerivesIdentically(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	pipeline := se.pipelineOverShim(t)
	safe := se.genesisRef(t)

	// Build the chain once, keeping every envelope and the attributes that produced it.
	var refs []eth.L2BlockRef
	var envs []*eth.ExecutionPayloadEnvelope
	var attrsSeen []*eth.PayloadAttributes
	for step := 0; step < 4000 && len(refs) < 3; step++ {
		attrib, err := pipeline.Step(context.Background(), safe)
		if err != nil && !isPipelineIdle(err) {
			require.NoError(t, err)
		}
		if attrib == nil {
			continue
		}
		ref, env := se.buildOne(t, safe, attrib.Attributes)
		refs, envs, attrsSeen = append(refs, ref), append(envs, env), append(attrsSeen, attrib.Attributes)
		safe = ref
	}
	require.Len(t, refs, 3)
	require.Equal(t, refs[2], se.shim.facts.Cursors().Unsafe)

	// Rewind to block 1 — a forkchoice update to an older head.
	ctx := context.Background()
	res, err := se.eng.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{
		HeadBlockHash: refs[0].Hash, SafeBlockHash: refs[0].Hash, FinalizedBlockHash: refs[0].Hash}, nil)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
	require.Equal(t, refs[0], se.shim.facts.Cursors().Unsafe, "the cursors must retreat with the head")
	_, held := e.facts.Rendering(refs[2].Hash)
	require.False(t, held, "renderings above the new head must be forgotten")
	_, ok := se.shim.factByHash(refs[2].Hash)
	require.True(t, ok, "the FACTS stay: an engine rewind is not a proof rewind")

	// Re-derive the same two blocks and compare byte for byte.
	for i := 1; i < 3; i++ {
		_, env := se.buildOne(t, refs[i-1], attrsSeen[i])
		require.NoError(t, env.ExecutionPayload.CheckEqual(envs[i].ExecutionPayload),
			"re-derivation of block %d must be byte-identical", refs[i].Number)
		require.Equal(t, envs[i].ParentBeaconBlockRoot, env.ParentBeaconBlockRoot)
	}
}

// TestShimRefusesReceiptsAndProofs: the two methods that are refused rather than answered, and why.
func TestShimRefusesReceiptsAndProofs(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	se := e.newShim(t)
	ctx := context.Background()

	var receipts any
	err := se.rpc.CallContext(ctx, &receipts, "eth_getBlockReceipts", "latest")
	require.ErrorContains(t, err, "display-only and never an ingestion source",
		"the LogsDB rule is binding: rendering-device receipts must never be ingestible")

	_, err = se.eng.GetProof(ctx, common.Address{}, nil, "latest")
	require.ErrorContains(t, err, "holds no state and no trie")
}

// TestShimSelfDeclaresAtTheServiceLayer: the mitigation DR-1 requires, in the place G2 D8 moved it
// to. The header carries NO marker — that is asserted here as a property of the served block, not
// just as an absence in the code.
func TestShimSelfDeclaresAtTheServiceLayer(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	require.Len(t, se.deriveAndBuild(t, 1), 1)
	ctx := context.Background()

	var version string
	require.NoError(t, se.rpc.CallContext(ctx, &version, "web3_clientVersion"))
	require.Contains(t, version, "silhouette")
	require.Contains(t, version, "executes nothing")

	var decl SelfDeclaration
	require.NoError(t, se.rpc.CallContext(ctx, &decl, "silhouette_selfDeclaration"))
	require.True(t, decl.ProofRendered)
	require.False(t, decl.ExecutesTransactions)
	require.False(t, decl.HeadersReHash, "the declaration must say the headers do not re-hash")
	require.Contains(t, decl.FabricatedFields, "receiptsRoot")
	require.Contains(t, decl.RealFields, "stateRoot")
	require.Equal(t, hexutil.Uint64(bigs.Uint64Strict(e.rollup.L2ChainID)), decl.L2ChainID)

	var blockDecl BlockDeclaration
	require.NoError(t, se.rpc.CallContext(ctx, &blockDecl, "silhouette_blockProvenance",
		rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(batch.Blocks[0].Number))))
	require.Equal(t, "proven", blockDecl.Provenance)
	require.True(t, blockDecl.RootsKnown)
	require.Equal(t, batch.Blocks[0].Hash, blockDecl.Hash)
	require.Equal(t, batch.Blocks[0].OutputRoot(), blockDecl.OutputRoot)
	require.NotNil(t, blockDecl.Carrier, "a proven block names the L1 block that carried its proof")
	require.Equal(t, spec.carrier, blockDecl.Carrier.Number)

	var status Status
	require.NoError(t, se.rpc.CallContext(ctx, &status, "silhouette_status"))
	require.False(t, status.Halted)
	require.Equal(t, batch.Blocks[0].Hash, status.Unsafe.Hash)

	// And the header itself carries no marker: extraData is the consensus-legal eip-1559 encoding, so
	// op-node's OWN SystemConfig reconstruction round-trips the frozen parameters (G2 D8).
	payload, err := se.eng.PayloadByNumber(ctx, batch.Blocks[0].Number)
	require.NoError(t, err)
	require.NotContains(t, string(payload.ExecutionPayload.ExtraData), "silhouette")
	sysCfg, err := derive.PayloadToSystemConfig(e.rollup, payload.ExecutionPayload)
	require.NoError(t, err)
	require.Equal(t, e.sysCfg.EIP1559Params, sysCfg.EIP1559Params)
}

// TestShimNeverReportsSyncing: `--syncmode=consensus-layer`, and the EL-sync state machine stays
// idle because the shim gives it nothing to react to.
func TestShimNeverReportsSyncing(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	se := e.newShim(t)
	var syncing any
	require.NoError(t, se.rpc.CallContext(context.Background(), &syncing, "eth_syncing"))
	require.Equal(t, false, syncing)
}

// TestShimGetPayloadV5WhenKarstIsActive: the generated silhouette rollup config activates Karst at
// genesis, which makes getPayloadV5 the version op-node's own gating selects
// (op-node/rollup/types.go:777-780). Registering only V4 would leave a real deployment unable to
// seal a single block, so the version choice is made by the stock client here rather than asserted.
func TestShimGetPayloadV5WhenKarstIsActive(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	zero := uint64(0)
	e.rollup.KarstTime = &zero
	e.rollup.LagoonTime = &zero
	require.Equal(t, eth.GetPayloadV5, e.rollup.GetPayloadVersion(e.rollup.Genesis.L2Time),
		"the fixture must actually exercise V5")

	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)
	se := e.newShim(t)
	built := se.deriveAndBuild(t, 1)
	require.Len(t, built, 1)
	require.Equal(t, batch.Blocks[0].Hash, built[0].Hash)
}

// attributesFor builds well-formed payload attributes for the block after `parent`, with the given
// L1 origin. It uses the STOCK L1-info builder, so the attributes are the ones a real pipeline would
// produce — which is what makes a negative test about the block rather than about malformed input.
func (se *shimEnv) attributesFor(t *testing.T, parent eth.L2BlockRef, originNum uint64) *eth.PayloadAttributes {
	t.Helper()
	ctx := context.Background()
	origin, err := se.l1.L1BlockRefByNumber(ctx, originNum)
	require.NoError(t, err)
	info, err := se.l1.InfoByHash(ctx, origin.Hash)
	require.NoError(t, err)
	seqNumber := parent.SequenceNumber + 1
	if originNum != parent.L1Origin.Number {
		seqNumber = 0
	}
	ts := parent.Time + se.rollup.BlockTime
	l1Info, err := derive.L1InfoDepositBytes(se.rollup, sepoliaChainConfig(), se.sysCfg, seqNumber, info, ts)
	require.NoError(t, err)
	gasLimit := eth.Uint64Quantity(se.sysCfg.GasLimit)
	withdrawals := gethtypes.Withdrawals{}
	return &eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(ts),
		PrevRandao:            eth.Bytes32(info.MixDigest()),
		SuggestedFeeRecipient: common.HexToAddress("0x4200000000000000000000000000000000000011"),
		Withdrawals:           &withdrawals,
		ParentBeaconBlockRoot: info.ParentBeaconRoot(),
		Transactions:          []eth.Data{l1Info},
		NoTxPool:              true,
		GasLimit:              &gasLimit,
	}
}

// isPipelineIdle reports the "nothing to do yet" errors a pipeline step returns while it waits for
// L1 data or asks to be driven again. Neither is a failure of the harness.
func isPipelineIdle(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, derive.ErrTemporary) || errors.Is(err, derive.ErrReset)
}

// pipelineOverShim is G2's real-stages pipeline with THE STUB REMOVED: `factsL2` is gone and the
// stock op-node L2 client, talking to the shim over JSON-RPC, stands where it stood. Everything else
// is the same stage list — L1Traversal, L1Retrieval, FrameQueue, ChannelMux, ChannelInReader,
// BatchMux, AttributesQueue and the FetchingAttributesBuilder — with the injected data source.
func (se *shimEnv) pipelineOverShim(t *testing.T) *derive.DerivationPipeline {
	t.Helper()
	logger := testlog.Logger(t, 3)
	pipeline := derive.NewDerivationPipeline(
		logger,
		se.rollup,
		staticDepSet{chains: []eth.ChainID{eth.ChainIDFromBig(se.rollup.L2ChainID)}},
		se.l1,
		se.blobs,
		altda.NewAltDA(logger, altda.CLIConfig{}, altda.Config{}, &altda.NoopMetrics{}),
		se.eng,
		metrics.NoopMetrics,
		sepoliaChainConfig(),
		derive.WithDataSource(se.src),
	)
	pipeline.ConfirmEngineReset()
	return pipeline
}

// withRealFeeScalars gives the fixture the fee scalars and batcher address a real chain carries.
//
// It exists so that op-node's SystemConfig reconstruction can be asserted as the EXACT identity. With
// an all-zero scalar the reconstruction is deliberately not the identity: post-Ecotone it re-encodes
// the scalars into the v1 form (`info.L1FeeScalar[0] = 1`, derive/payload_util.go:99-106) because the
// L1-info transaction does not record which encoding the config used. Asserting around that would
// have tested the fixture rather than the shim.
func (e *testEnv) withRealFeeScalars(t *testing.T) *testEnv {
	t.Helper()
	e.sysCfg.Scalar = EcotoneScalar(1368, 810949)
	e.sysCfg.BatcherAddr = common.HexToAddress("0x00000000000000000000000000000000000ba7c4")
	e.rollup.Genesis.SystemConfig = e.sysCfg
	verifier, err := e.cfg.NewVerifier()
	require.NoError(t, err)
	e.src = NewDataSource(testlog.Logger(t, 3), e.cfg, e.rollup, sepoliaChainConfig(),
		e.sysCfg, e.l1, e.blobs, verifier, e.facts)
	return e
}
