package silhouette

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// THE SEQUENCER POSTURE'S ONE PUBLIC SURFACE.
//
// In the verifier posture the `silhouette` namespace is the shim's, and it exists to say what is
// fabricated (shim_declare.go). There is no shim here — this node fronts P's REAL execution client,
// which answers `eth_*` for itself and answers it honestly — so the question this namespace has to
// answer is a different one.
//
// It is the question the whole posture turns on: WHAT HAS BEEN PROVEN. On this node P's public
// safety labels are taken from the proven head rather than from a derivation pipeline
// (Container.OptimisticAt), and that head is not visible anywhere else. `optimism_syncStatus` reports
// the virtual node's own labels, which on a chain with no batcher sit at genesis forever and are
// SUPPOSED to; the readiness check the cross-safety round runs consults the container instead. So
// without this method the load-bearing state of the sequencer side is unreadable, and an operator
// confirming a cutover would have nothing to confirm it against but chain A's frontier moving.
//
// It also states the disarm. Reporting both sequencing windows — the committed one the proofs and
// the forced-extension convention are computed under, and the effectively-infinite one this node's
// own pipeline runs with — puts the one deliberate divergence in the deployment where an operator
// reads it, rather than leaving it as a constant in a Go file.

// ProvenHeadAPI is the `silhouette` RPC namespace in the sequencer posture.
type ProvenHeadAPI struct {
	facts   *FactStore
	tracker *ProvenHeadTracker
	// committedSeqWindow and pipelineSeqWindow are reported rather than recomputed: they are the two
	// halves of G4 D5 and the point is to show that they differ on purpose.
	committedSeqWindow uint64
	pipelineSeqWindow  uint64
}

// ProvenBlock is one block of P as this node knows it publicly.
type ProvenBlock struct {
	Number    hexutil.Uint64 `json:"number"`
	Hash      common.Hash    `json:"hash"`
	Timestamp hexutil.Uint64 `json:"timestamp"`
	// Forced is true for a block no proof carried: one the forced-extension convention computed
	// (DR-2). It is reported rather than smoothed over, because "proven" and "forced" are different
	// claims and a consumer that could not tell them apart would be told a proof exists where none
	// does.
	Forced bool `json:"forced"`
	// Carrier is the L1 block that carried this block's proof batch. It is ABSENT for a forced block,
	// which has no carrier (G3 D8) — the label surface substitutes the nearest proven ancestor's
	// carrier as a deliberate understatement (G4 D2), and that substitution belongs there rather than
	// here, where the question is what this node was told.
	Carrier *eth.BlockID `json:"carrier,omitempty"`
}

// ProvenHeadStatus is the sequencer side's answer to "how far has P been proven".
type ProvenHeadStatus struct {
	// Head is the top of proven-or-forced history, absent when nothing has been proven yet. Absent is
	// the correct cold-start state and not an error: a sequencer posture brought up before its first
	// proof batch lands has a real chain and no public history of it.
	Head *ProvenBlock `json:"head,omitempty"`
	// Oldest is the bottom of the window this node still holds, which is what bounds the timestamps
	// the cross-safety round can be answered for (Container.FirstSafeHeadTimestamp).
	Oldest *ProvenBlock `json:"oldest,omitempty"`
	// TrackerCursor is the next L1 block the proven-head walk will read. Compared against the L1 head
	// it is the one number that distinguishes "no proofs are landing" from "this node has stopped
	// looking", which are the same symptom and different incidents.
	TrackerCursor hexutil.Uint64 `json:"trackerCursor"`
	// CommittedSeqWindowSize is the sequencing window the proofs, the prover and every verifier are
	// computed under, including the forced-extension convention this node applies.
	CommittedSeqWindowSize hexutil.Uint64 `json:"committedSeqWindowSize"`
	// PipelineSeqWindowSize is the window this node's OWN derivation pipeline runs with, which is
	// effectively infinite on purpose (G4 D5). A value equal to the committed one here means the
	// disarm is not in place and this node will reorg P's real chain out when the window expires.
	PipelineSeqWindowSize hexutil.Uint64 `json:"pipelineSeqWindowSize"`
}

// ProvenHead reports what this node knows publicly about P.
func (a *ProvenHeadAPI) ProvenHead() *ProvenHeadStatus {
	out := &ProvenHeadStatus{
		TrackerCursor:          hexutil.Uint64(a.tracker.Cursor()),
		CommittedSeqWindowSize: hexutil.Uint64(a.committedSeqWindow),
		PipelineSeqWindowSize:  hexutil.Uint64(a.pipelineSeqWindow),
	}
	if head, ok := a.facts.Head(); ok {
		out.Head = a.describe(head)
	}
	if oldest, ok := a.facts.Oldest(); ok {
		out.Oldest = a.describe(oldest)
	}
	return out
}

func (a *ProvenHeadAPI) describe(f Fact) *ProvenBlock {
	blk := &ProvenBlock{
		Number:    hexutil.Uint64(f.Number),
		Hash:      f.Hash,
		Timestamp: hexutil.Uint64(f.Timestamp),
		Forced:    f.Forced,
	}
	if carrier, ok := a.facts.CarrierOf(f.Number); ok {
		blk.Carrier = &carrier
	}
	return blk
}

// provenHeadAPIs is the namespace the sequencer posture mounts on P's own route.
func provenHeadAPIs(facts *FactStore, tracker *ProvenHeadTracker, committed, pipeline uint64) []rpc.API {
	return []rpc.API{{
		Namespace: "silhouette",
		Service: &ProvenHeadAPI{
			facts: facts, tracker: tracker,
			committedSeqWindow: committed, pipelineSeqWindow: pipeline,
		},
	}}
}
