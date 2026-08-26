package silhouette

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// These tests ARE the executable spec of the forced-extension convention (g-decisions.md G2 D2, as
// corrected by G2 D7). The Rust guest and the superroot program must produce the same blocks, so
// every expectation here is written as a closed-form value rather than as whatever the code happens
// to return: a test that asserted "the same thing the implementation did" would agree with a wrong
// implementation.

// provenParent is a plausible last-proven block: real-looking roots off the wire, sitting on the
// genesis L1 origin.
func provenParent() Fact {
	state := crypto.Keccak256Hash([]byte("state"), []byte{9})
	mp := crypto.Keccak256Hash([]byte("mp"), []byte{9})
	hash := crypto.Keccak256Hash([]byte("l2"), []byte{9})
	return Fact{
		Number:                   100,
		Timestamp:                l1GenesisT,
		Hash:                     hash,
		StateRoot:                state,
		MessagePasserStorageRoot: mp,
		OutputRoot: common.Hash(eth.OutputRoot(&eth.OutputV0{
			StateRoot:                eth.Bytes32(state),
			MessagePasserStorageRoot: eth.Bytes32(mp),
			BlockHash:                hash,
		})),
		L1Origin:  eth.BlockID{Hash: l1Hash(l1GenesisNum), Number: l1GenesisNum},
		SeqNumber: 3,
	}
}

func forcedTestParams() ForcedParams {
	cfg := silhouetteRollupConfig()
	return ForcedParams{Rollup: cfg, L1Chain: sepoliaChainConfig(), SysCfg: cfg.Genesis.SystemConfig}
}

// TestForcedExtensionWindowPredicate pins WHEN forced blocks are due and HOW MANY there are.
//
// The window predicate is `epoch.number + seq_window_size <= pipeline_origin`, evaluated against the
// OLDEST buffered epoch, and forced blocks fill the current epoch up to the next epoch's L1
// timestamp before the epoch advances by exactly one. The counts below are the arithmetic of that
// rule with a 60-block window, 12-second L1 and 2-second L2, and they are the numbers the guest has
// to reproduce.
func TestForcedExtensionWindowPredicate(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()

	for _, tc := range []struct {
		name           string
		pipelineOrigin uint64
		wantCount      int
	}{
		{
			// One block below expiry: nothing is due. The window has not expired for epoch 1000,
			// and DR-2's whole point is that this is the ordinary state of the world.
			name: "window not expired", pipelineOrigin: l1GenesisNum + seqWindow - 1, wantCount: 0,
		},
		{
			// Expiry exactly: epoch 1000 is forced. Its blocks fill from parent.time+2 up to but not
			// including epoch 1001's timestamp (+12), so 5 blocks at +2,+4,+6,+8,+10. Then the epoch
			// would advance to 1001, whose own window has NOT expired — so generation stops there.
			name: "expiry exactly fills one epoch", pipelineOrigin: l1GenesisNum + seqWindow, wantCount: 5,
		},
		{
			// One block past expiry: epoch 1001 is now forced too. It contributes the firstOfEpoch
			// block at +12 (which the timestamp bound would have refused — firstOfEpoch overrides it)
			// and then 5 more at +14..+22.
			name: "one past expiry advances an epoch", pipelineOrigin: l1GenesisNum + seqWindow + 1, wantCount: 11,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l1 := newFakeL1(tc.pipelineOrigin + 10)
			got, err := ForcedExtension(context.Background(), p, l1, parent, tc.pipelineOrigin)
			require.NoError(t, err)
			require.Len(t, got, tc.wantCount)
		})
	}
}

// TestForcedExtensionStopsAtTheTerminalEpoch is the regression test for G7R D11, a bug found on the
// RUST side of this convention and checked here because both sides implement the same arithmetic.
//
// The bug: test the force predicate ONCE, before the walk, instead of on every iteration. The walk
// then keeps advancing epochs past the last one whose sequencing window has actually expired, and
// generates forced blocks for epochs that are not due — on the live config, about a window's worth,
// ~21 600 spurious blocks. Every one of them would be a fact in the store for a block that never
// existed, and the two implementations would disagree about P's public history.
//
// This asserts the INVARIANT rather than a block count, because the invariant is what the bug
// violates and a count is what a future config change breaks:
//
//	no forced block may sit on an L1 origin whose window has not expired —
//	for every block: origin.Number + SeqWindowSize <= pipelineOrigin
//
// plus the boundary: the highest origin used is EXACTLY the terminal epoch, so the walk stops there
// rather than short of it. The L1 head is put far above the pipeline origin on purpose — with a
// once-tested predicate the walk would run for hundreds of epochs, so the failure is loud rather than
// off-by-one.
func TestForcedExtensionStopsAtTheTerminalEpoch(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()
	const epochsPastExpiry = 3
	pipelineOrigin := l1GenesisNum + seqWindow + epochsPastExpiry
	terminalEpoch := pipelineOrigin - seqWindow

	// L1 continues far past the pipeline origin: nothing but the predicate stops the walk.
	l1 := newFakeL1(pipelineOrigin + seqWindow + 200)
	got, err := ForcedExtension(context.Background(), p, l1, parent, pipelineOrigin)
	require.NoError(t, err)
	require.NotEmpty(t, got, "the window HAS expired here; a zero-length extension would make this test vacuous")

	highest := uint64(0)
	for i, blk := range got {
		require.True(t, blk.Forced)
		require.LessOrEqual(t, blk.L1Origin.Number+seqWindow, pipelineOrigin,
			"forced block %d (number %d) sits on L1 origin %d, whose window has NOT expired at pipeline "+
				"origin %d: the force predicate was not re-tested when the epoch advanced (G7R D11)",
			i, blk.Number, blk.L1Origin.Number, pipelineOrigin)
		highest = max(highest, blk.L1Origin.Number)
	}
	require.Equal(t, terminalEpoch, highest,
		"the walk must reach the terminal epoch exactly: stopping below it would leave forced blocks "+
			"the stock pipeline does generate uncomputed")

	// And the extension is bounded by the epochs, not by L1: one epoch holds l1BlockTime/l2BlockTime
	// blocks, so a walk that ran to the L1 head would be an order of magnitude longer.
	maxPlausible := int((epochsPastExpiry+1)*(l1BlockTime/l2BlockTime) + 1)
	require.LessOrEqual(t, len(got), maxPlausible,
		"the extension covers %d expired epochs, so it cannot be %d blocks long", epochsPastExpiry+1, len(got))
}

// TestForcedExtensionShape pins the sequence: numbers, timestamps, rendered origins, sequence
// numbers, and the hash chain.
func TestForcedExtensionShape(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()
	l1 := newFakeL1(l1GenesisNum + seqWindow + 20)

	got, err := ForcedExtension(context.Background(), p, l1, parent, l1GenesisNum+seqWindow+1)
	require.NoError(t, err)
	require.Len(t, got, 11)

	type want struct {
		number, timestamp, origin, seq uint64
	}
	// Epoch 1000 fills to just before epoch 1001's time; then firstOfEpoch forces a block AT epoch
	// 1001's time with sequence number 0, and epoch 1001 fills from there.
	expect := []want{
		{101, l1GenesisT + 2, 1000, 4},
		{102, l1GenesisT + 4, 1000, 5},
		{103, l1GenesisT + 6, 1000, 6},
		{104, l1GenesisT + 8, 1000, 7},
		{105, l1GenesisT + 10, 1000, 8},
		{106, l1GenesisT + 12, 1001, 0},
		{107, l1GenesisT + 14, 1001, 1},
		{108, l1GenesisT + 16, 1001, 2},
		{109, l1GenesisT + 18, 1001, 3},
		{110, l1GenesisT + 20, 1001, 4},
		{111, l1GenesisT + 22, 1001, 5},
	}
	prevHash := parent.Hash
	for i, w := range expect {
		f := got[i]
		require.Equal(t, w.number, f.Number, "block %d number", i)
		require.Equal(t, w.timestamp, f.Timestamp, "block %d timestamp", i)
		require.Equal(t, w.origin, f.L1Origin.Number, "block %d origin", i)
		require.Equal(t, l1Hash(w.origin), f.L1Origin.Hash, "block %d origin hash", i)
		require.Equal(t, w.seq, f.SeqNumber, "block %d sequence number", i)
		require.True(t, f.Forced, "block %d must be marked forced", i)
		// The hash chain: forced[i].parentHash == hash(forced[i-1]), forced[0] on the proven head.
		require.Equal(t, prevHash, f.Header.ParentHash, "block %d parent hash", i)
		require.Equal(t, f.Hash, f.Header.Hash(), "block %d hash must be its header's hash", i)
		prevHash = f.Hash
	}
}

// TestForcedBlockIdentitySTF pins the half of the convention that says state does not move: the
// roots carry forward, and the output root is therefore computable from two parent header fields
// plus the forced block's own hash, with no state access anywhere.
func TestForcedBlockIdentitySTF(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()
	l1 := newFakeL1(l1GenesisNum + seqWindow + 20)

	got, err := ForcedExtension(context.Background(), p, l1, parent, l1GenesisNum+seqWindow+1)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	for i, f := range got {
		require.Equal(t, parent.StateRoot, f.StateRoot, "block %d state root must carry forward", i)
		require.Equal(t, parent.MessagePasserStorageRoot, f.MessagePasserStorageRoot,
			"block %d message-passer root must carry forward", i)
		require.Equal(t, parent.StateRoot, f.Header.Root, "block %d header state root", i)
		// Post-Isthmus the header's withdrawalsRoot IS the message-passer storage root, so identity-STF
		// puts the parent's value straight into this field. That identity is what makes the output root
		// derivable from headers alone.
		require.NotNil(t, f.Header.WithdrawalsHash)
		require.Equal(t, parent.MessagePasserStorageRoot, *f.Header.WithdrawalsHash,
			"block %d withdrawals root must be the parent message-passer root", i)
		require.Equal(t, common.Hash(eth.OutputRoot(&eth.OutputV0{
			StateRoot:                eth.Bytes32(parent.StateRoot),
			MessagePasserStorageRoot: eth.Bytes32(parent.MessagePasserStorageRoot),
			BlockHash:                f.Hash,
		})), f.OutputRoot, "block %d output root", i)
	}
}

// TestForcedBlockHeaderFields pins every remaining header field of a forced block. This is the
// field-by-field table of G2 D2.4 as assertions; a guest that disagrees with any line here produces
// a different block hash and the convention breaks.
func TestForcedBlockHeaderFields(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()
	l1 := newFakeL1(l1GenesisNum + seqWindow + 20)

	got, err := ForcedExtension(context.Background(), p, l1, parent, l1GenesisNum+seqWindow)
	require.NoError(t, err)
	require.Len(t, got, 5)
	h := got[0].Header

	require.Equal(t, types.EmptyUncleHash, h.UncleHash)
	require.Equal(t, predeploys.SequencerFeeVaultAddr, h.Coinbase)
	// A forced block carries one transaction that nothing executed: a real transactions root, and
	// zero receipts.
	require.NotEqual(t, types.EmptyTxsHash, h.TxHash, "the L1-info transaction must be in the block")
	require.Equal(t, types.EmptyReceiptsHash, h.ReceiptHash)
	require.Equal(t, types.Bloom{}, h.Bloom)
	require.Equal(t, uint64(0), h.GasUsed)
	require.Equal(t, big.NewInt(0), h.Difficulty)
	require.Equal(t, types.BlockNonce{}, h.Nonce)
	require.Equal(t, uint64(30_000_000), h.GasLimit, "gas limit comes from the frozen SystemConfig")
	require.Equal(t, big.NewInt(101), h.Number)

	// G2 D7: the base fee is PINNED to the frozen SystemConfig's minimum, not computed by the stock
	// formula. The stock formula needs parent.baseFee and parent.gasUsed, and neither is on the wire —
	// so a verifier could not compute it while the guest could, which is exactly the asymmetry that
	// would have shipped a silent divergence.
	require.Equal(t, big.NewInt(0), h.BaseFee, "min base fee is 0 on this config")

	// extraData is the stock Jovian eip-1559 encoding of the frozen parameters — 17 bytes, version
	// byte 0x01, denominator 250, elasticity 6, min base fee 0. NOT the "silhouette-v1" ASCII marker
	// PLAN.md asks for: that is 13 bytes, fails Holocene validation, and would make the next block's
	// base-fee computation divide by zero (G2 D3, escalated).
	require.Len(t, h.Extra, 17)
	require.Equal(t, byte(0x01), h.Extra[0])
	require.Equal(t, []byte{0, 0, 0, 250}, h.Extra[1:5], "denominator 250")
	require.Equal(t, []byte{0, 0, 0, 6}, h.Extra[5:9], "elasticity 6")
	require.Equal(t, make([]byte, 8), h.Extra[9:17], "min base fee 0")

	// prevRandao is the mixHash of the L1 ORIGIN, which during a forced run is older than the
	// pipeline origin that triggered generation.
	require.Equal(t, crypto.Keccak256Hash([]byte("randao"), big.NewInt(int64(l1GenesisNum)).Bytes()),
		h.MixDigest)

	require.NotNil(t, h.BlobGasUsed)
	require.Equal(t, uint64(0), *h.BlobGasUsed)
	require.NotNil(t, h.ExcessBlobGas)
	require.Equal(t, uint64(0), *h.ExcessBlobGas, "OP chains always carry zero excess blob gas")
	require.NotNil(t, h.RequestsHash)
	require.Equal(t, types.EmptyRequestsHash, *h.RequestsHash)
	require.Nil(t, h.BlockAccessListHash, "Amsterdam is deliberately not active (G2 F3)")
}

// TestForcedExtensionDeterminism is the property the whole convention rests on: three independent
// implementations must agree, so the same inputs must give bit-identical output every time.
func TestForcedExtensionDeterminism(t *testing.T) {
	parent := provenParent()
	p := forcedTestParams()

	first, err := ForcedExtension(context.Background(), forcedTestParams(), newFakeL1(l1GenesisNum+seqWindow+20),
		parent, l1GenesisNum+seqWindow+1)
	require.NoError(t, err)
	second, err := ForcedExtension(context.Background(), p, newFakeL1(l1GenesisNum+seqWindow+40),
		parent, l1GenesisNum+seqWindow+1)
	require.NoError(t, err)

	require.Len(t, second, len(first))
	for i := range first {
		require.Equal(t, first[i].Hash, second[i].Hash, "block %d must be reproducible", i)
		require.Equal(t, first[i].OutputRoot, second[i].OutputRoot, "block %d output root", i)
	}
	// A deeper L1 head must not change the blocks already due: the extension is a function of the
	// pipeline origin, not of how much L1 happens to exist beyond it.
	require.Equal(t, l1GenesisNum+seqWindow+20, newFakeL1(l1GenesisNum+seqWindow+20).head)
}

// TestForcedExtensionIsAFunctionOfWireFactsOnly is the G2 D7 property stated as a test: a forced
// block depends on nothing except the parent's WIRE facts, the frozen SystemConfig and the L1
// headers. Changing a parent field the wire does not carry must not change anything.
func TestForcedExtensionIsAFunctionOfWireFactsOnly(t *testing.T) {
	p := forcedTestParams()
	base, err := ForcedExtension(context.Background(), p, newFakeL1(l1GenesisNum+seqWindow+20),
		provenParent(), l1GenesisNum+seqWindow)
	require.NoError(t, err)
	require.NotEmpty(t, base)

	// SeqNumber IS a rendered fact and legitimately shifts the L1-info transaction, so vary something
	// that is neither on the wire nor in config: a stale header on the parent Fact.
	parent := provenParent()
	parent.Header = &types.Header{GasUsed: 12345, BaseFee: big.NewInt(999)}
	got, err := ForcedExtension(context.Background(), p, newFakeL1(l1GenesisNum+seqWindow+20),
		parent, l1GenesisNum+seqWindow)
	require.NoError(t, err)
	require.Len(t, got, len(base))
	for i := range base {
		require.Equal(t, base[i].Hash, got[i].Hash,
			"block %d must not depend on anything absent from the wire", i)
	}
}

// TestForcedBlockExtraDataRoundTripsThroughOpNode is the assertion the extraData ruling asks for
// (G2 D8): not "we wrote the bytes we meant to write", but "a forced block is a legal parent for
// stock derivation".
//
// op-node reconstructs the SystemConfig from the PARENT header's extraData on every single block,
// through PayloadToSystemConfig. That is the path a marker in extraData would have corrupted — it
// would have silently reset the chain's eip-1559 parameters to the Canyon defaults on the following
// block — so it is the path a forced block has to survive. Feeding a forced block to op-node's own
// parser and requiring the frozen SystemConfig back is a much stronger statement than comparing 17
// bytes to a literal, because it also covers the gas limit, the scalars and the min base fee.
func TestForcedBlockExtraDataRoundTripsThroughOpNode(t *testing.T) {
	p := forcedTestParams()
	l1 := newFakeL1(l1GenesisNum + seqWindow + 20)
	got, err := ForcedExtension(context.Background(), p, l1, provenParent(), l1GenesisNum+seqWindow+1)
	require.NoError(t, err)
	require.NotEmpty(t, got)

	// The frozen genesis SystemConfig, with the fee scalars a real chain carries: this is what stock
	// derivation must read back out of every forced block.
	frozen := p.SysCfg
	frozen.Scalar = EcotoneScalar(1368, 810949)
	frozen.BatcherAddr = common.HexToAddress("0x00000000000000000000000000000000000ba7c4")

	for i, f := range got {
		// Rebuild the L1-info transaction the forced block carries, then present the block to op-node
		// exactly as a payload from an execution client would arrive.
		info, err := l1.InfoByHash(context.Background(), f.L1Origin.Hash)
		require.NoError(t, err)
		l1InfoTx, err := derive.L1InfoDepositBytes(p.Rollup, p.L1Chain, frozen, f.SeqNumber, info, f.Timestamp)
		require.NoError(t, err)

		payload := &eth.ExecutionPayload{
			ParentHash:   f.Header.ParentHash,
			BlockNumber:  hexutil.Uint64(f.Number),
			BlockHash:    f.Hash,
			Timestamp:    hexutil.Uint64(f.Timestamp),
			GasLimit:     hexutil.Uint64(f.Header.GasLimit),
			ExtraData:    f.Header.Extra,
			Transactions: []hexutil.Bytes{l1InfoTx},
		}

		back, err := derive.PayloadToSystemConfig(p.Rollup, payload)
		require.NoError(t, err, "forced block %d must be a legal parent for stock derivation", i)
		require.Equal(t, frozen.EIP1559Params, back.EIP1559Params,
			"forced block %d: op-node must read the frozen eip-1559 params back out of extraData", i)
		require.Equal(t, frozen.GasLimit, back.GasLimit, "forced block %d gas limit", i)
		require.Equal(t, frozen.MinBaseFee, back.MinBaseFee, "forced block %d min base fee", i)
		require.Equal(t, frozen.Scalar, back.Scalar, "forced block %d fee scalar", i)
		require.Equal(t, frozen.BatcherAddr, back.BatcherAddr, "forced block %d batcher address", i)
	}
}

// TestForcedBlockRejectsAMarkerInExtraData is the ruling stated negatively: if anyone ever puts an
// ASCII marker back into a forced block's extraData, op-node's own validator refuses the block. This
// is here so the reason the marker is dead cannot be quietly rediscovered as a preference.
func TestForcedBlockRejectsAMarkerInExtraData(t *testing.T) {
	p := forcedTestParams()
	l1 := newFakeL1(l1GenesisNum + seqWindow + 20)
	got, err := ForcedExtension(context.Background(), p, l1, provenParent(), l1GenesisNum+seqWindow)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	f := got[0]

	frozen := p.SysCfg
	frozen.Scalar = EcotoneScalar(1368, 810949)
	info, err := l1.InfoByHash(context.Background(), f.L1Origin.Hash)
	require.NoError(t, err)
	l1InfoTx, err := derive.L1InfoDepositBytes(p.Rollup, p.L1Chain, frozen, f.SeqNumber, info, f.Timestamp)
	require.NoError(t, err)

	_, err = derive.PayloadToSystemConfig(p.Rollup, &eth.ExecutionPayload{
		ParentHash:   f.Header.ParentHash,
		BlockNumber:  hexutil.Uint64(f.Number),
		BlockHash:    f.Hash,
		Timestamp:    hexutil.Uint64(f.Timestamp),
		GasLimit:     hexutil.Uint64(f.Header.GasLimit),
		ExtraData:    []byte("silhouette-v1"),
		Transactions: []hexutil.Bytes{l1InfoTx},
	})
	require.Error(t, err, "an ASCII marker in extraData must be refused by op-node's own validator")
}

// TestForcedBlockExtraDataFallsBackToChainDefaults covers the shape a LIVE silhouette-style chain
// actually has: Cove's chain P carries eip1559Params = 0x0000000000000000 in its genesis
// SystemConfig, which does NOT mean "no parameters" — it means "use the chain config's defaults",
// and the execution layer substitutes them when it builds the header.
//
// A forced block has to make the same substitution or its extraData disagrees with every other
// block on the chain, and the disagreement would surface as a wrong block hash rather than as an
// error. Found by configuring this suite against the real Cove P config.
func TestForcedBlockExtraDataFallsBackToChainDefaults(t *testing.T) {
	p := forcedTestParams()
	p.SysCfg.EIP1559Params = eth.Bytes8{} // all zero, exactly as live chain P has it

	got, err := ForcedExtension(context.Background(), p, newFakeL1(l1GenesisNum+seqWindow+20),
		provenParent(), l1GenesisNum+seqWindow)
	require.NoError(t, err, "all-zero eip1559 params mean chain defaults, not a broken config")
	require.NotEmpty(t, got)

	// The chain config's Canyon denominator (250) and elasticity (6) — the same values G1 measured
	// on live chain P — must appear in the header even though the SystemConfig field is zeroes.
	h := got[0].Header
	require.Len(t, h.Extra, 17)
	require.Equal(t, []byte{0, 0, 0, 250}, h.Extra[1:5], "Canyon denominator from the chain config")
	require.Equal(t, []byte{0, 0, 0, 6}, h.Extra[5:9], "elasticity from the chain config")

	// And a MIXED pair is still refused: Holocene requires both zero or both non-zero, so one of
	// each is a corrupt config rather than a request for defaults.
	p.SysCfg.EIP1559Params = eth.Bytes8{0, 0, 0, 0, 0, 0, 0, 6}
	_, err = ForcedExtension(context.Background(), p, newFakeL1(l1GenesisNum+seqWindow+20),
		provenParent(), l1GenesisNum+seqWindow)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both zero")
}
