package silhouette

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// The eth_ query surface.
//
// The contract is not "implement the JSON-RPC spec". It is "answer what op-service/sources sends",
// because that package is the only client that matters on the verified path and it is the compliance
// oracle for this file: every field below exists because sources.RPCHeader / sources.RPCBlock reads
// it, and the shapes are the shapes those types unmarshal. Two calls carry almost all of it —
// `eth_getBlockByNumber(id, false)` for headers and refs, `eth_getBlockByNumber(id, true)` for full
// blocks — and between them they serve L2BlockRefBy*, SystemConfigByL2Hash, PayloadBy*, InfoBy*, the
// FindL2Heads walk, and OutputV0AtBlock*.

// EthAPI is the `eth` RPC namespace.
type EthAPI struct{ s *Shim }

// rpcBlock is the eth_getBlockBy* response, field-for-field what sources.RPCBlock decodes.
//
// `hash` is the load-bearing field and the honest one: for a proven block it is the hash the proof
// committed to, and it is NOT keccak(RLP(the header above it)). The header's interior — receipts
// root, bloom, gas used, and the transaction list itself — is a rendering of a block whose interior
// is private. Nothing on the verifier's path re-hashes a header (trustCache), so nothing notices; a
// client configured to distrust this RPC would notice immediately, which is the correct behaviour for
// a client that has been told to verify what it is given.
type rpcBlock struct {
	ParentHash  common.Hash      `json:"parentHash"`
	UncleHash   common.Hash      `json:"sha3Uncles"`
	Coinbase    common.Address   `json:"miner"`
	Root        common.Hash      `json:"stateRoot"`
	TxHash      common.Hash      `json:"transactionsRoot"`
	ReceiptHash common.Hash      `json:"receiptsRoot"`
	Bloom       eth.Bytes256     `json:"logsBloom"`
	Difficulty  *hexutil.Big     `json:"difficulty"`
	Number      hexutil.Uint64   `json:"number"`
	GasLimit    hexutil.Uint64   `json:"gasLimit"`
	GasUsed     hexutil.Uint64   `json:"gasUsed"`
	Time        hexutil.Uint64   `json:"timestamp"`
	Extra       hexutil.Bytes    `json:"extraData"`
	MixDigest   common.Hash      `json:"mixHash"`
	Nonce       types.BlockNonce `json:"nonce"`

	BaseFee *hexutil.Big `json:"baseFeePerGas"`

	WithdrawalsRoot  *common.Hash    `json:"withdrawalsRoot,omitempty"`
	BlobGasUsed      *hexutil.Uint64 `json:"blobGasUsed,omitempty"`
	ExcessBlobGas    *hexutil.Uint64 `json:"excessBlobGas,omitempty"`
	ParentBeaconRoot *common.Hash    `json:"parentBeaconBlockRoot,omitempty"`
	RequestsHash     *common.Hash    `json:"requestsHash,omitempty"`

	Hash common.Hash `json:"hash"`

	// Transactions is either the hash list (fullTx=false) or the transaction objects
	// (fullTx=true). sources.RawTransaction marshals the object form, so the round trip through the
	// oracle's own codec is exact rather than approximate.
	Transactions any                `json:"transactions"`
	Withdrawals  *types.Withdrawals `json:"withdrawals,omitempty"`
	Uncles       []common.Hash      `json:"uncles"`
}

func renderRPCBlock(r Rendering, fullTx bool) *rpcBlock {
	hdr := r.Header
	out := &rpcBlock{
		ParentHash:       hdr.ParentHash,
		UncleHash:        hdr.UncleHash,
		Coinbase:         hdr.Coinbase,
		Root:             hdr.Root,
		TxHash:           hdr.TxHash,
		ReceiptHash:      hdr.ReceiptHash,
		Bloom:            eth.Bytes256(hdr.Bloom),
		Difficulty:       (*hexutil.Big)(hdr.Difficulty),
		Number:           hexutil.Uint64(hdr.Number.Uint64()),
		GasLimit:         hexutil.Uint64(hdr.GasLimit),
		GasUsed:          hexutil.Uint64(hdr.GasUsed),
		Time:             hexutil.Uint64(hdr.Time),
		Extra:            hdr.Extra,
		MixDigest:        hdr.MixDigest,
		Nonce:            hdr.Nonce,
		BaseFee:          (*hexutil.Big)(hdr.BaseFee),
		WithdrawalsRoot:  hdr.WithdrawalsHash,
		ParentBeaconRoot: hdr.ParentBeaconRoot,
		RequestsHash:     hdr.RequestsHash,
		Hash:             r.Hash,
		Uncles:           []common.Hash{},
	}
	if hdr.BlobGasUsed != nil {
		v := hexutil.Uint64(*hdr.BlobGasUsed)
		out.BlobGasUsed = &v
	}
	if hdr.ExcessBlobGas != nil {
		v := hexutil.Uint64(*hdr.ExcessBlobGas)
		out.ExcessBlobGas = &v
	}
	if hdr.WithdrawalsHash != nil {
		// Canyon+ L2 bodies carry a present-but-empty withdrawals list, and
		// sources.RPCBlock.validateL2Withdrawals requires exactly that when a withdrawals root is set.
		out.Withdrawals = &types.Withdrawals{}
	}
	txs := make(sources.RawTransactions, len(r.Txs))
	for i, tx := range r.Txs {
		txs[i] = tx
	}
	if fullTx {
		out.Transactions = txs
	} else {
		out.Transactions = txs.Hashes()
	}
	return out
}

// ChainId is the L2 chain ID, from the rollup config.
func (a *EthAPI) ChainId() hexutil.Uint64 {
	return hexutil.Uint64(a.s.params.Rollup.L2ChainID.Uint64())
}

// Syncing is STATICALLY false.
//
// `--syncmode=consensus-layer` and DR-1: the shim never syncs, so it never returns SYNCING from
// forkchoice either, and the CL's EL-sync state machine stays idle. A shim that reported syncing
// would be claiming it was catching up to a chain it could catch up to, and there is no such chain:
// its whole history is the proof stream, which arrives through the CL.
func (a *EthAPI) Syncing() bool { return false }

// GetBlockByNumber serves a block by number or label.
//
// An absent height is a JSON null with no error, which is what a stock execution client answers and
// what op-service/sources converts into ethereum.NotFound (eth_client.go:225-227, 249-251). That
// conversion is load-bearing far beyond this method: the interop round treats EXACTLY
// ethereum.NotFound as "this chain is behind, retry the round" and every other error as a failed
// round. A silhouette chain is behind BY CONSTRUCTION — its head moves once per proof cadence, not
// once per block — so an error here would take the whole dependency set's cross-safety down for as
// long as the chain lags.
func (a *EthAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (*rpcBlock, error) {
	fact, ok, err := a.s.factByLabel(number)
	if err != nil || !ok {
		return nil, err
	}
	return a.serve(ctx, fact, fullTx)
}

// GetBlockByHash serves a block by hash.
func (a *EthAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (*rpcBlock, error) {
	fact, ok := a.s.factByHash(hash)
	if !ok {
		// A nil result with no error is how this RPC says "not found", and the oracle turns it into
		// ethereum.NotFound (eth_client.go:225-227). It is the honest answer for a block below the
		// fact window as well as for one that never existed: proven history is re-derived from L1, not
		// remembered, and "not here any more" must never be reported as "not proven".
		return nil, nil
	}
	return a.serve(ctx, fact, fullTx)
}

func (a *EthAPI) serve(ctx context.Context, fact Fact, fullTx bool) (*rpcBlock, error) {
	r, err := a.s.rendering(ctx, fact)
	if err != nil {
		return nil, err
	}
	return renderRPCBlock(r, fullTx), nil
}

// GetBlockReceipts is REFUSED, always.
//
// PLAN.md's LogsDB rule, binding: LogsDB is fed from the wire — explicit indices and poison gaps —
// and rendering-device receipts are display-only, NEVER an ingestion source. Serving a plausible
// receipt list here is the one fabrication with a security consequence rather than a cosmetic one:
// positional ingestion would need filler logs with public preimages, i.e. forgeable initiating
// messages (the interop checksum carries no block-hash bits), and real-hash gaps would let a P
// insider reference an unexported log and bypass the export policy. Poison gaps are load-bearing for
// the export policy itself.
//
// A clean error is chosen over display-only fabrication because an error cannot be ingested by
// mistake. The honest exit exists and is a config decision, not a code one: under an `all-preimages`
// export policy the wire carries the log preimages and this method can serve receipts BUILT FROM THE
// WIRE.
func (a *EthAPI) GetBlockReceipts(ctx context.Context, id rpc.BlockNumberOrHash) (any, error) {
	return nil, errors.New("eth_getBlockReceipts is not served by a silhouette execution client: this " +
		"chain's logs are published on the proof wire with explicit indices and poison gaps, and " +
		"rendering-device receipts are display-only and never an ingestion source (PLAN.md LogsDB " +
		"rule). Read the exported logs from the proof stream, not from here")
}

// GetProof is REFUSED with the Isthmus citation.
//
// Nothing on the verified path needs it: post-Isthmus `outputV0` reads the message-passer storage
// root straight out of the header's withdrawalsRoot (op-service/sources/l2_client.go:192-227), which
// is the structural gift this whole design rests on. The pre-Isthmus branch of that function is the
// one that would call eth_getProof, and a silhouette chain that reached it would be asking a node
// with no state for a Merkle proof — better an explicit refusal than a fabricated proof that fails
// `proof.Verify(stateRoot)` in a confusing place.
func (a *EthAPI) GetProof(ctx context.Context, address common.Address, storage []common.Hash, blockTag string) (any, error) {
	return nil, fmt.Errorf("eth_getProof is not served by a silhouette execution client: it holds no " +
		"state and no trie. Nothing on the verified path needs it — post-Isthmus the L2ToL1MessagePasser " +
		"storage root is served in the header's withdrawalsRoot, which is what stock outputV0 reads " +
		"(op-service/sources/l2_client.go:192-227). A caller reaching this method is on the pre-Isthmus " +
		"state-proof path, which a proof-rendered chain does not have")
}

// factByLabel resolves a block number or label. The three labels are the three cursors, which is the
// whole of the safety ladder as this service sees it.
//
// It reports absence (ok=false) separately from failure (err), because they are different statements:
// a height this node has no fact for is an ordinary "not yet" that a caller retries, while a label
// pointing outside the fact window is a node that has lost track of its own chain.
func (s *Shim) factByLabel(number rpc.BlockNumber) (Fact, bool, error) {
	c := s.facts.Cursors()
	switch number {
	case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
		// There is no pending block: the shim has no transaction pool and builds nothing
		// speculatively, so "pending" is the head. Saying so beats an error for tooling that asks.
		return s.head(), true, nil
	case rpc.SafeBlockNumber:
		return s.factAtCursor(c.Safe)
	case rpc.FinalizedBlockNumber:
		return s.factAtCursor(c.Finalized)
	case rpc.EarliestBlockNumber:
		return s.genesisFact(), true, nil
	}
	if number < 0 {
		return Fact{}, false, fmt.Errorf("unsupported block number %s", number)
	}
	fact, ok := s.factByNumber(uint64(number))
	return fact, ok, nil
}

func (s *Shim) factAtCursor(ref eth.L2BlockRef) (Fact, bool, error) {
	if ref == (eth.L2BlockRef{}) {
		// No label has been set yet. Genesis is the honest answer: nothing above it is safe as far as
		// this engine has been told.
		return s.genesisFact(), true, nil
	}
	fact, ok := s.factByHash(ref.Hash)
	if !ok {
		return Fact{}, false, fmt.Errorf("the label points at block %s, which is no longer in this "+
			"node's fact window", ref)
	}
	return fact, true, nil
}

func uint256FromBig(v *big.Int) *uint256.Int {
	out := new(uint256.Int)
	if v != nil {
		out.SetFromBig(v)
	}
	return out
}
