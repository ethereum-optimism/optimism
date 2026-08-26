package silhouette

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// aProvingSystem stands in for ANY proof-verifying verifier: it accepts a batch that carries proof
// bytes and rejects one that does not. What it would verify is deliberately beside the point — the
// claim under test below is that the answer never reaches the judge's inputs, so a double that
// answers "yes" is exactly the right instrument.
//
// It is injected at the seam Config.NewVerifier resolves to, which is the seam the real proving
// system occupies.
type aProvingSystem struct{}

func (aProvingSystem) Verify(_ []byte, proof []byte) error {
	if len(proof) == 0 {
		return errors.New("a proving system requires a proof")
	}
	return nil
}

func (aProvingSystem) ProofType() ProofType { return "a-proving-system" }

// withVerifier rebuilds the environment's data source around a different proving system, leaving the
// L1, the blobs, the fact store and every other acceptance rule alone.
func (e *testEnv) withVerifier(v Verifier) *testEnv {
	e.src = NewDataSource(testlog.Logger(e.t, 3), e.cfg, e.rollup, params.SepoliaChainConfig,
		e.sysCfg, e.l1, e.blobs, v, e.facts)
	return e
}

// TestTheJudgeReadsTheWireNotTheProof is v1's licence to keep wire v3 and the judge flip switched ON
// while the proving system is attestation.
//
// The worry worth ruling out: G7's cross-safety flip was built and gated in a world where P's batches
// were expected to carry proofs, and if any part of that path quietly treated "the proof verified" as
// a precondition for trusting the import list, then attested mode would be inheriting a check that no
// longer means what it meant. It would still LOOK identical — same heads, same roots, same
// dependenciesVerified=true — because the failure mode of a vacuous check is that everything is fine.
//
// It does not, and the reason is structural: the import list is WIRE DATA. It is decoded from the
// envelope, recorded against the wire VERSION (Fact.ExecMsgsKnown), and handed to the stock judge.
// The verifier is one acceptance rule beside four others and contributes no input to any of it. So a
// lying attester's claimed imports are checked against real chain A exactly as a prover's would be —
// which is the honest half of v1's trust model and worth having an assertion behind.
//
// The assertion is EQUALITY, over two runs that differ in nothing but the proving system: one batch
// accepted on the operator's signature with an empty proof slot, one batch accepted by a proving
// system because it carried a proof. Every judge-facing fact must be identical.
func TestTheJudgeReadsTheWireNotTheProof(t *testing.T) {
	imports := []proofbatch.ExecMsg{
		anImport(424246, 5, 0, l1GenesisT),
		anImport(424248, 9, 2, l1GenesisT),
	}

	// Attested: the proof slot is empty, and rule 1's tx.from == submitterEOA is the whole authority.
	attested := newTestEnv(t, l1GenesisNum+10)
	aspec := attested.goodBatch()
	aspec.imports = imports
	attested.plantSpec(aspec)
	require.NotEmpty(t, attested.derive(aspec.carrier))

	// A proving system: the same batch, with a proof, accepted because the proof checked out.
	proven := newTestEnv(t, l1GenesisNum+10).withVerifier(aProvingSystem{})
	pspec := proven.goodBatch()
	pspec.imports = imports
	pspec.proof = []byte{0x01, 0x02, 0x03, 0x04}
	proven.plantSpec(pspec)
	require.NotEmpty(t, proven.derive(pspec.carrier),
		"the control arm must ACCEPT: an equality assertion between two rejections proves nothing")

	aFact, ok := attested.facts.ByNumber(1)
	require.True(t, ok, "the attested batch was not accepted")
	pFact, ok := proven.facts.ByNumber(1)
	require.True(t, ok, "the proven batch was not accepted")

	// The whole fact, not a chosen field: anything the judge or the frontier reads about this block
	// must be a function of the wire and this node's L1 view, never of who vouched for it.
	require.Equal(t, aFact, pFact,
		"a block's recorded facts must not depend on the proving system that admitted its batch")

	// Non-vacuous controls. Equality over two empty import lists, or over two blocks that were never
	// under the judge, would be true of a verifier that checks nothing.
	require.True(t, aFact.ExecMsgsKnown,
		"the import list must be KNOWN under attestation: unknown is the pre-v3 posture, where the "+
			"judge validates nothing and says it validates everything")
	require.Len(t, aFact.ExecMsgs, 2)
	require.True(t, attested.cfg.DependenciesVerified(),
		"wire v3 puts P's dependencies under the judge, and the proving system has no say in it")

	// And the two arms really did run different proving systems, which is what makes the equality
	// above a finding rather than a tautology.
	require.Equal(t, ProofTypeAttested, attested.src.verifier.ProofType())
	require.NotEqual(t, ProofTypeAttested, proven.src.verifier.ProofType())
}

// TestAttestedModeRefusesProofBytesAtTheAcceptancePath is the same rule as TestAttestedVerifier, one
// layer out: through the real acceptance path, on a planted L1, with the batch otherwise perfect.
//
// Worth its own test because the unit above proves what AttestedVerifier returns, and this proves the
// return value is WIRED — that a proof-carrying batch does not move the proven head.
func TestAttestedModeRefusesProofBytesAtTheAcceptancePath(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+10)
	require.Equal(t, ProofTypeAttested, e.cfg.ProofType)

	spec := e.goodBatch()
	spec.proof = []byte{0xde, 0xad, 0xbe, 0xef}
	e.plantSpec(spec)
	require.Empty(t, e.derive(spec.carrier),
		"a batch carrying proof bytes must be refused in attested mode, not accepted with the bytes ignored")
	_, ok := e.facts.Head()
	require.False(t, ok, "the refused batch must leave the proven head where it was")

	// ...and the chain is not poisoned by the refusal: the same blocks, posted correctly, are accepted.
	// Reject-and-log means a bad batch is skipped, never a chain that stops.
	good := e.goodBatch()
	good.carrier = spec.carrier + 2
	e.plantSpec(good)
	require.NotEmpty(t, e.derive(good.carrier))
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.Equal(t, good.firstBlock+uint64(good.count)-1, head.Number,
		"the re-posted batch must land whole, from the same chaining point the refusal left alone")
}

// TestAttestedChainIsRenderedWithoutAProvingToolchain is the v1 pitch as an assertion.
//
// The claim is that a verifier of an attested silhouette chain holds no proving-system artefact of
// any kind: no proving-system key, no circuit artefact, no path to one. It is easy to believe and
// easy to lose — a single config field added "so the flip is ready" would make every v1 deployment
// carry a proving-system key it does not use and cannot check, and nothing would fail.
func TestAttestedChainIsRenderedWithoutAProvingToolchain(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+10)
	require.NoError(t, e.cfg.Check())

	verifier, err := e.cfg.NewVerifier()
	require.NoError(t, err)
	require.IsType(t, AttestedVerifier{}, verifier)

	// And the chain renders: blocks, roots, exported logs, an L1-info transaction per block. The whole
	// system, with the proving system's slot on the wire empty.
	spec := e.goodBatch()
	e.plantSpec(spec)
	frames := e.derive(spec.carrier)
	require.NotEmpty(t, frames)
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.NotEqual(t, types.EmptyRootHash, head.OutputRoot)
	require.False(t, head.Forced)
}
