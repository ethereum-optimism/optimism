package silhouette

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestAttestedVerifier pins v1's proving system, including the half that is a SPEC RULE rather than a
// convenience: the proof slot must be EMPTY.
//
// The rule reads backwards until you say what it is protecting. An attested verifier cannot check a
// proof, so proof bytes arriving at one are bytes nobody will ever look at. Accepting the batch
// anyway would put the system in its worst state — a wire that carries something proof-shaped, an
// operator who believes it was checked, and no error anywhere to disagree. Refusing makes the
// configuration error loud and local, at the cost of one rejected batch.
func TestAttestedVerifier(t *testing.T) {
	v := AttestedVerifier{}
	require.Equal(t, ProofTypeAttested, v.ProofType())

	t.Run("an empty proof slot is the attestation", func(t *testing.T) {
		// Nothing is verified HERE because the attestation is not in the envelope: it is the L1
		// transaction's own signature, enforced by acceptance rule 1 (DataSource.isProofBatchTx). This
		// verifier's whole job is to insist that nothing else claims to be a proof.
		require.NoError(t, v.Verify([]byte("anything"), nil))
		require.NoError(t, v.Verify([]byte("anything"), []byte{}))
	})

	t.Run("proof bytes are refused, loudly", func(t *testing.T) {
		for _, proof := range [][]byte{{1}, make([]byte, 260), make([]byte, 100_000)} {
			err := v.Verify([]byte("anything"), proof)
			require.ErrorIs(t, err, ErrProofNotAttested,
				"a batch carrying %d proof bytes must be refused, not silently accepted", len(proof))
			require.ErrorContains(t, err, "proof bytes")
		}
	})
}

// TestProofTypeIsNeverADefault pins the config rule that the trust model is always stated.
//
// A default is wrong in a direction that cannot be recovered from by reading logs. Today's only
// candidate is `attested` itself, and a node that defaulted to it would rest on the operator's
// signature without anyone having written that down — and would go on doing so, silently, through the
// upgrade that gives the proof slot a verifier. So there is no default, and this is the test that
// keeps someone from kindly adding one.
func TestProofTypeIsNeverADefault(t *testing.T) {
	cfg := attestedTestConfig()
	cfg.ProofType = ""
	require.ErrorContains(t, cfg.Check(), "proofType is required")

	cfg.ProofType = "plonk"
	require.ErrorContains(t, cfg.Check(), "unknown proofType")

	// ...and a Config assembled in Go that skipped Check must not get a verifier by default either.
	cfg.ProofType = ""
	_, err := cfg.NewVerifier()
	require.ErrorContains(t, err, "proofType is required")
}

// TestFutureProofTypeIsRecognisedAndRefused pins the one unsupported value this build knows by name.
//
// A config asking for in-circuit proof verification is not malformed — it is ahead of the binary, and
// the two deserve different errors. An operator who reads "unknown proofType" concludes they typed
// something wrong; an operator who reads this one learns that the slot is real, that this build has
// no verifier for it, and where the trust model is written down. The generic branch is asserted
// alongside it so that this arm cannot quietly become the answer for every typo.
func TestFutureProofTypeIsRecognisedAndRefused(t *testing.T) {
	cfg := attestedTestConfig()

	cfg.ProofType = futureProofType
	err := cfg.Check()
	require.ErrorContains(t, err, "does not have")
	require.ErrorContains(t, err, "future proving upgrade")
	require.ErrorContains(t, err, "docs/TRUST-MODEL.md")
	require.ErrorContains(t, err, "attested",
		"the error must enumerate what this build DOES support, and the list is generated")
	require.NotContains(t, err.Error(), "unknown proofType",
		"a config ahead of the binary must not be reported as a malformed one")

	// No verifier exists for it either, and the failure is at the same place for the same reason.
	_, verr := cfg.NewVerifier()
	require.ErrorContains(t, verr, "does not have")

	cfg.ProofType = "groth17"
	require.ErrorContains(t, cfg.Check(), "unknown proofType")
}

// TestMockProofsIsRetired keeps the migration legible. `mockProofs: true` selected exactly the mode
// now called attested, so the error has to say that the MODE survived and the NAME did not —
// otherwise an operator reading "unknown field" concludes the bring-up path was removed.
func TestMockProofsIsRetired(t *testing.T) {
	yes := true
	cfg := attestedTestConfig()
	cfg.MockProofs = &yes

	err := cfg.Check()
	require.ErrorContains(t, err, "mockProofs is retired")
	require.ErrorContains(t, err, string(ProofTypeAttested))

	// Also refused when it AGREES with proofType: one spelling of the trust model, not two.
	require.Error(t, cfg.Check())
	no := false
	cfg.MockProofs = &no
	require.ErrorContains(t, cfg.Check(), "mockProofs is retired")
}

// attestedTestConfig is a minimal valid v1 verifier config: two addresses, three commitments, an
// anchor, and the proving system in words. Notably absent, and worth noticing: any proving-system
// key, any circuit artifact, any path to a proving toolchain.
func attestedTestConfig() *Config {
	return &Config{
		L1ChainID:        11155111,
		Submitter:        common.HexToAddress("0x11"),
		Inbox:            common.HexToAddress("0x22"),
		RollupConfigHash: common.HexToHash("0x33"),
		DepSetHash:       common.HexToHash("0x44"),
		ProofType:        ProofTypeAttested,
		Anchor: Anchor{
			OutputRoot: common.HexToHash("0x55"),
			L1Origin:   eth.BlockID{Hash: common.HexToHash("0x66"), Number: 1000},
		},
	}
}
