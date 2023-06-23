package node

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum/go-ethereum/log"
)

type Config struct {
	L1     L1EndpointSetup
	L2     L2EndpointSetup
	L2Sync L2SyncEndpointSetup

	Driver driver.Config

	Rollup rollup.Config

	// P2PSigner will be used for signing off on published content
	// if the node is sequencing and if the p2p stack is enabled
	P2PSigner p2p.SignerSetup

	RPC RPCConfig

	P2P p2p.SetupP2P

	Metrics MetricsConfig

	Pprof oppprof.CLIConfig

	// Used to poll the L1 for new finalized or safe blocks
	L1EpochPollInterval time.Duration

	ConfigPersistence ConfigPersistence

	// Optional
	Tracer    Tracer
	Heartbeat HeartbeatConfig
}

type RPCConfig struct {
	ListenAddr  string
	ListenPort  int
	EnableAdmin bool
}

func (cfg *RPCConfig) HttpEndpoint() string {
	return fmt.Sprintf("http://%s:%d", cfg.ListenAddr, cfg.ListenPort)
}

type MetricsConfig struct {
	Enabled    bool
	ListenAddr string
	ListenPort int
}

func (m MetricsConfig) Check() error {
	if !m.Enabled {
		return nil
	}

	if m.ListenPort < 0 || m.ListenPort > math.MaxUint16 {
		return errors.New("invalid metrics port")
	}

	return nil
}

type HeartbeatConfig struct {
	Enabled bool
	Moniker string
	URL     string
}

func (cfg *Config) LoadPersisted(log log.Logger) error {
	if !cfg.Driver.SequencerEnabled {
		return nil
	}
	if state, err := cfg.ConfigPersistence.SequencerState(); err != nil {
		return err
	} else if state != StateUnset {
		stopped := state == StateStopped
		if stopped != cfg.Driver.SequencerStopped {
			log.Warn(fmt.Sprintf("Overriding %v with persisted state", flags.SequencerStoppedFlag.Name), "stopped", stopped)
		}
		cfg.Driver.SequencerStopped = stopped
	} else {
		log.Info("No persisted sequencer state loaded")
	}
	return nil
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	if err := cfg.L2.Check(); err != nil {
		return fmt.Errorf("l2 endpoint config error: %w", err)
	}
	if err := cfg.L2Sync.Check(); err != nil {
		return fmt.Errorf("sync config error: %w", err)
	}
	if err := cfg.Rollup.Check(); err != nil {
		return fmt.Errorf("rollup config error: %w", err)
	}
	if err := cfg.Metrics.Check(); err != nil {
		return fmt.Errorf("metrics config error: %w", err)
	}
	if err := cfg.Pprof.Check(); err != nil {
		return fmt.Errorf("pprof config error: %w", err)
	}
	if cfg.P2P != nil {
		if err := cfg.P2P.Check(); err != nil {
			return fmt.Errorf("p2p config error: %w", err)
		}
	}
	return nil
}
