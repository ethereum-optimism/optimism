package silhouette

import (
	"errors"
	"fmt"
)

// ErrProofNotAttested is returned when a batch carries proof bytes to a verifier running the attested
// proving system. It is a SPEC rule, not a bring-up quirk: see AttestedVerifier.
var ErrProofNotAttested = errors.New("attested mode requires an empty proof slot")

// Verifier decides whether a proof-batch's public values are proven, under whichever proving system
// this node is configured for. It is the ONLY acceptance rule that differs between proving systems:
// every binding check against the node's own L1 view — rules 1 through 4, the chaining, the
// structural checks, the same-timestamp refusal, the import list — is enforced identically either way.
type Verifier interface {
	// Verify returns nil iff the batch's public values are proven to this verifier's satisfaction.
	// The bytes are the ones read off L1, never a re-encoding.
	Verify(publicValues []byte, proof []byte) error
	// ProofType names the proving system this verifier enforces, for logging and status.
	ProofType() ProofType
}

// AttestedVerifier is v1's proving system: the proof of the chain is the OPERATOR'S ATTESTATION, and
// the attestation is the L1 batch transaction's own signature. There is no separate signature on the
// wire because there does not need to be one — acceptance rule 1 already requires
// `tx.from == submitterEOA` (see DataSource.isProofBatchTx), so a batch that reached this verifier at
// all was signed by the configured operator, and it was signed over the blob hashes the envelope was
// read from. Attester and submitter are the same key by construction, not by convention.
//
// This is NOT a stub, and the naming carries the whole point: an attested batch is a claim the
// operator has staked their identity on, and this node accepts it on that basis. What that does and
// does not buy is written down in docs/TRUST-MODEL.md — authenticity, structure and cross-chain import
// consistency are real; state validity is not.
//
// The empty proof slot is enforced, and loudly. A batch carrying proof bytes is refused rather than
// waved through, because waving it through is the one behaviour that would make this mode a hole:
// bytes on the wire that look like a proof, nobody checking them, and no error to say so.
type AttestedVerifier struct{}

func (AttestedVerifier) Verify(_ []byte, proof []byte) error {
	if len(proof) != 0 {
		return fmt.Errorf("%w: this batch carries %d proof bytes, and a proof this node cannot check "+
			"must not look like one it did", ErrProofNotAttested, len(proof))
	}
	return nil
}

func (AttestedVerifier) ProofType() ProofType { return ProofTypeAttested }
