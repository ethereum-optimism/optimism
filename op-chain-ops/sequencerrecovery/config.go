// Package sequencerrecovery checks the health of an op-conductor sequencer
// cluster on an interop chain and restarts sequencing on the canonical chain
// after a reorg.
//
// An interop reorg replaces a block that holds an invalid executing message
// with a deposit-only block. The sequencers' op-nodes follow the replacement,
// but op-conductor's raft log still holds the latest payload committed before
// the reorg. The leader then refuses to start sequencing because its op-node's
// unsafe head no longer matches that payload, and the chain stalls. Recovery
// pauses the conductors, starts the leader's op-node from its own unsafe head
// (which must descend from the supernode's safe head), waits for the new block
// to overwrite the stale raft entry, and resumes the conductors.
package sequencerrecovery

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Sequencer is one member of a conductor cluster.
type Sequencer struct {
	ID           string
	RaftAddr     string
	ConductorURL string
	NodeURL      string
	Voting       bool
}

// Config is one network from an op-conductor-ops config file, the file netchef
// renders to scripts/op-conductor-ops/<chain>.toml.
type Config struct {
	Network    string
	Sequencers []Sequencer
	// CertPath is the CA bundle for the RPC endpoints, resolved relative to the
	// config file. Empty means the system roots.
	CertPath string
}

type fileSequencer struct {
	RaftAddr        string `toml:"raft_addr"`
	ConductorRPCURL string `toml:"conductor_rpc_url"`
	NodeRPCURL      string `toml:"node_rpc_url"`
	Voting          *bool  `toml:"voting"`
}

type fileNetwork struct {
	Sequencers []string `toml:"sequencers"`
}

type file struct {
	CertPath   string                   `toml:"cert_path"`
	Networks   map[string]fileNetwork   `toml:"networks"`
	Sequencers map[string]fileSequencer `toml:"sequencers"`
}

// LoadConfig reads network from an op-conductor-ops config file.
func LoadConfig(path, network string) (*Config, error) {
	var f file
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return parseConfig(f, filepath.Dir(path), network)
}

func parseConfig(f file, baseDir, network string) (*Config, error) {
	nw, ok := f.Networks[network]
	if !ok {
		return nil, fmt.Errorf("network %q not in config", network)
	}
	if len(nw.Sequencers) == 0 {
		return nil, fmt.Errorf("network %q has no sequencers", network)
	}
	cfg := &Config{Network: network}
	if f.CertPath != "" {
		cfg.CertPath = f.CertPath
		if !filepath.IsAbs(cfg.CertPath) {
			cfg.CertPath = filepath.Join(baseDir, cfg.CertPath)
		}
	}
	for _, id := range nw.Sequencers {
		s, ok := f.Sequencers[id]
		if !ok {
			return nil, fmt.Errorf("sequencer %q of network %q not in config", id, network)
		}
		if s.ConductorRPCURL == "" || s.NodeRPCURL == "" || s.RaftAddr == "" {
			return nil, fmt.Errorf("sequencer %q: raft_addr, conductor_rpc_url and node_rpc_url are required", id)
		}
		voting := true
		if s.Voting != nil {
			voting = *s.Voting
		}
		cfg.Sequencers = append(cfg.Sequencers, Sequencer{
			ID:           id,
			RaftAddr:     s.RaftAddr,
			ConductorURL: s.ConductorRPCURL,
			NodeURL:      s.NodeRPCURL,
			Voting:       voting,
		})
	}
	return cfg, nil
}
