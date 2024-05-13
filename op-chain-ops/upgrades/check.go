package upgrades

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/upgrades/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

var (
	// The default values for the ResourceConfig, used as part of
	// an EIP-1559 curve for deposit gas.
	DefaultResourceConfig = bindings.ResourceMeteringResourceConfig{
		MaxResourceLimit:            genesis.MaxResourceLimit,
		ElasticityMultiplier:        genesis.ElasticityMultiplier,
		BaseFeeMaxChangeDenominator: genesis.BaseFeeMaxChangeDenominator,
		MinimumBaseFee:              genesis.MinimumBaseFee,
		SystemTxMaxGas:              genesis.SystemTxMaxGas,
		MaximumBaseFee:              genesis.MaximumBaseFee,
	}
)

// CheckL1 will check that the versions of the contracts on L1 match the versions
// in the superchain registry.
func CheckL1(ctx context.Context, list *superchain.ImplementationList, backend bind.ContractBackend) error {
	if err := CheckVersionedContract(ctx, list.L1CrossDomainMessenger, backend); err != nil {
		return fmt.Errorf("L1CrossDomainMessenger: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.L1ERC721Bridge, backend); err != nil {
		return fmt.Errorf("L1ERC721Bridge: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.L1StandardBridge, backend); err != nil {
		return fmt.Errorf("L1StandardBridge: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.L2OutputOracle, backend); err != nil {
		return fmt.Errorf("L2OutputOracle: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.OptimismMintableERC20Factory, backend); err != nil {
		return fmt.Errorf("OptimismMintableERC20Factory: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.OptimismPortal, backend); err != nil {
		return fmt.Errorf("OptimismPortal: %w", err)
	}
	if err := CheckVersionedContract(ctx, list.SystemConfig, backend); err != nil {
		return fmt.Errorf("SystemConfig: %w", err)
	}
	return nil
}

// CheckVersionedContract will check that the version of the deployed contract matches
// the artifact in the superchain registry.
func CheckVersionedContract(ctx context.Context, contract superchain.VersionedContract, backend bind.ContractBackend) error {
	addr := common.Address(contract.Address)
	code, err := backend.CodeAt(ctx, addr, nil)
	if err != nil {
		return err
	}
	if len(code) == 0 {
		return fmt.Errorf("no code at %s", addr)
	}
	version, err := getVersion(ctx, addr, backend)
	if err != nil {
		return err
	}
	if !cmpVersion(version, contract.Version) {
		return fmt.Errorf("version mismatch: expected %s, got %s", contract.Version, version)
	}
	return nil
}

// getContractVersions will fetch the versions of all of the contracts.
func GetContractVersions(ctx context.Context, addresses *superchain.AddressList, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) (superchain.ContractVersions, error) {
	var versions superchain.ContractVersions
	var err error

	versions.L1CrossDomainMessenger, err = getVersion(ctx, common.Address(addresses.L1CrossDomainMessengerProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("L1CrossDomainMessenger: %w", err)
	}
	versions.L1ERC721Bridge, err = getVersion(ctx, common.Address(addresses.L1ERC721BridgeProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("L1ERC721Bridge: %w", err)
	}
	versions.L1StandardBridge, err = getVersion(ctx, common.Address(addresses.L1StandardBridgeProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("L1StandardBridge: %w", err)
	}
	versions.L2OutputOracle, err = getVersion(ctx, common.Address(addresses.L2OutputOracleProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("L2OutputOracle: %w", err)
	}
	versions.OptimismMintableERC20Factory, err = getVersion(ctx, common.Address(addresses.OptimismMintableERC20FactoryProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("OptimismMintableERC20Factory: %w", err)
	}
	versions.OptimismPortal, err = getVersion(ctx, common.Address(addresses.OptimismPortalProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("OptimismPortal: %w", err)
	}
	versions.SystemConfig, err = getVersion(ctx, common.Address(addresses.SystemConfigProxy), backend)
	if err != nil {
		return versions, fmt.Errorf("SystemConfig: %w", err)
	}
	return versions, err
}

// getVersion will get the version of a contract at a given address.
func getVersion(ctx context.Context, addr common.Address, backend bind.ContractBackend) (string, error) {
	isemver, err := bindings.NewISemver(addr, backend)
	if err != nil {
		return "", fmt.Errorf("%s: %w", addr, err)
	}
	version, err := isemver.Version(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", addr, err)
	}
	return version, nil
}

// cmpVersion will compare 2 semver strings, accounting for
// lack of "v" prefix.
func cmpVersion(v1, v2 string) bool {
	if !strings.HasPrefix(v1, "v") {
		v1 = "v" + v1
	}
	if !strings.HasPrefix(v2, "v") {
		v2 = "v" + v2
	}
	return v1 == v2
}
