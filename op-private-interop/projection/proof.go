package projection

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// AdmissionDigest binds the entire public range and independently collected canonical
// continuation. The records root already commits to every normalized claim field,
// output, message, and transaction envelope. Proof bytes and carrier signatures are
// excluded to avoid circularity. This encoding is shared with Kona.
func AdmissionDigest(s Statement) common.Hash {
	var number [8]byte
	binary.BigEndian.PutUint64(number[:], s.Continuation.Anchor.Number)
	return crypto.Keccak256Hash([]byte("optimism.private-admission.v1\x00"),
		s.ChainID[:], s.ParentHash[:], s.ProjectionHash[:], number[:],
		s.Continuation.Anchor.Hash[:], s.Continuation.OutputRoot[:], s.Continuation.RecoveryHash[:])
}

// ExecutionMockProof is an explicitly forgeable test envelope. The native producer
// must run the execution relation before emitting it. Admission cannot establish
// that the producer did so; this mode provides no private-execution soundness.
// The prefix versions both the envelope and the tested program relation.
func ExecutionMockProof(s Statement) []byte {
	digest := AdmissionDigest(s)
	return append([]byte("optimism.private-execution.mock.v1\x00"), digest[:]...)
}

// ExecutionMockVerifier checks the complete envelope against the canonical public
// statement. It never accepts missing bytes, other proof modes, or a different span.
// It intentionally does not claim to authenticate the private execution context.
type ExecutionMockVerifier struct{}

func (ExecutionMockVerifier) Verify(s Statement, proof []byte) error {
	if !bytes.Equal(proof, ExecutionMockProof(s)) {
		return fmt.Errorf("execution mock envelope does not match canonical span")
	}
	return nil
}
