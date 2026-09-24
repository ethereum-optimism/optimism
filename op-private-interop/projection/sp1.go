package projection

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection/sp1groth16"
	"github.com/ethereum/go-ethereum/common"
)

// Envelope framing of the claim's proof slot under sp1-private-projection-v1 (§F.2):
//
//	version(1) = 0x01 ‖ kind(1) ‖ L = u16be(len(proof)) ‖ proof(L) ‖ PublicValuesV1(672)
const (
	EnvelopeVersion     byte = 0x01
	EnvelopeKindGroth16 byte = 0x01 // SP1 Groth16 over the circuit pinned by the verifier ID
	EnvelopeKindMock    byte = 0x02 // SP1 mock: five 32-byte public-input words

	envelopeHeaderLength = 4
	// MockProofLength is five words [vkeyHash, committedValuesDigest, exitCode, vkRoot, proofNonce].
	MockProofLength = 5 * 32
	// Groth16EnvelopeLength and MockEnvelopeLength are the only valid envelope sizes.
	Groth16EnvelopeLength = envelopeHeaderLength + sp1groth16.ProofLength + PublicValuesLength
	MockEnvelopeLength    = envelopeHeaderLength + MockProofLength + PublicValuesLength
)

// Envelope is a decoded proof slot.
type Envelope struct {
	Kind         byte
	Proof        []byte
	PublicValues [PublicValuesLength]byte
}

// DecodeEnvelope is strict: known version and kind, the kind's exact proof length, and a total
// length of exactly 676+L (no trailing bytes).
func DecodeEnvelope(b []byte) (*Envelope, error) {
	if len(b) < envelopeHeaderLength {
		return nil, fmt.Errorf("sp1 envelope truncated: %d bytes", len(b))
	}
	if b[0] != EnvelopeVersion {
		return nil, fmt.Errorf("unsupported sp1 envelope version %d", b[0])
	}
	l := int(binary.BigEndian.Uint16(b[2:4]))
	switch b[1] {
	case EnvelopeKindGroth16:
		if l != sp1groth16.ProofLength {
			return nil, fmt.Errorf("sp1 groth16 envelope proof length %d, expected %d", l, sp1groth16.ProofLength)
		}
	case EnvelopeKindMock:
		if l != MockProofLength {
			return nil, fmt.Errorf("sp1 mock envelope proof length %d, expected %d", l, MockProofLength)
		}
	default:
		return nil, fmt.Errorf("unsupported sp1 envelope kind %d", b[1])
	}
	if len(b) != envelopeHeaderLength+l+PublicValuesLength {
		return nil, fmt.Errorf("sp1 envelope length %d, expected %d", len(b), envelopeHeaderLength+l+PublicValuesLength)
	}
	e := &Envelope{Kind: b[1], Proof: append([]byte(nil), b[envelopeHeaderLength:envelopeHeaderLength+l]...)}
	copy(e.PublicValues[:], b[envelopeHeaderLength+l:])
	return e, nil
}

// EncodeEnvelope frames e. It does not validate the kind or proof length; DecodeEnvelope does.
// A proof longer than 65535 bytes cannot be framed and yields nil.
func EncodeEnvelope(e *Envelope) []byte {
	if len(e.Proof) > 0xffff {
		return nil
	}
	out := make([]byte, 0, envelopeHeaderLength+len(e.Proof)+PublicValuesLength)
	out = append(out, EnvelopeVersion, e.Kind)
	out = binary.BigEndian.AppendUint16(out, uint16(len(e.Proof)))
	out = append(out, e.Proof...)
	return append(out, e.PublicValues[:]...)
}

// MockEnvelope is the kind-0x02 envelope an SP1 mock prover produces for publicValues:
// words [programVKey, masked sha256(publicValues), 0, 0, 0]. It proves nothing.
func MockEnvelope(programVKey common.Hash, publicValues [PublicValuesLength]byte) []byte {
	digest := sp1groth16.PublicValuesDigest(publicValues[:])
	proof := make([]byte, MockProofLength)
	copy(proof[0:32], programVKey[:])
	copy(proof[32:64], digest[:])
	return EncodeEnvelope(&Envelope{Kind: EnvelopeKindMock, Proof: proof, PublicValues: publicValues})
}

// SP1Verifier implements sp1-private-projection-v1 verification (§F.2). It is pure. Construct it
// with VerifierFor, which applies the §B.5 gate to allowMock.
type SP1Verifier struct {
	cfg       Config
	allowMock bool
}

func (v SP1Verifier) Verify(s Statement, proof []byte) error {
	env, err := DecodeEnvelope(proof)
	if err != nil {
		return err
	}
	if want := PublicValues(&s); env.PublicValues != want {
		for i := 0; i < PublicValuesWords; i++ {
			if [32]byte(env.PublicValues[32*i:32*i+32]) != [32]byte(want[32*i:32*i+32]) {
				return fmt.Errorf("sp1 public values differ from the admission statement at word %d", i)
			}
		}
	}
	switch env.Kind {
	case EnvelopeKindGroth16:
		return sp1groth16.Verify(sp1groth16.CircuitV6_1_0, env.Proof, v.cfg.ProgramVKey, env.PublicValues[:])
	case EnvelopeKindMock:
		if !v.allowMock {
			return fmt.Errorf("sp1 mock envelopes are not enabled")
		}
		if common.Hash(env.Proof[0:32]) != v.cfg.ProgramVKey {
			return fmt.Errorf("sp1 mock envelope program vkey mismatch")
		}
		if digest := sp1groth16.PublicValuesDigest(env.PublicValues[:]); [32]byte(env.Proof[32:64]) != digest {
			return fmt.Errorf("sp1 mock envelope public values digest mismatch")
		}
		if [96]byte(env.Proof[64:160]) != ([96]byte{}) {
			return fmt.Errorf("sp1 mock envelope exit code, vk root or nonce is non-zero")
		}
		return nil
	default:
		return fmt.Errorf("unsupported sp1 envelope kind %d", env.Kind)
	}
}

// bn254ScalarModulus is the BN254 scalar field modulus r. program_vkey must be below it, since it
// is Groth16 public input 0.
var bn254ScalarModulus = fr.Modulus()

func isBN254Scalar(h common.Hash) bool {
	return new(big.Int).SetBytes(h[:]).Cmp(bn254ScalarModulus) < 0
}
