package interop

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/managed"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type Config struct {
	// RPCAddr address to bind RPC server to, to serve external supervisor nodes.
	// Optional. This will soon be required: running op-node without supervisor is being deprecated.
	RPCAddr string
	// RPCPort port to bind RPC server to, to serve external supervisor nodes.
	// Binds to any available port if set to 0.
	RPCPort int
	// RPCJwtSecretPath path of JWT secret file to apply authentication to the interop server address.
	RPCJwtSecretPath string
}

func (cfg *Config) Check() error {
	if cfg.RPCAddr != "" && cfg.RPCJwtSecretPath == "" {
		return errors.New("interop RPC server requires JWT setup, but no JWT path was specified")
	}
	return nil
}

// Setup creates an interop sub-system. This drives the node syncing.
// If setup returns a nil system (without error) the node should fall back to legacy mode.
func (cfg *Config) Setup(ctx context.Context, logger log.Logger, rollupCfg *rollup.Config, l1 L1Source, l2 L2Source, m opmetrics.RPCMetricer) (SubSystem, error) {
	if cfg.RPCAddr == "" {
		logger.Warn("No interop RPC configured, falling back to legacy sync mode.")
		return nil, nil // a `nil` system will result in legacy mode.
	}
	logger.Info("Setting up Interop RPC server to serve supervisor sync work")
	// Load JWT secret, if any, generate one otherwise.
	jwtSecret, err := rpc.ObtainJWTSecret(logger, cfg.RPCJwtSecretPath, true)
	if err != nil {
		return nil, err
	}
	return managed.NewManagedMode(logger, rollupCfg, cfg.RPCAddr, cfg.RPCPort, jwtSecret, l1, l2, m), nil
}
