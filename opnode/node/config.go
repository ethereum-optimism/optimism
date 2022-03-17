package node

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
)

type Config struct {
	// L1 and L2 nodes
	L1NodeAddr    string   // Address of L1 User JSON-RPC endpoint to use (eth namespace required)
	L2EngineAddrs []string // Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)

	Rollup rollup.Config

	// Sequencer flag, enables sequencing
	Sequencer bool

	// SubmitterPrivKey, temporary config var while the batch-submitter is part of the rollup node
	SubmitterPrivKey *ecdsa.PrivateKey
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	if err := cfg.Rollup.Check(); err != nil {
		return fmt.Errorf("rollup config error: %v", err)
	}

	return nil
}
