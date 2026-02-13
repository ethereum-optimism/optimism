package testutil

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	opbindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func NeedsSuperchainConfigUpgrade(
	ctx context.Context,
	client *ethclient.Client,
	currentProxy, targetImpl common.Address,
) (bool, error) {
	currentVersion, err := superchainConfigVersion(ctx, client, currentProxy)
	if err != nil {
		return false, fmt.Errorf("failed to fetch proxy superchain config version: %w", err)
	}

	targetVersion, err := superchainConfigVersion(ctx, client, targetImpl)
	if err != nil {
		return false, fmt.Errorf("failed to fetch implementation superchain config version: %w", err)
	}

	return currentVersion.LessThan(targetVersion), nil
}

func superchainConfigVersion(
	ctx context.Context,
	client *ethclient.Client,
	addr common.Address,
) (*semver.Version, error) {
	contract, err := opbindings.NewSuperchainConfig(addr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind superchain config at %s: %w", addr.Hex(), err)
	}
	versionStr, err := contract.Version(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to read version from %s: %w", addr.Hex(), err)
	}
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %q from %s: %w", versionStr, addr.Hex(), err)
	}
	return version, nil
}
