package silhouette

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// The PUBLIC RENDERING of a silhouette block: the header and body this node serves for a block it
// never executed.
//
// There is exactly one header assembler in this stack, and this is it. Forced blocks (forced.go) and
// proven blocks (the shim EL) differ in only two inputs — where the state root and message-passer
// root come from, and which transactions the block carries — and in one output: a forced block's
// hash IS keccak(RLP(header)) because the convention defines the block, while a proven block's hash
// comes off the wire and the rendered header deliberately does not re-hash to it.
//
// Sharing the assembler is not tidiness. G2 D2.4's field list is normative across three
// implementations, and a second copy of it in the shim would be a second chance to diverge from the
// prover on a field nobody looks at until a hash disagrees.
//
// What is REAL in a rendered header (fabrication class 1, served verbatim): parentHash, number,
// timestamp, stateRoot, withdrawalsRoot (= the message-passer storage root), and — for proven
// blocks — the block hash the proof committed to. What is DETERMINISTIC RESIDUE (class 2):
// receiptsRoot, logsBloom, gasUsed, and baseFeePerGas. What is CONFIG-DETERMINED and consensus-legal
// (class 1 by the 2026-08-24 amendment, G2 D8): extraData and gasLimit. What is STOCK BYTES
// (class 3): the transaction list, whose first element is the origin-accurate L1-info deposit.

// HeaderInputs is everything a rendered header needs that is not configuration.
type HeaderInputs struct {
	// Parent is the parent block's facts. Only its hash is read; the caller decides which roots
	// carry forward, because that is exactly where forced and proven blocks differ.
	Parent Fact
	// Number and Timestamp are the block's own.
	Number    uint64
	Timestamp uint64
	// StateRoot and MessagePasserStorageRoot are the settlement-facing roots: off the wire for a
	// proven block, carried from the parent for a forced one (identity-STF).
	StateRoot                common.Hash
	MessagePasserStorageRoot common.Hash
	// Origin is the block's RENDERED L1 origin header (G2 D4). Two header fields read it —
	// mixDigest (prevRandao) and parentBeaconBlockRoot — and so does the L1-info transaction.
	Origin eth.BlockInfo
	// Txs is the block's transaction list in canonical binary form, txs[0] the L1-info deposit.
	// These are the bytes the CL supplied at build time, echoed back verbatim, which is what makes
	// SystemConfigByL2Hash and FindL2Heads work through the stock parsers.
	Txs [][]byte
}

// RenderHeader assembles a silhouette block's served header. The field list and its citations are
// g-decisions.md G2 D2.4, corrected by G2 D7 for the base fee.
func RenderHeader(p ForcedParams, in HeaderInputs) (*types.Header, error) {
	if in.Origin == nil {
		return nil, fmt.Errorf("render block %d: no rendered L1 origin header", in.Number)
	}
	if len(in.Txs) == 0 {
		return nil, fmt.Errorf("render block %d: an OP block always carries at least the L1-info "+
			"deposit; an empty transaction list would make the block unparseable by the stock CL", in.Number)
	}
	hdr, err := renderShell(p, in.Timestamp)
	if err != nil {
		return nil, err
	}

	hdr.ParentHash = in.Parent.Hash
	hdr.Root = in.StateRoot
	hdr.TxHash = types.DeriveSha(rawTxs(in.Txs), trie.NewStackTrie(nil))
	hdr.Number = new(big.Int).SetUint64(in.Number)
	hdr.MixDigest = in.Origin.MixDigest() // the L1 ORIGIN's mixHash
	if p.Rollup.IsCanyon(in.Timestamp) {
		// Isthmus+: withdrawalsRoot IS the L2ToL1MessagePasser storage root, which is why stock
		// outputV0 needs no eth_getProof and why a shim can serve P's real output roots.
		mpRoot := in.MessagePasserStorageRoot
		hdr.WithdrawalsHash = &mpRoot
	}
	if p.Rollup.IsEcotone(in.Timestamp) {
		beaconRoot := in.Origin.ParentBeaconRoot()
		if beaconRoot == nil {
			beaconRoot = new(common.Hash)
		}
		hdr.ParentBeaconRoot = beaconRoot
	}
	return hdr, nil
}

// RenderGenesisHeader is the served header of the rollup genesis block.
//
// Genesis is the one block in a silhouette node's universe that is neither proven nor forced: it is
// configuration, and it carries no transactions because it was never derived. Its roots are NOT
// KNOWN to a verifier — there is no wire record of them and no state to read them from — so they are
// served as zero, which fails closed rather than plausibly. See G3 D6.
func RenderGenesisHeader(p ForcedParams) (*types.Header, error) {
	hdr, err := renderShell(p, p.Rollup.Genesis.L2Time)
	if err != nil {
		return nil, err
	}
	hdr.TxHash = types.EmptyTxsHash
	hdr.Number = new(big.Int).SetUint64(p.Rollup.Genesis.L2.Number)
	if p.Rollup.IsCanyon(p.Rollup.Genesis.L2Time) {
		empty := common.Hash{}
		hdr.WithdrawalsHash = &empty
	}
	if p.Rollup.IsEcotone(p.Rollup.Genesis.L2Time) {
		hdr.ParentBeaconRoot = new(common.Hash)
	}
	return hdr, nil
}

// renderShell fills every header field that is a constant, a configuration value or pure residue —
// everything that does not depend on which block this is. Both renderers share it so that the
// residue conventions and the eip-1559 encoding have exactly one definition.
func renderShell(p ForcedParams, timestamp uint64) (*types.Header, error) {
	extra, err := forcedExtra(p, timestamp)
	if err != nil {
		return nil, err
	}
	// G2 D7: the base fee is pinned to the frozen config's minimum rather than computed. Its inputs
	// (parent.baseFee, parent.gasUsed) are NOT on the wire, so a verifier cannot compute what the
	// real chain had — and pinning it is what keeps a forced block computable by both sides. The same
	// reasoning applies to a proven block's rendered header: the real value is private, the rendered
	// one is residue, and nothing on the verifier's path reads it. See G3 D3.
	baseFee := new(big.Int)
	if p.Rollup.IsJovian(timestamp) {
		baseFee.SetUint64(p.SysCfg.MinBaseFee)
	}
	hdr := &types.Header{
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    predeploys.SequencerFeeVaultAddr,
		ReceiptHash: types.EmptyReceiptsHash, // nothing executed: zero receipts
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		GasLimit:    p.SysCfg.GasLimit, // frozen SystemConfig (DR-2)
		GasUsed:     0,                 // nothing executed
		Time:        timestamp,
		Extra:       extra,
		Nonce:       types.BlockNonce{},
		BaseFee:     baseFee,
	}
	if p.Rollup.IsEcotone(timestamp) {
		zero := uint64(0)
		hdr.BlobGasUsed = &zero
		hdr.ExcessBlobGas = &zero // CalcExcessBlobGas short-circuits to zero on every OP chain
	}
	if p.Rollup.IsIsthmus(timestamp) {
		reqs := types.EmptyRequestsHash // Isthmus forces an empty request list
		hdr.RequestsHash = &reqs
	}
	return hdr, nil
}

// rawTxs adapts opaque transaction bytes to the trie's DerivableList. It uses the op-service/sources
// type deliberately: that is the same list implementation the stock client computes a transactions
// root with when it checks a block, so the root this header carries is the root that client would
// compute over the body this node serves.
func rawTxs(txs [][]byte) sources.RawTransactions {
	out := make(sources.RawTransactions, len(txs))
	for i, tx := range txs {
		out[i] = tx
	}
	return out
}

// RenderedBody is the deterministic transaction list of a silhouette block.
//
// A silhouette block's body is not a wire fact and not a stored one: it is a function of the frozen
// SystemConfig and the rendered L1 origin. The transcoder emits EMPTY singular batches (G2 D6) and
// P takes no user deposits (DR-2), so the stock attributes builder produces exactly one transaction
// — the L1-info deposit — and that is the whole body. Recomputing it here rather than requiring it
// to be stored is what lets the shim answer eth_getBlockByNumber for a block it has not been asked
// to build in this process lifetime.
//
// It REFUSES on a fork-activation block, where the stock builder also prepends that fork's upgrade
// transactions (G1 D7's pinned bundle). Chain P activates every fork at genesis so no such block
// exists today, but a future activation would make this reconstruction silently short — and a
// silently short body is a wrong transactionsRoot in a header we serve. The stored rendering from
// the build job is authoritative when it exists; this is only the fallback.
func RenderedBody(ctx context.Context, p ForcedParams, l1 L1Headers, fact Fact) ([][]byte, eth.BlockInfo, error) {
	// forks.From(forks.Regolith), not forks.All: Bedrock is not a scheduleable fork and
	// rollup.Config.IsForkActive PANICS on it ("unknown fork: bedrock"). op-node's own fork loops use
	// the same slice (rollup.scheduleableForks, types.go:722).
	for _, fork := range forks.From(forks.Regolith) {
		if p.Rollup.IsActivationBlockForFork(fact.Timestamp, fork) {
			return nil, nil, fmt.Errorf("cannot reconstruct the body of block %d: the %s fork activates "+
				"at timestamp %d, so the block legally carries that fork's upgrade transactions (G1 D7) "+
				"and only the transaction list the CL supplied at build time is authoritative",
				fact.Number, fork, fact.Timestamp)
		}
	}
	origin, err := l1.InfoByHash(ctx, fact.L1Origin.Hash)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch rendered L1 origin %s of block %d: %w", fact.L1Origin, fact.Number, err)
	}
	l1Info, err := derive.L1InfoDepositBytes(p.Rollup, p.L1Chain, p.SysCfg, fact.SeqNumber, origin, fact.Timestamp)
	if err != nil {
		return nil, nil, fmt.Errorf("build L1-info tx for block %d: %w", fact.Number, err)
	}
	return [][]byte{l1Info}, origin, nil
}
