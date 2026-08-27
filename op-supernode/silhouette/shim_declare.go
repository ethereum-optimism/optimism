package silhouette

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// SELF-DECLARATION AT THE SERVICE LAYER.
//
// PLAN.md DR-1 made self-declaration a binding mitigation of the shim's declared fake-execution
// surface, and originally put it in the block header as `ExtraData = "silhouette-v1"`. That is dead
// (G2 D3 → RULED in G2 D8): extraData is the consensus-critical carrier of the eip-1559 parameters
// on a Holocene+ chain, strictly length-checked, and op-node reconstructs the SystemConfig by reading
// those parameters back out of the PARENT header on every single block
// (derive.PayloadToSystemConfig, op-node/rollup/derive/payload_util.go:126-131). A marker there would
// not have been a harmless lie in a field nobody prints — it would have silently reset the chain's
// fee market on the following block and divided by zero computing the next base fee. On a Holocene+
// OP header there are no spare bytes at all: nonce and difficulty must be zero, ommersHash is the
// empty-uncle hash, and mixDigest is prevRandao.
//
// So the declaration lives here, and it is strictly MORE visible than thirteen bytes in a header:
// this is the surface an operator, an integrator or an incident responder actually reads. Three
// methods, and the design rule for all of them is that they state what is fabricated rather than
// advertising what is proven.

// SilhouetteAPI is the `silhouette` RPC namespace.
type SilhouetteAPI struct{ s *Shim }

// SelfDeclaration is the answer to "what am I talking to".
type SelfDeclaration struct {
	Client string `json:"client"`
	// ProofRendered is the headline: every block this service describes was rendered from a proof
	// commitment, not executed.
	ProofRendered bool `json:"proofRendered"`
	// ExecutesTransactions is false, and it is stated as its own field rather than implied, because
	// "renders blocks from proofs" and "does not execute" are two claims and an integrator needs both.
	ExecutesTransactions bool `json:"executesTransactions"`
	// L2ChainID identifies the chain.
	L2ChainID hexutil.Uint64 `json:"l2ChainID"`
	// RealFields are the header fields served verbatim from the proof commitment. An output root
	// computed from these is byte-identical to the chain's settlement claims.
	RealFields []string `json:"realFields"`
	// FabricatedFields are the header fields that are deterministic filler. Nothing on the verified
	// path reads them, and no consumer should.
	FabricatedFields []string `json:"fabricatedFields"`
	// HeadersReHash is false: keccak(RLP(a served header)) does not equal the block hash served with
	// it, for a proven block. This is the single most important sentence in this struct.
	HeadersReHash bool `json:"headersReHash"`
	// Unserved lists methods that are refused rather than answered, with the reason.
	Unserved map[string]string `json:"unserved"`
}

// SelfDeclaration declares what this service is.
func (a *SilhouetteAPI) SelfDeclaration() *SelfDeclaration {
	return &SelfDeclaration{
		Client:               ClientVersion,
		ProofRendered:        true,
		ExecutesTransactions: false,
		L2ChainID:            hexutil.Uint64(bigs.Uint64Strict(a.s.params.Rollup.L2ChainID)),
		RealFields: []string{
			"hash", "parentHash", "number", "timestamp", "stateRoot", "withdrawalsRoot",
		},
		FabricatedFields: []string{
			"receiptsRoot", "logsBloom", "gasUsed", "baseFeePerGas", "transactionsRoot",
		},
		HeadersReHash: false,
		Unserved: map[string]string{
			"eth_getBlockReceipts": "this chain's logs are published on the proof wire with explicit " +
				"indices and poison gaps; rendering-device receipts are display-only and never an " +
				"ingestion source (PLAN.md LogsDB rule)",
			"eth_getProof": "this service holds no state and no trie; post-Isthmus the message-passer " +
				"storage root is served in the header's withdrawalsRoot instead",
		},
	}
}

// BlockDeclaration is the per-block provenance answer: which kind of block this is, and
// the settlement-facing values it carries.
type BlockDeclaration struct {
	Number    hexutil.Uint64 `json:"number"`
	Hash      common.Hash    `json:"hash"`
	Timestamp hexutil.Uint64 `json:"timestamp"`
	// Provenance is "proven", "forced", "replacement" or "genesis". "proven" means
	// the hash and roots came off the wire, inside an accepted proof batch. "forced" means the
	// forced-extension convention produced them, computed identically by this node, the prover and the
	// superroot program. "replacement" is a real deposits-only block executed by P's private EL and
	// waiting for its proof batch. "genesis" is configuration, and its roots are NOT KNOWN here.
	Provenance string `json:"provenance"`
	// RootsKnown is false only for genesis, and it is a separate field so that a caller cannot read a
	// zero root as a real one.
	RootsKnown               bool           `json:"rootsKnown"`
	StateRoot                common.Hash    `json:"stateRoot"`
	MessagePasserStorageRoot common.Hash    `json:"messagePasserStorageRoot"`
	OutputRoot               common.Hash    `json:"outputRoot"`
	L1Origin                 eth.BlockID    `json:"l1Origin"`
	SequenceNumber           hexutil.Uint64 `json:"sequenceNumber"`
	// Carrier is the L1 block whose proof batch proved this block. A forced block has none — nothing
	// proved it — and that absence is the honest answer the safety ladder needs.
	Carrier *eth.BlockID `json:"carrier,omitempty"`
}

// BlockProvenance declares how this node knows about a block.
func (a *SilhouetteAPI) BlockProvenance(ctx context.Context, id rpc.BlockNumberOrHash) (*BlockDeclaration, error) {
	var fact Fact
	if hash, ok := id.Hash(); ok {
		f, found := a.s.factByHash(hash)
		if !found {
			return nil, fmt.Errorf("no proven-or-forced fact for block %s", hash)
		}
		fact = f
	} else if number, ok := id.Number(); ok {
		f, found, err := a.s.factByLabel(number)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("no proven-or-forced fact for block %d", number)
		}
		fact = f
	} else {
		return nil, fmt.Errorf("a block number or hash is required")
	}

	out := &BlockDeclaration{
		Number:                   hexutil.Uint64(fact.Number),
		Hash:                     fact.Hash,
		Timestamp:                hexutil.Uint64(fact.Timestamp),
		Provenance:               "proven",
		RootsKnown:               true,
		StateRoot:                fact.StateRoot,
		MessagePasserStorageRoot: fact.MessagePasserStorageRoot,
		OutputRoot:               fact.OutputRoot,
		L1Origin:                 fact.L1Origin,
		SequenceNumber:           hexutil.Uint64(fact.SeqNumber),
	}
	switch {
	case a.s.isGenesis(fact.Number):
		out.Provenance = "genesis"
		out.RootsKnown = false
	case fact.Forced:
		out.Provenance = "forced"
	case fact.Replacement:
		out.Provenance = "replacement"
	}
	if carrier, ok := a.s.facts.CarrierOf(fact.Number); ok {
		out.Carrier = &carrier
	}
	return out, nil
}

// InteropBlock is the receipt-free interop view of one proof-rendered block. It is intentionally
// narrower than Fact: this is the component boundary between the standalone EL and a supernode,
// and only public block identity plus the proof-declared message sets cross it.
type InteropBlock struct {
	Number        hexutil.Uint64              `json:"number"`
	Timestamp     hexutil.Uint64              `json:"timestamp"`
	ParentHash    common.Hash                 `json:"parentHash"`
	Hash          common.Hash                 `json:"hash"`
	Logs          []proofbatch.LogExport      `json:"logs"`
	ExecMsgs      []messages.ExecutingMessage `json:"executingMessages"`
	ExecMsgsKnown bool                        `json:"executingMessagesKnown"`
	Denied        bool                        `json:"denied"`
	// Replacement distinguishes a temporary deposits-only handoff from a corrected proof fact.
	// The former still needs the stock receipts-ingestion equivalent to reseal LogsDB.
	Replacement bool `json:"replacement"`
}

// InteropBlock returns the data an ordinary supernode would learn from receipts. Silhouette blocks
// have no public receipts, so the standalone EL exposes the proof's equivalent facts directly.
func (a *SilhouetteAPI) InteropBlock(ctx context.Context, number hexutil.Uint64) (*InteropBlock, error) {
	fact, ok := a.s.factByNumber(uint64(number))
	if !ok {
		return nil, nil
	}
	denied, err := a.s.facts.Denied(fact.Number, fact.Hash)
	if err != nil {
		return nil, fmt.Errorf("check denial of block %d: %w", fact.Number, err)
	}
	return &InteropBlock{
		Number:        hexutil.Uint64(fact.Number),
		Timestamp:     hexutil.Uint64(fact.Timestamp),
		ParentHash:    fact.ParentHash,
		Hash:          fact.Hash,
		Logs:          cloneLogExports(fact.LogExports),
		ExecMsgs:      append([]messages.ExecutingMessage(nil), fact.ExecMsgs...),
		ExecMsgsKnown: fact.ExecMsgsKnown,
		Denied:        denied,
		Replacement:   fact.Replacement,
	}, nil
}

// MarkDenied mirrors the supernode's durable cross-safety verdict into the EL's durable replay
// state. The supernode denylist remains the decision source; this makes replacement handling
// immediate and restart-safe without a callback during the build request.
func (a *SilhouetteAPI) MarkDenied(number hexutil.Uint64, hash common.Hash) error {
	if err := a.s.facts.MarkDenied(uint64(number), hash); err != nil {
		return err
	}
	return a.s.facts.Flush()
}

// DeniedBlock identifies a local denial-cache entry to forget after the supernode prunes its
// durable denylist during an L1 rewind.
type DeniedBlock struct {
	Number hexutil.Uint64 `json:"number"`
	Hash   common.Hash    `json:"hash"`
}

// PruneDenied keeps the standalone EL's replay cache aligned with the supernode's durable denylist.
func (a *SilhouetteAPI) PruneDenied(blocks []DeniedBlock) error {
	removed := make(map[uint64][]common.Hash)
	for _, block := range blocks {
		n := uint64(block.Number)
		removed[n] = append(removed[n], block.Hash)
	}
	a.s.facts.PruneDenied(removed)
	return a.s.facts.Flush()
}

// Status is the operational picture: where the labels stand, what the fact window covers, and whether
// the fail-stop has fired.
type Status struct {
	Unsafe    eth.L2BlockRef `json:"unsafe"`
	Safe      eth.L2BlockRef `json:"safe"`
	Finalized eth.L2BlockRef `json:"finalized"`
	// CanonicalHead is always present, including when the proof-fact window is empty. During a
	// deposits-only handoff it may be the temporary replacement above HeadFact.
	CanonicalHead eth.BlockID `json:"canonicalHead"`
	// OldestFact and HeadFact bound the window of blocks this node can answer about. Below
	// OldestFact the answer is "not here any more", which is a different statement from "not proven"
	// and must never be conflated with it.
	OldestFact *hexutil.Uint64 `json:"oldestFact,omitempty"`
	HeadFact   *hexutil.Uint64 `json:"headFact,omitempty"`
	// Halted and HaltReason report the fail-stop. A halted shim refuses everything, on purpose: the
	// alternative to stopping is describing a block no proof covers.
	Halted     bool   `json:"halted"`
	HaltReason string `json:"haltReason,omitempty"`
}

// Status reports the shim's operational state.
func (a *SilhouetteAPI) Status() *Status {
	c := a.s.facts.Cursors()
	head := a.s.head()
	out := &Status{
		Unsafe: c.Unsafe, Safe: c.Safe, Finalized: c.Finalized,
		CanonicalHead: eth.BlockID{Hash: head.Hash, Number: head.Number},
	}
	if oldest, ok := a.s.facts.Oldest(); ok {
		v := hexutil.Uint64(oldest.Number)
		out.OldestFact = &v
	}
	if head, ok := a.s.facts.Head(); ok {
		v := hexutil.Uint64(head.Number)
		out.HeadFact = &v
	}
	if reason, halted := a.s.Halted(); halted {
		out.Halted = true
		out.HaltReason = reason.Error()
	}
	return out
}

// Web3API is the `web3` namespace: the one method every tool calls first.
type Web3API struct{}

// ClientVersion names this service as a proof-rendering client.
func (Web3API) ClientVersion() string { return ClientVersion }
