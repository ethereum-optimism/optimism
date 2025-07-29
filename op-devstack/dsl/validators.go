package dsl

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/params"
)

// L2NetworkProvider is an interface that provides access to L2 networks
type L2NetworkProvider interface {
	L2Networks() []*L2Network
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

// getChainConfig is a helper function that retrieves the ChainConfig for a specific L2 network.
func getChainConfig(ctx context.Context, system L2NetworkProvider, networkIdx int) (*params.ChainConfig, *uint64, error) {
	l2Networks := system.L2Networks()
	if len(l2Networks) <= networkIdx {
		return nil, nil, fmt.Errorf("network index %d out of range, only %d L2 networks available", networkIdx, len(l2Networks))
	}

	network := l2Networks[networkIdx]
	underlyingNetwork := network.Escape()
	chainConfig := underlyingNetwork.ChainConfig()

	if len(underlyingNetwork.L2ELNodes()) == 0 {
		return nil, nil, fmt.Errorf("no EL nodes found for L2 network %d", networkIdx)
	}

	// Get the first EL node to check the latest block
	elNode := underlyingNetwork.L2ELNodes()[0]
	ethClient := elNode.L2EthClient()

	blockInfo, err := ethClient.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block for L2 network %d: %w", networkIdx, err)
	}

	timestamp := blockInfo.Time()

	return chainConfig, &timestamp, nil
}

// RequiresL2Fork ensures a specific L2 network has a specific fork activated.
func RequiresL2Fork(ctx context.Context, system L2NetworkProvider, networkIdx int, forkName rollup.ForkName) error {
	chainConfig, timestamp, err := getChainConfig(ctx, system, networkIdx)
	if err != nil {
		return err
	}
	if chainConfig == nil {
		return fmt.Errorf("chain config is nil")
	}

	isActive, err := IsForkActivated(chainConfig, forkName, *timestamp)
	if err != nil {
		return err
	}

	if !isActive {
		return fmt.Errorf("L2 network %d does not have fork %s activated", networkIdx, forkName)
	}

	return nil
}

// RequiresL2WithoutFork ensures a specific L2 network does not have a specific fork activated.
func RequiresL2WithoutFork(ctx context.Context, system L2NetworkProvider, networkIdx int, forkName rollup.ForkName) error {
	chainConfig, timestamp, err := getChainConfig(ctx, system, networkIdx)
	if err != nil {
		return err
	}
	if chainConfig == nil {
		return fmt.Errorf("chain config is nil")
	}

	isActive, err := IsForkActivated(chainConfig, forkName, *timestamp)
	if err != nil {
		return err
	}

	if isActive {
		return fmt.Errorf("L2 network %d has fork %s activated, but it should not be", networkIdx, forkName)
	}

	return nil
}
