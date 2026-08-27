package silhouette

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The Engine API. Three methods carry the whole build dance
// (op-node/rollup/engine/build_sealed.go:29-39):
//
//	fcU(attrs) → payloadID → getPayload → newPayload → fcU
//
// Versions come from op-node/rollup/types.go:745-790 against P's fork schedule: Ecotone active means
// forkchoiceUpdatedV3, Isthmus means newPayloadV4, Isthmus means getPayloadV4 and KARST means
// getPayloadV5 — and the generated silhouette rollup config activates Karst at genesis, so V5 is
// what the CL will actually call. Both are registered, dispatching to one implementation: the
// difference between them is fields of the envelope op-node does not read (it decodes into
// eth.ExecutionPayloadEnvelope, which carries the payload and the parent beacon root and nothing
// else).

// EngineAPI is the `engine` RPC namespace.
type EngineAPI struct{ s *Shim }

// ForkchoiceUpdatedV3 moves the head/safe/finalized cursors, and opens a build job when attributes
// are present.
//
// THIS IS THE LADDER. The shim never computes a safety label: the stock Finalizer and the
// cross-safety judge drive safe and finalized down through ordinary forkchoice calls, and all the
// engine does is record where they put them. That is why a proof-carried chain needs no bespoke
// safety plumbing on the execution side.
//
// An unknown head is answered with an InvalidForkchoiceState RPC error rather than a status, because
// that is the one answer that makes the CL do the right thing: the engine controller maps that code
// to a RESET (engine_controller.go:686-693), which re-runs FindL2Heads over the shim's served
// headers and re-discovers where the chain actually is. Answering SYNCING — the stock EL's answer —
// is forbidden here (`--syncmode=consensus-layer`, DR-1), and answering INVALID would be a claim
// about a block rather than about this node's knowledge of it.
func (e *EngineAPI) ForkchoiceUpdatedV3(ctx context.Context, state eth.ForkchoiceState, attrs *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	if err := e.s.checkLive(); err != nil {
		return nil, err
	}
	s := e.s

	head, ok := s.factByHash(state.HeadBlockHash)
	if !ok {
		s.log.Warn("forkchoice named a head this node has no facts for; asking the CL to reset",
			"head", state.HeadBlockHash)
		return nil, eth.InputError{
			Code: eth.InvalidForkchoiceState,
			Inner: fmt.Errorf("head %s is not a proven-or-forced block of this chain, or has fallen out "+
				"of the fact window", state.HeadBlockHash),
		}
	}
	cursors := Cursors{}
	headRef, err := s.ref(head)
	if err != nil {
		return nil, eth.InputError{Code: eth.InvalidForkchoiceState, Inner: err}
	}
	cursors.Unsafe = headRef

	// safe and finalized are optional (zero means "no opinion") and must never lead the head: a
	// label below a block the engine does not have as canonical is the inconsistency the
	// InvalidForkchoiceState code exists for.
	for _, label := range []struct {
		name string
		hash common.Hash
		out  *eth.L2BlockRef
	}{
		{"safe", state.SafeBlockHash, &cursors.Safe},
		{"finalized", state.FinalizedBlockHash, &cursors.Finalized},
	} {
		if label.hash == (common.Hash{}) {
			continue
		}
		fact, ok := s.factByHash(label.hash)
		if !ok {
			return nil, eth.InputError{
				Code:  eth.InvalidForkchoiceState,
				Inner: fmt.Errorf("%s block %s is not a proven-or-forced block of this chain", label.name, label.hash),
			}
		}
		if fact.Number > head.Number {
			return nil, eth.InputError{
				Code:  eth.InvalidForkchoiceState,
				Inner: fmt.Errorf("%s block %d is above the head %d", label.name, fact.Number, head.Number),
			}
		}
		ref, err := s.ref(fact)
		if err != nil {
			return nil, eth.InputError{Code: eth.InvalidForkchoiceState, Inner: err}
		}
		*label.out = ref
	}

	// A forkchoice update to an older head is a reorg, and it is stock: an L1 reorg resets the
	// pipeline, the transcoder's chaining state rewinds with it (G2 D5), FindL2Heads walks the shim's
	// headers through the trusted client, and the CL points the cursors lower. The engine's part is
	// to forget the renderings above the new head and cancel the build jobs, so a re-derivation
	// re-renders from the facts rather than answering out of a stale table.
	previous := s.facts.Cursors()
	if previous.Unsafe.Number > head.Number || (previous.Unsafe.Number == head.Number && previous.Unsafe.Hash != head.Hash) {
		s.log.Info("forkchoice rewound the head; forgetting renderings above it",
			"from", previous.Unsafe, "to", headRef)
		s.facts.DropRenderingsAbove(head.Number)
		s.dropJobs()
	}
	s.facts.SetCursors(cursors)
	if !s.isRewindFact(head.Hash) {
		s.clearRewindFacts()
	}

	result := &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &head.Hash},
	}
	if attrs == nil {
		return result, nil
	}

	// Attributes present: open a build job. The job is a promise to look one fact up, not a partial
	// block — there is nothing to build. Validation of what it will produce happens in getPayload,
	// where the fail-stop belongs: the same attributes may legitimately be re-attempted with a new
	// job after a transient fact-store lag, and the CL's own error mapping
	// (build_seal.go, ErrSealExpired) is built for exactly that.
	if want := head.Timestamp + s.params.Rollup.BlockTime; uint64(attrs.Timestamp) != want {
		return nil, eth.InputError{
			Code: eth.InvalidPayloadAttributes,
			Inner: fmt.Errorf("attributes at timestamp %d do not extend head %d (timestamp %d) by one "+
				"block time: expected %d", uint64(attrs.Timestamp), head.Number, head.Timestamp, want),
		}
	}
	id := payloadID(head.Hash, uint64(attrs.Timestamp))
	s.mu.Lock()
	s.jobs[id] = &buildJob{parent: head, attrs: attrs}
	s.mu.Unlock()
	result.PayloadID = &id
	return result, nil
}

// payloadID is DETERMINISTIC: keccak(parentHash ‖ be64(timestamp))[:8].
//
// A stock EL picks an id from the attributes' hash and it does not matter what it is. Here it does:
// the shim is meant to be a pure function of (facts, attributes) so that a re-derivation after a
// reset produces byte-identical RPC traffic, which is what makes the dark-launch equality gate a
// comparison rather than an interpretation. A random id would make every trace differ.
func payloadID(parent common.Hash, timestamp uint64) eth.PayloadID {
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], timestamp)
	sum := crypto.Keccak256(parent[:], ts[:])
	var id eth.PayloadID
	copy(id[:], sum[:8])
	return id
}

func (s *Shim) dropJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = make(map[eth.PayloadID]*buildJob)
}

// GetPayloadV4 assembles the envelope for an open build job. THE FAIL-STOP LIVES HERE.
//
// The block it describes must already be a fact: either PROVEN — its hash, state root and
// message-passer root came off the wire, inside an accepted proof batch — or FORCED, computed
// by the forced-extension convention that the prover and the superroot program compute identically.
// There is no third kind. If neither holds, this returns an error and the chain's public rendering
// stops, which is the entire point: derivation may never outrun the proof stream, and a shim that
// invented a block would make the chain's public identity a claim by this process rather than by a
// proof.
//
// What comes off the wire (served verbatim): blockHash, stateRoot, withdrawalsRoot (= the
// message-passer storage root), number, timestamp. What is echoed: the transactions, exactly the
// bytes the CL supplied in the attributes. What is deterministic residue: receiptsRoot (the
// empty-receipts constant), logsBloom (zero), gasUsed (zero), baseFeePerGas (the frozen minimum).
// What is consensus-legal configuration: extraData and gasLimit (G2 D8 — NOT a marker).
func (e *EngineAPI) GetPayloadV4(ctx context.Context, id eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.getPayload(ctx, id)
}

// GetPayloadV5 is getPayloadV4 for a Karst-active chain. op-node decodes both into
// eth.ExecutionPayloadEnvelope and reads only the payload and the parent beacon root, so the two are
// the same answer; the generated silhouette rollup config activates Karst at genesis, which makes
// this the version actually called (op-node/rollup/types.go:777-780).
func (e *EngineAPI) GetPayloadV5(ctx context.Context, id eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	return e.getPayload(ctx, id)
}

func (e *EngineAPI) getPayload(ctx context.Context, id eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	if err := e.s.checkLive(); err != nil {
		return nil, err
	}
	s := e.s
	s.mu.Lock()
	job, ok := s.jobs[id]
	s.mu.Unlock()
	if !ok {
		return nil, eth.InputError{Code: eth.UnknownPayload, Inner: fmt.Errorf("no build job %s", id)}
	}

	if env, handled, err := s.buildDeniedReplacement(ctx, job); handled {
		if err != nil {
			s.log.Error("failed to prepare stock deposits-only replacement in the private EL",
				"parent", job.parent.Number, "timestamp", uint64(job.attrs.Timestamp), "err", err)
			return nil, eth.InputError{Code: eth.UnknownPayload, Inner: err}
		}
		s.mu.Lock()
		delete(s.jobs, id)
		s.mu.Unlock()
		return env, nil
	}

	fact, err := s.factForBuild(ctx, job)
	if err != nil {
		// Loud, and retryable by design. The CL maps UnknownPayload to ErrSealExpired
		// (build_seal.go), which lets the same attributes be re-attempted with a new job once the
		// facts arrive — a proof batch is minutes away, not never. What it must not do is produce a
		// block.
		s.log.Error("refusing to build a block with no proven-or-forced fact. Derivation "+
			"has outrun the proof stream, or the transcoder produced a block the facts do not cover.",
			"parent", job.parent.Number, "timestamp", uint64(job.attrs.Timestamp), "err", err)
		return nil, eth.InputError{Code: eth.UnknownPayload, Inner: err}
	}
	if fact.Replacement {
		rendering, ok := s.facts.Rendering(fact.Hash)
		if !ok {
			return nil, eth.InputError{Code: eth.UnknownPayload, Inner: fmt.Errorf("replacement block %d has no retained payload", fact.Number)}
		}
		env := payloadEnvelope(rendering.Header, fact.Hash, rendering.Txs)
		env.ParentBeaconBlockRoot = job.attrs.ParentBeaconBlockRoot
		s.mu.Lock()
		delete(s.jobs, id)
		s.mu.Unlock()
		return env, nil
	}

	txs := make([][]byte, len(job.attrs.Transactions))
	for i, tx := range job.attrs.Transactions {
		txs[i] = tx
	}
	origin, err := s.l1.InfoByHash(ctx, fact.L1Origin.Hash)
	if err != nil {
		return nil, fmt.Errorf("fetch rendered L1 origin %s of block %d: %w", fact.L1Origin, fact.Number, err)
	}
	hdr, err := RenderHeader(s.params, HeaderInputs{
		Parent:                   job.parent,
		Number:                   fact.Number,
		Timestamp:                fact.Timestamp,
		StateRoot:                fact.StateRoot,
		MessagePasserStorageRoot: fact.MessagePasserStorageRoot,
		Origin:                   origin,
		Txs:                      txs,
	})
	if err != nil {
		return nil, err
	}
	// The prevRandao the CL asked for must be the one the rendered origin implies, or the header this
	// node serves describes a different block from the one the CL believes it built. It is a free
	// cross-check of the rendered-origin convention (G2 D4) at the one place both sides meet.
	if got := eth.Bytes32(hdr.MixDigest); got != job.attrs.PrevRandao {
		return nil, eth.InputError{
			Code: eth.InvalidPayloadAttributes,
			Inner: fmt.Errorf("attributes carry prevRandao %s but the rendered L1 origin %s of block %d "+
				"has mixHash %s", job.attrs.PrevRandao, fact.L1Origin, fact.Number, got),
		}
	}

	s.facts.RecordRendering(Rendering{Header: hdr, Txs: txs, Hash: fact.Hash})
	s.mu.Lock()
	delete(s.jobs, id)
	s.mu.Unlock()

	env := payloadEnvelope(hdr, fact.Hash, txs)
	env.ParentBeaconBlockRoot = job.attrs.ParentBeaconBlockRoot
	s.log.Debug("served a proof-rendered payload", "number", fact.Number, "hash", fact.Hash,
		"forced", fact.Forced, "txs", len(txs))
	return env, nil
}

// buildDeniedReplacement handles the one case where a proof fact must be replaced. The stock
// supernode has already rewound its op-node and generated deposits-only attributes. The magic EL
// delegates execution of those exact attributes to P's real EL, then serves the resulting payload
// back through the ordinary op-node build path.
func (s *Shim) buildDeniedReplacement(ctx context.Context, job *buildJob) (*eth.ExecutionPayloadEnvelope, bool, error) {
	number := job.parent.Number + 1
	deniedFact, ok := s.facts.ByNumber(number)
	if !ok {
		deniedFact, ok = s.facts.DeniedFact(number)
		if !ok {
			return nil, false, nil
		}
	}
	denied, err := s.facts.Denied(deniedFact.Number, deniedFact.Hash)
	if err != nil {
		return nil, true, fmt.Errorf("check denial of proof fact %d (%s): %w", deniedFact.Number, deniedFact.Hash, err)
	}
	if !denied {
		return nil, false, nil
	}
	if !job.attrs.NoTxPool || !job.attrs.IsDepositsOnly() {
		return nil, true, fmt.Errorf("denied block %d replacement attributes are not stock deposits-only attributes", number)
	}
	s.mu.Lock()
	builder := s.replacementBuilder
	s.mu.Unlock()
	if builder == nil {
		return nil, true, fmt.Errorf("denied block %d requires a private replacement engine, but none is configured", number)
	}

	parentRef, err := s.ref(job.parent)
	if err != nil {
		return nil, true, err
	}
	env, err := builder.BuildReplacement(ctx, parentRef, job.attrs)
	if err != nil {
		return nil, true, err
	}
	replacement, rendering, err := factFromReplacement(s.params.Rollup, job.parent, job.attrs, env)
	if err != nil {
		return nil, true, err
	}

	// Commit the handoff only after the private EL has successfully imported and the payload has
	// passed every continuity/hash/body check. A failed external build leaves the denied facts intact
	// and is retryable through the stock seal-expired path.
	if s.isGenesis(job.parent.Number) {
		s.facts.replaceAllForSupersession()
	} else if err := s.facts.ReplaceSuffix(job.parent); err != nil {
		return nil, true, fmt.Errorf("drop denied proof suffix at block %d: %w", number, err)
	}
	s.facts.RecordRendering(rendering)
	s.mu.Lock()
	s.replacementsByHash[replacement.Hash] = replacement
	s.replacementsByNum[replacement.Number] = replacement
	s.mu.Unlock()
	s.log.Warn("prepared stock deposits-only replacement in P's private EL",
		"number", replacement.Number, "denied_hash", deniedFact.Hash,
		"replacement_hash", replacement.Hash, "parent", job.parent.Hash)
	return env, true, nil
}

// factForBuild resolves the facts of the block a build job describes, computing a forced block's
// facts on demand when the convention defines one and the store has not recorded it yet.
//
// The on-demand path is what keeps DR-2's designed liveness: the stock pipeline force-generates
// blocks the moment the sequencing window expires, and it does so upstream of anything a verifier
// controls, so an engine that could only serve pre-recorded forced facts would stall the chain's
// public rendering exactly when a dead prover is supposed to cost nothing. The computed block goes
// into the fact store, so the transcoder's chaining sees the same forced head this engine served.
func (s *Shim) factForBuild(ctx context.Context, job *buildJob) (Fact, error) {
	number := job.parent.Number + 1
	timestamp := uint64(job.attrs.Timestamp)
	if fact, ok := s.factByNumber(number); ok {
		denied, err := s.facts.Denied(fact.Number, fact.Hash)
		if err != nil {
			return Fact{}, fmt.Errorf("check denial of proof fact %d (%s): %w", fact.Number, fact.Hash, err)
		}
		if denied {
			return Fact{}, fmt.Errorf("denied proof fact %d (%s) requires the private replacement path", fact.Number, fact.Hash)
		} else {
			if fact.Timestamp != timestamp {
				return Fact{}, fmt.Errorf("block %d is a fact at timestamp %d, but the attributes "+
					"ask for timestamp %d", number, fact.Timestamp, timestamp)
			}
			if parent, err := s.parentOf(fact); err != nil {
				return Fact{}, err
			} else if parent.Hash != job.parent.Hash {
				return Fact{}, fmt.Errorf("block %d is a fact whose parent is %s, but the build job "+
					"was opened on parent %s", number, parent.Hash, job.parent.Hash)
			}
			return fact, nil
		}
	}

	origin, seqNumber, err := originFromAttributes(s.params.Rollup, job.attrs)
	if err != nil {
		return Fact{}, fmt.Errorf("no fact for block %d, and its attributes are not readable "+
			"as a forced block: %w", number, err)
	}
	if len(job.attrs.Transactions) != 1 {
		return Fact{}, fmt.Errorf("no fact for block %d; it carries %d transactions, and a "+
			"forced block carries exactly one (the L1-info deposit)", number, len(job.attrs.Transactions))
	}
	forced, err := ForcedBlockAt(ctx, s.params, s.l1, job.parent, origin, seqNumber, timestamp)
	if err != nil {
		return Fact{}, err
	}
	s.log.Warn("the sequencing window expired: recording a FORCED block from the convention. A dead "+
		"prover costs P its own progress and never the dependency set's frontier (DR-2), and this block "+
		"exports nothing.", "number", forced.Number, "hash", forced.Hash, "l1_origin", forced.L1Origin)
	s.facts.Record(forced)
	return forced, nil
}

// NewPayloadV4 checks continuity and answers VALID. It NEVER executes.
//
// "VALID" here means exactly two things: the payload extends the chain this node knows (parent hash,
// number and timestamp all line up with the facts) and its block hash IS the hash the proof
// committed to at that height. Anything else is INVALID.
//
// The sole accepted non-fact payload is the stock chain container's exact rewind sentinel: the
// canonical payload with only ExtraData toggled and its hash recomputed. It is a temporary fork used
// to move the engine head to the invalid block's parent; it is never inserted into the proof facts,
// never returned by number, and is discarded as soon as forkchoice returns to a proven head. The
// private LightCL sequencer, not this verifier, produces the replacement block.
//
// A payload at a height with no fact at all is refused without halting: it is what a rewind that
// crossed this call looks like, and the CL's own reset resolves it. Either way nothing is inserted,
// so the safety of the answer does not depend on telling the two apart — only its visibility does.
func (e *EngineAPI) NewPayloadV4(ctx context.Context, payload *eth.ExecutionPayload,
	versionedHashes []common.Hash, parentBeaconBlockRoot *common.Hash, executionRequests []hexutil.Bytes,
) (*eth.PayloadStatusV1, error) {
	if err := e.s.checkLive(); err != nil {
		return nil, err
	}
	s := e.s
	if payload == nil {
		return nil, eth.InputError{Code: eth.InvalidParams, Inner: errors.New("nil payload")}
	}
	if len(versionedHashes) != 0 {
		// P's blocks contain no blob transactions: there is no DA and no user traffic on the public
		// side at all. A versioned hash here is a payload from somewhere this chain does not have.
		return invalid(s, payload, fmt.Errorf("payload carries %d blob versioned hashes; a silhouette "+
			"block has none", len(versionedHashes)))
	}
	if len(executionRequests) != 0 {
		return invalid(s, payload, fmt.Errorf("payload carries %d execution requests; Isthmus forces an "+
			"empty request list", len(executionRequests)))
	}

	number := uint64(payload.BlockNumber)
	fact, ok := s.factByNumber(number)
	if !ok {
		s.log.Error("newPayload for a height with no proven-or-forced fact: refusing without halting, "+
			"because this is what a fact rewind across the build dance looks like",
			"number", number, "hash", payload.BlockHash)
		return invalid(s, payload, fmt.Errorf("no proven-or-forced fact exists for block %d", number))
	}

	parent, err := s.parentOf(fact)
	if err != nil {
		return invalid(s, payload, err)
	}

	if payload.BlockHash != fact.Hash {
		reason := fmt.Errorf("payload for block %d claims hash %s; the proven-or-forced fact for that "+
			"height is %s", number, payload.BlockHash, fact.Hash)
		if payload.ParentHash == parent.Hash {
			if err := s.acceptRewindPayload(payload, fact); err == nil {
				return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &payload.BlockHash}, nil
			} else {
				s.halt(fmt.Errorf("newPayload offered a non-canonical block for block %d on the same parent %s: %w: %v",
					number, parent.Hash, reason, err))
			}
		}
		return invalid(s, payload, reason)
	}
	if payload.ParentHash != parent.Hash {
		return invalid(s, payload, fmt.Errorf("payload for block %d names parent %s; the fact chain's "+
			"parent is %s", number, payload.ParentHash, parent.Hash))
	}
	if uint64(payload.Timestamp) != fact.Timestamp {
		return invalid(s, payload, fmt.Errorf("payload for block %d has timestamp %d; the fact says %d",
			number, uint64(payload.Timestamp), fact.Timestamp))
	}

	return &eth.PayloadStatusV1{Status: eth.ExecutionValid, LatestValidHash: &fact.Hash}, nil
}

// acceptRewindPayload recognizes exactly the temporary payload constructed by
// chain_container/engine_controller.Rewind. Accepting this one mechanical fork lets the unmodified
// stock rewind protocol operate against an execution service that otherwise accepts only facts.
func (s *Shim) acceptRewindPayload(payload *eth.ExecutionPayload, canonical Fact) error {
	rendering, ok := s.facts.Rendering(canonical.Hash)
	if !ok {
		return fmt.Errorf("canonical block %s has no rendering to validate the rewind payload against", canonical.Hash)
	}
	expectedEnvelope := payloadEnvelope(rendering.Header, canonical.Hash, rendering.Txs)
	expectedEnvelope.ParentBeaconBlockRoot = rendering.Header.ParentBeaconRoot
	expected := expectedEnvelope.ExecutionPayload
	extra := append(eth.BytesMax32(nil), expected.ExtraData...)
	if len(extra) == 0 {
		extra = []byte{0x00}
	} else {
		extra[len(extra)-1] ^= 0xff
	}
	expected.ExtraData = extra
	expectedHash, _ := expectedEnvelope.CheckBlockHash()
	expected.BlockHash = expectedHash
	if !reflect.DeepEqual(expected, payload) {
		return fmt.Errorf("payload is not the stock extraData-only rewind sentinel")
	}
	rewind := canonical
	rewind.Hash = expectedHash
	hdr := types.CopyHeader(rendering.Header)
	hdr.Extra = append([]byte(nil), extra...)
	s.facts.RecordRendering(Rendering{Header: hdr, Txs: rendering.Txs, Hash: expectedHash})
	s.recordRewindFact(rewind)
	s.log.Info("accepted temporary stock rewind sentinel", "number", rewind.Number,
		"canonical", canonical.Hash, "sentinel", expectedHash)
	return nil
}

// invalid builds the INVALID answer, with the latest valid hash set to the head this node stands on
// so the CL knows where to come back to.
func invalid(s *Shim, payload *eth.ExecutionPayload, reason error) (*eth.PayloadStatusV1, error) {
	head := s.head()
	msg := reason.Error()
	s.log.Error("newPayload INVALID: the shim describes proven-or-forced blocks and nothing else",
		"number", uint64(payload.BlockNumber), "hash", payload.BlockHash, "reason", msg)
	return &eth.PayloadStatusV1{
		Status:          eth.ExecutionInvalid,
		LatestValidHash: &head.Hash,
		ValidationError: &msg,
	}, nil
}

// payloadEnvelope turns a rendered header and body into the envelope shape op-node decodes.
//
// blockHash is passed in rather than taken from the header: for a proven block those two disagree by
// design, and this is the exact line where the disagreement enters the wire. Everything downstream
// takes the hash at face value (trustCache), which is what makes the chain's public identity its
// real one.
func payloadEnvelope(hdr *types.Header, blockHash common.Hash, txs [][]byte) *eth.ExecutionPayloadEnvelope {
	opaque := make([]eth.Data, len(txs))
	for i, tx := range txs {
		opaque[i] = tx
	}
	payload := &eth.ExecutionPayload{
		ParentHash:    hdr.ParentHash,
		FeeRecipient:  hdr.Coinbase,
		StateRoot:     eth.Bytes32(hdr.Root),
		ReceiptsRoot:  eth.Bytes32(hdr.ReceiptHash),
		LogsBloom:     eth.Bytes256(hdr.Bloom),
		PrevRandao:    eth.Bytes32(hdr.MixDigest),
		BlockNumber:   eth.Uint64Quantity(bigs.Uint64Strict(hdr.Number)),
		GasLimit:      eth.Uint64Quantity(hdr.GasLimit),
		GasUsed:       eth.Uint64Quantity(hdr.GasUsed),
		Timestamp:     eth.Uint64Quantity(hdr.Time),
		ExtraData:     eth.BytesMax32(hdr.Extra),
		BaseFeePerGas: eth.Uint256Quantity(*uint256FromBig(hdr.BaseFee)),
		BlockHash:     blockHash,
		Transactions:  opaque,
	}
	if hdr.WithdrawalsHash != nil {
		// Canyon+ bodies carry an empty withdrawals list; Isthmus+ additionally puts the
		// message-passer storage root in the header field, and that is the value stock outputV0 reads.
		empty := types.Withdrawals{}
		payload.Withdrawals = &empty
		if *hdr.WithdrawalsHash != types.EmptyWithdrawalsHash {
			root := *hdr.WithdrawalsHash
			payload.WithdrawalsRoot = &root
		}
	}
	if hdr.BlobGasUsed != nil {
		v := eth.Uint64Quantity(*hdr.BlobGasUsed)
		payload.BlobGasUsed = &v
	}
	if hdr.ExcessBlobGas != nil {
		v := eth.Uint64Quantity(*hdr.ExcessBlobGas)
		payload.ExcessBlobGas = &v
	}
	return &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}
}
