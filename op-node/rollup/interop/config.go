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
	// Cannot be set if SupervisorAddr is set.
	RPCAddr string
	// RPCPort port to bind RPC server to, to serve external supervisor nodes.
	// Binds to any available port if set to 0.
	// Only applicable if RPCAddr is set.
	RPCPort int
	// RPCJwtSecretPath path of JWT secret file to apply authentication to the interop server address.
	RPCJwtSecretPath string
}

func (cfg *Config) Check() error {
	if cfg.RPCAddr == "" {
		return errors.New("must have either a supervisor RPC endpoint to follow, or interop RPC address to serve from")
	}
	return nil
}

func (cfg *Config) Setup(ctx context.Context, logger log.Logger, rollupCfg *rollup.Config, l1 L1Source, l2 L2Source, m opmetrics.RPCMetricer) (SubSystem, error) {
	logger.Info("Setting up Interop RPC server to serve supervisor sync work")
	// Load JWT secret, if any, generate one otherwise.
	jwtSecret, err := rpc.ObtainJWTSecret(logger, cfg.RPCJwtSecretPath, true)
	if err != nil {
		return nil, err
	}
	return managed.NewManagedMode(logger, rollupCfg, cfg.RPCAddr, cfg.RPCPort, jwtSecret, l1, l2, m), nil
}
