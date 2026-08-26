package silhouette

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Manifest is the supernode-level declaration of which chains are silhouette chains.
//
// It is a separate file rather than a set of per-chain CLI flags because the thing being declared
// is not a tuning knob: it says that a chain in the supernode's chains map has no execution client
// and no batcher, and that its history arrives as proofs. That is a statement about the deployment,
// and a deployment statement belongs in a file an operator can diff and a runbook can name.
//
// The supernode reads it once at startup. A chain absent from the manifest is an ordinary driven
// chain and nothing about its construction changes — which is the property that makes this switch
// safe to carry in a binary that also runs normal clusters.
type Manifest struct {
	Chains []ManifestChain `json:"chains"`
}

// ManifestChain declares one silhouette chain.
type ManifestChain struct {
	// ChainID must also appear in the supernode's --chains list. A manifest entry for a chain the
	// supernode is not running is a configuration mistake rather than a no-op, and is rejected:
	// the likely intent was to run it.
	ChainID uint64 `json:"chainID"`
	// VerifierConfig is the path to this chain's silhouette verifier Config — the wire bindings,
	// the anchor, the proving system. Relative paths resolve against the manifest's own directory, so a
	// config directory can be copied to a VM as a unit.
	VerifierConfig string `json:"verifierConfig"`
	// Labels selects the posture: "derivation" (a verifier, which derives P itself) or
	// "proven-head" (a sequencer-side node, which fronts P's real execution client and must take
	// its PUBLIC labels from the proven head — see Container.OptimisticAt).
	//
	// Empty means "derivation". The verifier posture is the one a public node runs, and defaulting
	// to it means a missing field cannot silently turn a verifier into a node that labels P from
	// facts it has not derived.
	Labels string `json:"labels"`

	// resolved holds the loaded verifier config, filled by LoadManifest.
	resolved *Config
}

// Config returns the loaded verifier config for this chain.
func (m ManifestChain) Config() *Config { return m.resolved }

// LabelSource resolves the declared posture.
func (m ManifestChain) LabelSource() (LabelSource, error) {
	switch m.Labels {
	case "", "derivation":
		return LabelsFromDerivation, nil
	case "proven-head":
		return LabelsFromProvenHead, nil
	default:
		return 0, fmt.Errorf("unknown labels posture %q: want \"derivation\" or \"proven-head\"", m.Labels)
	}
}

// LoadManifest reads a silhouette manifest and every verifier config it names.
//
// Both halves are loaded here, at startup, rather than lazily at chain construction: a bad vkey
// path or an unparseable config is a reason not to start, and a supernode that discovered it while
// bringing up its third chain would leave the first two running against a cluster it cannot join.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read silhouette manifest %q: %w", path, err)
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse silhouette manifest %q: %w", path, err)
	}
	if len(m.Chains) == 0 {
		return nil, fmt.Errorf("silhouette manifest %q declares no chains", path)
	}
	dir := filepath.Dir(path)
	seen := make(map[uint64]struct{}, len(m.Chains))
	for i := range m.Chains {
		c := &m.Chains[i]
		if c.ChainID == 0 {
			return nil, fmt.Errorf("silhouette manifest %q: entry %d has no chainID", path, i)
		}
		if _, dup := seen[c.ChainID]; dup {
			return nil, fmt.Errorf("silhouette manifest %q: chain %d declared twice", path, c.ChainID)
		}
		seen[c.ChainID] = struct{}{}
		if _, err := c.LabelSource(); err != nil {
			return nil, fmt.Errorf("silhouette manifest %q: chain %d: %w", path, c.ChainID, err)
		}
		if c.VerifierConfig == "" {
			return nil, fmt.Errorf("silhouette manifest %q: chain %d has no verifierConfig", path, c.ChainID)
		}
		p := c.VerifierConfig
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		cfg, err := LoadConfig(p)
		if err != nil {
			return nil, fmt.Errorf("silhouette manifest %q: chain %d: %w", path, c.ChainID, err)
		}
		c.resolved = cfg
	}
	return &m, nil
}

// Lookup returns the declaration for a chain, if it has one.
func (m *Manifest) Lookup(id eth.ChainID) (ManifestChain, bool) {
	if m == nil {
		return ManifestChain{}, false
	}
	for _, c := range m.Chains {
		if eth.ChainIDFromUInt64(c.ChainID) == id {
			return c, true
		}
	}
	return ManifestChain{}, false
}

// CheckChains rejects a manifest entry for a chain the supernode is not running.
func (m *Manifest) CheckChains(running []uint64) error {
	if m == nil {
		return nil
	}
	have := make(map[uint64]struct{}, len(running))
	for _, id := range running {
		have[id] = struct{}{}
	}
	for _, c := range m.Chains {
		if _, ok := have[c.ChainID]; !ok {
			return fmt.Errorf("silhouette manifest declares chain %d, which is not in --chains", c.ChainID)
		}
	}
	return nil
}

// L1ChainConfig resolves the settlement chain's config.
//
// The forced-extension convention reads exactly one thing from it — whether the L1 block's excess
// blob gas is priced under Cancun or Prague, for the L1-info transaction's blob-base-fee field. For
// a public network that is a lookup of a public constant, and looking it up beats letting an
// operator supply a value they could get wrong.
//
// For any other L1 — a devnet, a local cluster — there is no constant to look up, and guessing
// would put a fabricated number in a consensus-relevant transaction. So an unknown chain ID is an
// error unless the config names a file, which is the operator saying explicitly what this L1 is.
// Refusing to start beats starting with an invented fee schedule.
func L1ChainConfig(cfg *Config) (*params.ChainConfig, error) {
	if cfg.L1ChainConfigPath != "" {
		raw, err := os.ReadFile(cfg.L1ChainConfigPath)
		if err != nil {
			return nil, fmt.Errorf("read L1 chain config %q: %w", cfg.L1ChainConfigPath, err)
		}
		var c params.ChainConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("parse L1 chain config %q: %w", cfg.L1ChainConfigPath, err)
		}
		if c.ChainID == nil || c.ChainID.Uint64() != cfg.L1ChainID {
			return nil, fmt.Errorf("L1 chain config %q is for chain %v, but this chain settles on %d",
				cfg.L1ChainConfigPath, c.ChainID, cfg.L1ChainID)
		}
		return &c, nil
	}
	for _, c := range []*params.ChainConfig{
		params.MainnetChainConfig,
		params.SepoliaChainConfig,
		params.HoleskyChainConfig,
		params.HoodiChainConfig,
	} {
		if c.ChainID != nil && c.ChainID.Uint64() == cfg.L1ChainID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no known L1 chain config for chain ID %d: "+
		"set l1ChainConfigPath in the verifier config to name it explicitly", cfg.L1ChainID)
}
