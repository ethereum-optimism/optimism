package node

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum-optimism/optimistic-specs/opnode/p2p"

	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
)

type Config struct {
	// L1 and L2 nodes
	L1NodeAddr    string   // Address of L1 User JSON-RPC endpoint to use (eth namespace required)
	L2EngineAddrs []string // Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)
	L2NodeAddr    string   // Address of L2 User JSON-RPC endpoint to use (eth namespace required)

	// L1TrustRPC: if we trust the L1 RPC we do not have to validate L1 response contents like headers
	// against block hashes, or cached transaction sender addresses.
	// Thus we can sync faster at the risk of the source RPC being wrong.
	L1TrustRPC bool

	Rollup rollup.Config

	// Sequencer flag, enables sequencing
	Sequencer bool

	// SubmitterPrivKey, temporary config var while the batch-submitter is part of the rollup node
	SubmitterPrivKey *ecdsa.PrivateKey

	RPC RPCConfig

	P2P p2p.SetupP2P
}

type RPCConfig struct {
	ListenAddr string
	ListenPort int
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	if err := cfg.Rollup.Check(); err != nil {
		return fmt.Errorf("rollup config error: %v", err)
	}
	if cfg.P2P != nil {
		if err := cfg.P2P.Check(); err != nil {
			return fmt.Errorf("p2p config error: %v", err)
		}
	}
	return nil
}
