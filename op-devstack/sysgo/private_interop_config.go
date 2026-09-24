package sysgo

import (
	"errors"
	"fmt"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum/go-ethereum/common"
)

// Proof profiles a devstack pair can run (op-private-interop/docs/BATCHES.md). Every one except
// PrivateInteropSP1 with a real prover is test-gated: accepted only by test binaries on the
// devstack chain IDs.
const (
	// PrivateInteropStub is insecure-stub-v1, the default: dummy proof bytes, no producer.
	PrivateInteropStub = "stub"
	// PrivateInteropExecutionMock is execution-mock-v1: the producer runs the native relation and
	// emits the legacy, forgeable admission-digest envelope.
	PrivateInteropExecutionMock = "execution-mock"
	// PrivateInteropSP1 is sp1-private-projection-v1, the sound profile.
	PrivateInteropSP1 = "sp1"
)

// PrivateInteropNativeMockVKey is the program vkey the devstack pins when no private-projection ELF
// is available and the producer runs in native-mock mode. It is a placeholder, not any program's
// key; with an ELF present the devstack pins the ELF's real vkey instead.
var PrivateInteropNativeMockVKey = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

// PrivateInteropConfig is the devstack's description of a private-interop chain pair.
//
// A pair is two chains sharing ONE chain ID: the PRIVATE half, which is what tests send
// transactions to, and its public RENDERING, which is what the supernode judges and what every
// counterparty means when it names the chain. See op-private-interop/docs/DESIGN.md.
//
// Everything here has a working default, because the point of the preset is that an ordinary
// interop test does not have to know it is running against a pair.
type PrivateInteropConfig struct {
	// ProofCommand is the producer the batcher invokes (kona-sp1-private-projection-executor). It
	// is required by the execution-mock and sp1 profiles.
	ProofCommand string
	// Profile selects the projection verifier: PrivateInteropStub (default),
	// PrivateInteropExecutionMock or PrivateInteropSP1.
	Profile string
	// SP1Prover is the producer's prover under PrivateInteropSP1: "mock" or "native-mock" (mock
	// envelopes; mock_proofs is set), or "cpu" / "network" (real Groth16; mock_proofs is not set).
	// Empty under PrivateInteropSP1 means "mock" when $KONA_SP1_ELF_DIR holds the
	// private-projection ELF and "native-mock" otherwise.
	SP1Prover string
	// TestHooks are the batcher's programmatic test hooks (never a flag). Nil in every ordinary
	// test.
	TestHooks *bss.PrivateInteropTestHooks
	// MaxBlocksPerRange is the builder's cadence: how many private blocks one span batch, and one
	// range claim, covers.
	//
	// Production is ~300 blocks (ten minutes at 2 s). A devstack wants the opposite trade: the
	// rendering only advances when a range lands, and a test that waits ten minutes for its
	// message to become public is a test nobody runs. Small enough that the rendering tracks the
	// private chain closely, large enough that a range is a range.
	MaxBlocksPerRange uint64

	// SkipRenderingInvariant opts out of the standing rendering-invariant checker.
	//
	// The checker is on by default (the `NewSingleChainMultiNode` / `...WithoutCheck` precedent):
	// a background assertion that costs nothing when the system is healthy is an assertion that
	// should not need opting into. Tests that deliberately break the correspondence -- by stopping
	// the builder, by diverging the private chain from a landed claim -- turn it off.
	SkipRenderingInvariant bool
}

// PrivateInteropOption mutates a pair's configuration.
type PrivateInteropOption func(cfg *PrivateInteropConfig)

// DefaultPrivateInteropConfig is the devstack pair every preset gets unless a test says otherwise.
func DefaultPrivateInteropConfig() PrivateInteropConfig {
	return PrivateInteropConfig{
		// Twelve blocks can amortize L1 inclusion and follow polling during
		// recovery; four-block ranges can remain at the expiry frontier.
		MaxBlocksPerRange: 12,
	}
}

// WithPrivateInteropCadence sets how many private blocks one range covers.
func WithPrivateInteropCadence(blocks uint64) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) { cfg.MaxBlocksPerRange = blocks }
}

// WithoutRenderingInvariantCheck turns off the standing rendering-invariant checker for tests that
// deliberately break block-for-block correspondence.
func WithoutRenderingInvariantCheck() PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) { cfg.SkipRenderingInvariant = true }
}

// Check validates the pair's configuration.
func (c *PrivateInteropConfig) Check() error {
	if c.MaxBlocksPerRange == 0 {
		return errors.New("private interop: the range cadence must be at least one block")
	}
	switch c.profile() {
	case PrivateInteropStub:
		if c.ProofCommand != "" || c.SP1Prover != "" {
			return errors.New("private interop: the stub profile takes no proof command or prover")
		}
	case PrivateInteropExecutionMock:
		if c.ProofCommand == "" || c.SP1Prover != "" {
			return errors.New("private interop: the execution-mock profile needs a proof command and no sp1 prover")
		}
	case PrivateInteropSP1:
		if c.ProofCommand == "" {
			return errors.New("private interop: the sp1 profile needs a proof command")
		}
		switch c.SP1Prover {
		case "", "mock", "native-mock", "cpu", "network":
		default:
			return fmt.Errorf("private interop: unknown sp1 prover %q", c.SP1Prover)
		}
	default:
		return fmt.Errorf("private interop: unknown proof profile %q", c.Profile)
	}
	return nil
}

// profile resolves the legacy shorthand: a proof command without a profile is execution-mock.
func (c *PrivateInteropConfig) profile() string {
	switch {
	case c.Profile != "":
		return c.Profile
	case c.ProofCommand != "":
		return PrivateInteropExecutionMock
	default:
		return PrivateInteropStub
	}
}

// WithPrivateInteropExecutionMock exercises native execution before publication under the legacy,
// test-only execution-mock-v1 verifier.
func WithPrivateInteropExecutionMock(command string) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) {
		cfg.ProofCommand, cfg.Profile, cfg.SP1Prover = command, PrivateInteropExecutionMock, ""
	}
}

// WithPrivateInteropSP1Mock runs the sound profile, sp1-private-projection-v1, with SP1 mock
// envelopes (mock_proofs=true, test-gated). The producer uses the SP1 mock prover when
// $KONA_SP1_ELF_DIR holds the private-projection ELF, pinning the ELF's vkey, and the native-mock
// mode with PrivateInteropNativeMockVKey otherwise.
func WithPrivateInteropSP1Mock(command string) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) {
		cfg.ProofCommand, cfg.Profile, cfg.SP1Prover = command, PrivateInteropSP1, ""
	}
}

// WithPrivateInteropSP1Prover runs the sound profile with a real Groth16 prover ("cpu" or
// "network") and mock_proofs=false. It requires the private-projection ELF.
func WithPrivateInteropSP1Prover(command, prover string) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) {
		cfg.ProofCommand, cfg.Profile, cfg.SP1Prover = command, PrivateInteropSP1, prover
	}
}

// WithPrivateInteropTestHooks installs the batcher's programmatic test hooks. The hooks are shared
// by pointer, so a test may change behaviour at runtime through fields it controls (for example a
// MutateProof closure over an atomic flag).
func WithPrivateInteropTestHooks(hooks *bss.PrivateInteropTestHooks) PrivateInteropOption {
	return func(cfg *PrivateInteropConfig) { cfg.TestHooks = hooks }
}
