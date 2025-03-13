package validators

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/params"
)

// getChainConfig is a helper function that retrieves the ForkConfig for a specific L2 chain.
func getChainConfig(t systest.T, sys system.System, chainIdx uint64) (*params.ChainConfig, *uint64, error) {
	if len(sys.L2s()) <= int(chainIdx) {
		return nil, nil, fmt.Errorf("chain index %d out of range, only %d L2 chains available", chainIdx, len(sys.L2s()))
	}

	chain := sys.L2s()[chainIdx]

	chainConfig, err := chain.Config()
	if err != nil || chainConfig == nil {
		return nil, nil, fmt.Errorf("failed to get chain config for L2 chain %d: %w", chainIdx, err)
	}

	block, err := chain.Node().BlockByNumber(t.Context(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block for L2 chain %d: %w", chainIdx, err)
	}

	timestamp := block.Time()

	return chainConfig, &timestamp, nil
}

// IsForkActivated checks if a specific fork is activated at the given timestamp
// based on the chain configuration.
func IsForkActivated(c *params.ChainConfig, forkName rollup.ForkName, timestamp uint64) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("provided chain config is nil")
	}

	switch forkName {
	case rollup.Bedrock:
		// Bedrock is activated based on block number, not timestamp
		return true, nil // Assuming bedrock is always active in the context of this validator
	case rollup.Regolith:
		return c.IsOptimismRegolith(timestamp), nil
	case rollup.Canyon:
		return c.IsOptimismCanyon(timestamp), nil
	case rollup.Ecotone:
		return c.IsOptimismEcotone(timestamp), nil
	case rollup.Fjord:
		return c.IsOptimismFjord(timestamp), nil
	case rollup.Granite:
		return c.IsOptimismGranite(timestamp), nil
	case rollup.Holocene:
		return c.IsOptimismHolocene(timestamp), nil
	case rollup.Isthmus:
		return c.IsOptimismIsthmus(timestamp), nil
	case rollup.Interop:
		return c.IsInterop(timestamp), nil
	default:
		return false, fmt.Errorf("unknown fork name: %s", forkName)
	}
}

// forkConfigValidator is a helper function that checks if a specific L2 chain meets a fork condition.
func forkConfigValidator(chainIdx uint64, forkName rollup.ForkName, shouldBeActive bool, forkConfigMarker interface{}) systest.PreconditionValidator {
	return func(t systest.T, sys system.System) (context.Context, error) {
		chainConfig, timestamp, err := getChainConfig(t, sys, chainIdx)
		if err != nil {
			return nil, err
		}
		if chainConfig == nil {
			return nil, fmt.Errorf("chain config is nil")
		}

		isActive, err := IsForkActivated(chainConfig, forkName, *timestamp)
		if err != nil {
			return nil, err
		}

		if isActive != shouldBeActive {
			if shouldBeActive {
				return nil, fmt.Errorf("L2 chain %d does not have fork %s activated, which it should be for this validator to pass", chainIdx, forkName)
			} else {
				return nil, fmt.Errorf("L2 chain %d has fork %s activated, but it should not be for this validator to pass", chainIdx, forkName)
			}
		}

		return context.WithValue(t.Context(), forkConfigMarker, chainConfig), nil
	}
}

// ChainConfigGetter is a function type that retrieves a ForkConfig from a context.
type ChainConfigGetter = func(context.Context) *params.ChainConfig

// AcquireForkConfig returns a ForkConfigGetter and a PreconditionValidator
// that ensures a ForkConfig is available for the specified L2 chain.
// The ForkConfig can be used to check if various forks are activated.
func acquireForkConfig(chainIdx uint64, forkName rollup.ForkName, shouldBeActive bool) (ChainConfigGetter, systest.PreconditionValidator) {
	chainConfigMarker := new(byte)
	validator := forkConfigValidator(chainIdx, forkName, shouldBeActive, chainConfigMarker)
	return func(ctx context.Context) *params.ChainConfig {
		return ctx.Value(chainConfigMarker).(*params.ChainConfig)
	}, validator
}

// RequiresFork returns a validator that ensures a specific L2 chain has a specific fork activated.
func AcquireL2WithFork(chainIdx uint64, forkName rollup.ForkName) (ChainConfigGetter, systest.PreconditionValidator) {
	return acquireForkConfig(chainIdx, forkName, true)
}

// RequiresNotFork returns a validator that ensures a specific L2 chain does not
// have a specific fork activated. Will not work with the interop fork
// specifically since interop is not an ordered release fork.
func AcquireL2WithoutFork(chainIdx uint64, forkName rollup.ForkName) (ChainConfigGetter, systest.PreconditionValidator) {
	return acquireForkConfig(chainIdx, forkName, false)
}
