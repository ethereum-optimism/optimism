// Package sp1groth16 verifies SP1 Groth16 proofs over BN254 with gnark-crypto.
//
// It mirrors sp1-verifier 6.8.0 (src/groth16/mod.rs, converter.rs, verify.rs) for the SHA-256
// public-values digest only. The blake3 digest that sp1-verifier also tries is deliberately not
// accepted: the private-projection guest commits its public values with SHA-256 semantics and a
// second accepted digest would be a second statement encoding.
//
// The package is pure. It does no I/O beyond the embedded production verifying key.
package sp1groth16

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

const (
	// ProofLength is the SP1 Groth16 proof layout of SP1ProofWithPublicValues::bytes():
	// 4-byte circuit prefix ‖ exit code ‖ vk root ‖ proof nonce ‖ 256-byte gnark proof.
	ProofLength = 356
	// PublicInputCount is the number of circuit public inputs: program vkey, public-values digest,
	// exit code, vk root and proof nonce.
	PublicInputCount = 5

	prefixLength     = 4
	gnarkProofOffset = prefixLength + 3*32
	vkMinLength      = 292
)

// Circuit pins one SP1 Groth16 wrap circuit: its gnark verifying key and the recursion vk root
// that every proof over it must carry.
type Circuit struct {
	VK     []byte
	VKRoot [32]byte
}

//go:embed vk/groth16_vk_v6.1.0.bin
var groth16VKV6_1_0 []byte

// CircuitV6_1_0 is the SP1 Groth16 circuit shipped with sp1-sdk / sp1-verifier 6.8.0
// (sp1-contracts v6.1.0 SP1VerifierGroth16). It is the circuit pinned by the verifier ID
// sp1-private-projection-v1.
var CircuitV6_1_0 = Circuit{
	VK: groth16VKV6_1_0,
	VKRoot: [32]byte{
		0x00, 0x2f, 0x85, 0x0e, 0xe9, 0x98, 0x97, 0x4d, 0x6c, 0xc0, 0x0e, 0x50, 0xcd, 0x08, 0x14, 0xb0,
		0x98, 0xc0, 0x5b, 0xfa, 0xde, 0x46, 0x6d, 0x28, 0x57, 0x32, 0x40, 0xd0, 0x57, 0xf2, 0x53, 0x52,
	},
}

var (
	ErrProofLength   = errors.New("sp1 groth16 proof has the wrong length")
	ErrCircuitPrefix = errors.New("sp1 groth16 proof is for a different circuit")
	ErrExitCode      = errors.New("sp1 groth16 proof has a non-zero exit code")
	ErrVKRoot        = errors.New("sp1 groth16 proof has the wrong vk root")
	ErrVerifyingKey  = errors.New("invalid sp1 groth16 verifying key")
	ErrProofPoint    = errors.New("invalid sp1 groth16 proof point")
	ErrPublicInput   = errors.New("sp1 groth16 public input is not a canonical field element")
	ErrPairing       = errors.New("sp1 groth16 pairing check failed")
)

// PublicValuesDigest is SP1's committed-values digest: sha256 with the top three bits cleared so
// that it is a BN254 scalar.
func PublicValuesDigest(publicValues []byte) [32]byte {
	d := sha256.Sum256(publicValues)
	d[0] &= 0x1f
	return d
}

// CircuitPrefix is the first four bytes of sha256(vk), which SP1 prepends to every proof.
func (c Circuit) CircuitPrefix() [4]byte {
	h := sha256.Sum256(c.VK)
	return [4]byte(h[:4])
}

type verifyingKey struct {
	alpha              bn254.G1Affine
	beta, gamma, delta bn254.G2Affine
	k                  []bn254.G1Affine
}

func parseVerifyingKey(vk []byte) (*verifyingKey, error) {
	if len(vk) < vkMinLength {
		return nil, fmt.Errorf("%w: %d bytes", ErrVerifyingKey, len(vk))
	}
	var out verifyingKey
	if err := setG1(&out.alpha, vk[0:32]); err != nil {
		return nil, fmt.Errorf("%w: alpha: %w", ErrVerifyingKey, err)
	}
	for _, p := range []struct {
		dst *bn254.G2Affine
		src []byte
	}{{&out.beta, vk[64:128]}, {&out.gamma, vk[128:192]}, {&out.delta, vk[224:288]}} {
		if err := setG2(p.dst, p.src); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrVerifyingKey, err)
		}
	}
	nK := binary.BigEndian.Uint32(vk[288:292])
	if nK != PublicInputCount+1 {
		return nil, fmt.Errorf("%w: %d K points, expected %d", ErrVerifyingKey, nK, PublicInputCount+1)
	}
	if uint64(len(vk)) < vkMinLength+uint64(nK)*32 {
		return nil, fmt.Errorf("%w: truncated K points", ErrVerifyingKey)
	}
	out.k = make([]bn254.G1Affine, nK)
	for i := range out.k {
		off := vkMinLength + 32*i
		if err := setG1(&out.k[i], vk[off:off+32]); err != nil {
			return nil, fmt.Errorf("%w: K[%d]: %w", ErrVerifyingKey, i, err)
		}
	}
	return &out, nil
}

// setG1 and setG2 require the encoding to consume exactly the given bytes: 32/64 for gnark
// compressed points (verifying key) and 64/128 for uncompressed points (proof). SetBytes performs
// the curve and subgroup checks.
func setG1(p *bn254.G1Affine, b []byte) error {
	n, err := p.SetBytes(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("G1 encoding length %d, expected %d", n, len(b))
	}
	return nil
}

func setG2(p *bn254.G2Affine, b []byte) error {
	n, err := p.SetBytes(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return fmt.Errorf("G2 encoding length %d, expected %d", n, len(b))
	}
	return nil
}

// Verify checks an SP1 Groth16 proof (356 bytes) of the program programVKey committing exactly
// publicValues, over the circuit c. It returns nil only if the pairing check passes.
func Verify(c Circuit, proof356 []byte, programVKey [32]byte, publicValues []byte) error {
	if len(proof356) != ProofLength {
		return fmt.Errorf("%w: %d bytes, expected %d", ErrProofLength, len(proof356), ProofLength)
	}
	if prefix := c.CircuitPrefix(); !bytes.Equal(proof356[:prefixLength], prefix[:]) {
		return ErrCircuitPrefix
	}
	exit := [32]byte(proof356[4:36])
	if exit != ([32]byte{}) {
		return ErrExitCode
	}
	vkRoot := [32]byte(proof356[36:68])
	if vkRoot != c.VKRoot {
		return ErrVKRoot
	}
	nonce := [32]byte(proof356[68:100])
	raw := proof356[gnarkProofOffset:]

	vk, err := parseVerifyingKey(c.VK)
	if err != nil {
		return err
	}
	var a, cc bn254.G1Affine
	var b bn254.G2Affine
	if err := setG1(&a, raw[0:64]); err != nil {
		return fmt.Errorf("%w: A: %w", ErrProofPoint, err)
	}
	if err := setG2(&b, raw[64:192]); err != nil {
		return fmt.Errorf("%w: B: %w", ErrProofPoint, err)
	}
	if err := setG1(&cc, raw[192:256]); err != nil {
		return fmt.Errorf("%w: C: %w", ErrProofPoint, err)
	}

	digest := PublicValuesDigest(publicValues)
	inputs := [PublicInputCount][32]byte{programVKey, digest, exit, vkRoot, nonce}
	// L = K[0] + Σ x_i·K[i+1]
	l := vk.k[0]
	for i, in := range inputs {
		var x fr.Element
		if err := x.SetBytesCanonical(in[:]); err != nil {
			return fmt.Errorf("%w: input %d: %w", ErrPublicInput, i, err)
		}
		if x.IsZero() {
			continue
		}
		var term bn254.G1Affine
		term.ScalarMultiplication(&vk.k[i+1], x.BigInt(new(big.Int)))
		l.Add(&l, &term)
	}
	var negA bn254.G1Affine
	negA.Neg(&a)
	ok, err := bn254.PairingCheck(
		[]bn254.G1Affine{negA, vk.alpha, l, cc},
		[]bn254.G2Affine{b, vk.beta, vk.gamma, vk.delta},
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPairing, err)
	}
	if !ok {
		return ErrPairing
	}
	return nil
}
