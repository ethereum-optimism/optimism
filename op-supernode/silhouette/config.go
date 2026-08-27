package silhouette

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// ProofType names the proving system a verifier requires of a batch's public values.
//
// Which one a node runs is the single largest fact about what its acceptance of a batch means, so it
// is written in the config in words, logged at startup, and reported on every accepted batch. The
// values are proving systems with different strengths — never a real one and a fake one; see
// proofTypes for the set this build supports.
type ProofType string

// ProofTypeAttested is v1's proving system: the operator's L1 transaction signature IS the proof
// (acceptance rule 1), and the proof slot on the wire MUST be empty. See AttestedVerifier and
// docs/TRUST-MODEL.md.
const ProofTypeAttested ProofType = "attested"

// futureProofType is the name of the proving system this system is designed to ratchet to, and the
// only unsupported value this build recognises. It exists so that `checkProofType` can tell an
// operator whose config is AHEAD of the binary from one whose config is wrong — see the error there.
// It is not a ProofType a verifier can be built for; there is no arm for it and no vestige of one.
const futureProofType ProofType = "groth16"

// Anchor is where a verifier's proven history starts: the output root the first accepted batch must
// extend, and the L2 block it belongs to.
type Anchor struct {
	OutputRoot  common.Hash `json:"outputRoot"`
	BlockNumber uint64      `json:"blockNumber"`
	// BlockHash is the anchor block's L2 hash. It is the parent hash the first proven block's
	// rendered batch is built against, so on a chain that starts from its own genesis it is the
	// genesis hash. Left unset it stays the zero hash rather than being invented — since wire v2
	// every other block hash in the table is real, and a fabricated one here would be the single
	// dishonest value in it.
	BlockHash common.Hash `json:"blockHash,omitempty"`
	// Timestamp is the anchor block's L2 timestamp.
	Timestamp uint64 `json:"timestamp,omitempty"`
	// L1Origin is the anchor block's L1 origin, the seed of the rendered-origin walk (G2 D4). The
	// greedy rule advances from here, so without it the first batch has no epoch to start from.
	L1Origin eth.BlockID `json:"l1Origin"`
}

// Config is a silhouette verifier's whole configuration for one proven chain.
//
// Everything here is either a binding the wire is checked against or an L1 coordinate. Notably
// absent: anything about how the chain executes. A verifier of a silhouette chain holds no state,
// no genesis allocation and no EVM — the bound on the chain is data, not software.
type Config struct {
	// L1ChainID is the settlement chain's ID, used to recover proof-batch transaction senders.
	L1ChainID uint64 `json:"l1ChainID"`
	// Submitter is the only address whose transactions are considered. Together with Inbox it is the
	// entire authenticity rule for the envelope stream: the same trust shape as a batcher inbox.
	Submitter common.Address `json:"submitter"`
	// Inbox is the L1 address proof batches are sent to.
	Inbox common.Address `json:"inbox"`
	// RollupConfigHash says WHICH chain a batch is about: without it, a valid proof of some other
	// chain's derivation would be accepted as this one's history.
	RollupConfigHash common.Hash `json:"rollupConfigHash"`
	// DepSetHash is the dependency set the batch's interop attributes were built from. A batch
	// derived under a different dep set is a different chain's history.
	DepSetHash common.Hash `json:"depSetHash"`
	// L1StartBlock is retained for standalone proof-source walkers such as the magic EL service.
	// The supernode verifier's stock derivation pipeline starts from the rollup genesis instead.
	L1StartBlock uint64 `json:"l1StartBlock"`
	// L1HeadMaxDepth bounds how far below the L1 block that carried a batch its claimed l1Head may
	// sit.
	L1HeadMaxDepth uint64 `json:"l1HeadMaxDepth"`
	// Anchor is the genesis of proven history.
	Anchor Anchor `json:"anchor"`
	// ExportPolicyHash is the export filter the prover must be committing to. It is checked because
	// it is what says whether a log missing from a batch is a log the chain never emitted or a log
	// the policy excluded — a verifier that disagreed with the prover about the filter would
	// silently treat exclusions as absences.
	ExportPolicyHash *common.Hash `json:"exportPolicyHash,omitempty"`
	// ProofType is the proving system this verifier requires of every batch. It is the trust model,
	// stated in one word, and it has NO DEFAULT: a config that does not say gets an error rather than
	// an assumption, because a node meant to check something and a node that checks nothing must never
	// be one unstated field apart.
	//
	// This build has exactly one: `attested`, which requires an empty proof slot and rests the chain's
	// validity on the operator's L1 signature (acceptance rule 1). It is a field rather than an
	// implied constant because the proving system is the single most consequential fact about an
	// accepted batch, and because it is the slot a stronger proving system upgrades into without
	// changing the wire or any other acceptance rule. See docs/TRUST-MODEL.md.
	ProofType ProofType `json:"proofType"`
	// MockProofs is the RETIRED spelling of `proofType: attested`. It is still decoded so that the
	// error can say what to write instead — dropping the field would surface as "unknown field
	// mockProofs" from the JSON decoder, which tells an operator that the key is wrong but not that
	// the mode it names is alive and renamed.
	//
	// It is an error rather than a silent alias on purpose. Two spellings for the trust model is
	// exactly the ambiguity the rename exists to remove, and the worse spelling is the one that says
	// "mock" about a mode that is now the real, shipped v1 proving system (V1G D1).
	MockProofs *bool `json:"mockProofs,omitempty"`
	// WireVersion is the proof-batch envelope version this verifier accepts. EXACTLY one, and zero
	// means the codec's current version.
	//
	// One version rather than a set, and the reason is the acceptance rule. A verifier's config pins
	// the version it accepts, and that version decides whether this node
	// CHECKS a proven chain's dependencies (v3, the import list is on the wire) or TRUSTS them (v2,
	// the wire says nothing about imports). A node that accepted both would silently apply the weaker
	// posture to a chain whose operator believes it is running the stronger one — the failure mode
	// there is not "a batch was rejected", it is "nothing was checked and everything looked fine".
	//
	// So a v2→v3 rotation is TWO configured verifiers, each strict, not one lenient one. That is what
	// makes a dark launch a comparison rather than a merge: the same L1, two inboxes, two configs, two
	// wire versions, and every value on both sides diffable.
	WireVersion uint8 `json:"wireVersion,omitempty"`
	// L1ChainConfigPath supplies the settlement chain's config for an L1 that is not one of the
	// public networks — a devnet, or a local cluster.
	//
	// It is a path rather than a fallback default because of WHAT the value is read for: whether
	// the L1 block's excess blob gas is priced under Cancun or Prague, for the L1-info
	// transaction's blob-base-fee field. Guessing that would put a fabricated number in a
	// consensus-relevant transaction, so an unknown L1 is an error and this is how an operator
	// answers it, explicitly.
	L1ChainConfigPath string `json:"l1ChainConfigPath,omitempty"`
}

// DefaultL1HeadMaxDepth is a day of Sepolia blocks: an l1Head older than this is not a batch that is
// merely late, it is a batch derived against a view of L1 the node has long since passed.
const DefaultL1HeadMaxDepth = 7200

// LoadConfig reads and validates a silhouette verifier config file.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read silhouette config %q: %w", path, err)
	}
	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse silhouette config %q: %w", path, err)
	}
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("invalid silhouette config %q: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) Check() error {
	switch {
	case c.L1ChainID == 0:
		return errors.New("l1ChainID is required")
	case c.Submitter == (common.Address{}):
		return errors.New("submitter is required")
	case c.Inbox == (common.Address{}):
		return errors.New("inbox is required")
	case c.Submitter == c.Inbox:
		return errors.New("submitter and inbox must differ")
	case c.RollupConfigHash == (common.Hash{}):
		return errors.New("rollupConfigHash is required")
	case c.DepSetHash == (common.Hash{}):
		return errors.New("depSetHash is required")
	case c.Anchor.OutputRoot == (common.Hash{}):
		return errors.New("anchor.outputRoot is required")
	case c.Anchor.L1Origin == (eth.BlockID{}):
		return errors.New("anchor.l1Origin is required: it seeds the rendered-origin walk")
	}
	if err := c.checkProofType(); err != nil {
		return err
	}
	// A wire version this codec cannot read is refused HERE, at load, rather than at the first blob.
	// The difference matters operationally: a config error surfaces as a node that will not start,
	// while the same value reaching the decode path surfaces as a node that started, looks healthy,
	// and rejects every batch forever.
	if err := proofbatch.CheckVersion(c.wireVersion()); err != nil {
		return fmt.Errorf("wireVersion: %w", err)
	}
	return nil
}

// EffectiveWireVersion is the envelope version this verifier actually accepts, with the default
// resolved. Log THIS, never the raw field: a config that omits `wireVersion` means "the current
// version", and printing the raw 0 beside `dependenciesVerified=true` in the startup line an operator
// is told to read would be a number that is not a wire version.
func (c *Config) EffectiveWireVersion() uint8 { return c.wireVersion() }

// wireVersion is the envelope version this verifier accepts, defaulting to the codec's current one.
func (c *Config) wireVersion() uint8 {
	if c.WireVersion == 0 {
		return proofbatch.Version
	}
	return c.WireVersion
}

// DependenciesVerified reports whether this configuration puts a proven chain's cross-chain
// dependencies under the cross-safety judge, or leaves them proof-trusted.
//
// It is a method rather than a comment because it is the one thing about a silhouette verifier an
// operator cannot see from the outside: both postures derive the chain, serve the same roots and
// report the same heads. Only this decides whether the chain's IMPORTS were ever checked.
func (c *Config) DependenciesVerified() bool {
	return proofbatch.VersionHasExecMsgs(c.wireVersion())
}

// ExportPolicy is the export-filter commitment a batch must carry.
func (c *Config) ExportPolicy() common.Hash {
	if c.ExportPolicyHash != nil {
		return *c.ExportPolicyHash
	}
	return proofbatch.ExportPolicyAllHashes
}

func (c *Config) l1HeadMaxDepth() uint64 {
	if c.L1HeadMaxDepth == 0 {
		return DefaultL1HeadMaxDepth
	}
	return c.L1HeadMaxDepth
}

// proofTypes is every proving system this build supports. Error messages enumerate it rather than
// listing values inline, so that a build with one proving system says so and offers nothing it cannot
// do.
var proofTypes = []ProofType{
	ProofTypeAttested,
}

func proofTypeList() string {
	names := make([]string, len(proofTypes))
	for i, pt := range proofTypes {
		names[i] = string(pt)
	}
	return strings.Join(names, " | ")
}

// checkProofType validates the proving system, and is where the retired `mockProofs` spelling is
// refused with instructions.
func (c *Config) checkProofType() error {
	if c.MockProofs != nil {
		// Deliberately does not translate `false` for the operator. `mockProofs: true` is the case that
		// exists in the wild and the case worth being unambiguous about; `false` meant "verify proofs",
		// which is a proving system this build may or may not have, and proofTypeList is the honest
		// answer to that.
		return fmt.Errorf("mockProofs is retired: name the proving system in words instead — "+
			"proofType: %s. The mode it selected did not change and was never a stub: "+
			"`mockProofs: true` is `proofType: %s`, v1's real proving system, and it is called that "+
			"because that is what it is (see docs/TRUST-MODEL.md)", proofTypeList(), ProofTypeAttested)
	}
	if c.ProofType == "" {
		return fmt.Errorf("proofType is required (%s): the trust model is never a default", proofTypeList())
	}
	if c.ProofType == futureProofType {
		// Recognised by name and refused, because this config is not malformed — it is ahead of the
		// binary, and those two deserve different errors. The slot it asks for is real: the wire
		// carries a proof field, acceptance rule 5 is the rung that reads it, and every other rule is
		// identical either way. What is missing is a verifier.
		return fmt.Errorf("proofType %q names a proving system this build does not have: v1 settles "+
			"chain validity on the operator's attestation, and in-circuit proof verification arrives "+
			"with a future proving upgrade — the wire, the acceptance rules and this config all keep "+
			"the slot for it (see docs/TRUST-MODEL.md). This build supports %s",
			c.ProofType, proofTypeList())
	}
	if !slices.Contains(proofTypes, c.ProofType) {
		return fmt.Errorf("unknown proofType %q: this build supports %s", c.ProofType, proofTypeList())
	}
	return nil
}

// Attested reports whether this verifier rests a batch's validity on the operator's attestation
// rather than on a proof of the derivation. It is the trust model in one boolean; docs/TRUST-MODEL.md is
// the same thing in prose.
func (c *Config) Attested() bool { return c.ProofType == ProofTypeAttested }

// NewVerifier builds the proof verifier this config asks for.
func (c *Config) NewVerifier() (Verifier, error) {
	switch c.ProofType {
	case ProofTypeAttested:
		return AttestedVerifier{}, nil
	default:
		// Unreachable through LoadConfig, which calls Check first. Reached only by a Config built in
		// Go that skipped it, and an unset ProofType must not become a proving system by default.
		return nil, fmt.Errorf("cannot build a verifier: %w", c.checkProofType())
	}
}
