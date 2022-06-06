package node

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/p2p"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type Config struct {
	L1 L1EndpointSetup
	L2 L2EndpointsSetup

	Rollup rollup.Config

	// Sequencer flag, enables sequencing
	Sequencer bool

	// P2PSigner will be used for signing off on published content
	// if the node is sequencing and if the p2p stack is enabled
	P2PSigner p2p.SignerSetup

	RPC RPCConfig

	P2P p2p.SetupP2P

	// Optional
	Tracer Tracer
}

type RPCConfig struct {
	ListenAddr string
	ListenPort int
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	if err := cfg.L2.Check(); err != nil {
		return fmt.Errorf("l2 endpoint config error: %v", err)
	}
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
