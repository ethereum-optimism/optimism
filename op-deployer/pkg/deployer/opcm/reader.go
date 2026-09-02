package opcm

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

// ImplementationsMethod is the canonical implementations() signature for OPContractsManagerV2.
var ImplementationsMethod = w3.MustNewFunc("implementations()", `(address superchainConfigImpl,address l1ERC721BridgeImpl,address optimismPortalImpl,address ethLockboxImpl,address systemConfigImpl,address optimismMintableERC20FactoryImpl,address l1CrossDomainMessengerImpl,address l1StandardBridgeImpl,address disputeGameFactoryImpl,address anchorStateRegistryImpl,address delayedWETHImpl,address mipsImpl,address faultDisputeGameImpl,address permissionedDisputeGameImpl,address superFaultDisputeGameImpl,address superPermissionedDisputeGameImpl,address zkDisputeGameImpl,address storageSetterImpl,address sp1PlonkAdapterImpl)`)

var (
	contractsContainerMethod    = w3.MustNewFunc("contractsContainer()", "address")
	opcmUtilsMethod             = w3.MustNewFunc("opcmUtils()", "address")
	opcmMigratorMethod          = w3.MustNewFunc("opcmMigrator()", "address")
	opcmStandardValidatorMethod = w3.MustNewFunc("opcmStandardValidator()", "address")
	mipsOracleMethod            = w3.MustNewFunc("oracle()", "address")
)

// opcmImplementations mirrors IOPContractsManagerContainer.Implementations.
type opcmImplementations struct {
	SuperchainConfigImpl             common.Address
	L1ERC721BridgeImpl               common.Address
	OptimismPortalImpl               common.Address
	ETHLockboxImpl                   common.Address `abi:"ethLockboxImpl"`
	SystemConfigImpl                 common.Address
	OptimismMintableERC20FactoryImpl common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1StandardBridgeImpl             common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	DelayedWETHImpl                  common.Address
	MipsImpl                         common.Address
	FaultDisputeGameImpl             common.Address
	PermissionedDisputeGameImpl      common.Address
	SuperFaultDisputeGameImpl        common.Address
	SuperPermissionedDisputeGameImpl common.Address
	ZkDisputeGameImpl                common.Address
	StorageSetterImpl                common.Address
	SP1PlonkAdapterImpl              common.Address `abi:"sp1PlonkAdapterImpl"`
}

// CallGetter encodes, calls and decodes a zero-argument view getter.
func CallGetter[T any](
	ctx context.Context,
	backend CallContractBackend,
	contract common.Address,
	method *w3.Func,
) (T, error) {
	var value T
	calldata, err := method.EncodeArgs()
	if err != nil {
		return value, fmt.Errorf("failed to encode %s call: %w", method.Signature, err)
	}
	result, err := backend.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: calldata}, nil)
	if err != nil {
		return value, fmt.Errorf("%s call to %s failed: %w", method.Signature, contract, err)
	}
	if err := method.DecodeReturns(result, &value); err != nil {
		return value, fmt.Errorf("failed to decode %s result from %s: %w", method.Signature, contract, err)
	}
	return value, nil
}

// ReadImplementations resolves the full implementation address set from a deployed OPCM.
// It is the eth_call equivalent of ReadImplementationAddresses.s.sol.
func ReadImplementations(
	ctx context.Context,
	backend CallContractBackend,
	opcmAddr common.Address,
) (*addresses.ImplementationsContracts, error) {
	impls, err := CallGetter[opcmImplementations](ctx, backend, opcmAddr, ImplementationsMethod)
	if err != nil {
		return nil, err
	}
	container, err := CallGetter[common.Address](ctx, backend, opcmAddr, contractsContainerMethod)
	if err != nil {
		return nil, err
	}
	utils, err := CallGetter[common.Address](ctx, backend, opcmAddr, opcmUtilsMethod)
	if err != nil {
		return nil, err
	}
	migrator, err := CallGetter[common.Address](ctx, backend, opcmAddr, opcmMigratorMethod)
	if err != nil {
		return nil, err
	}
	validator, err := CallGetter[common.Address](ctx, backend, opcmAddr, opcmStandardValidatorMethod)
	if err != nil {
		return nil, err
	}

	// PreimageOracle is not part of the container struct, MIPS holds it.
	if impls.MipsImpl == (common.Address{}) {
		return nil, fmt.Errorf("OPCM at %s reports a zero MIPS implementation", opcmAddr)
	}
	oracle, err := CallGetter[common.Address](ctx, backend, impls.MipsImpl, mipsOracleMethod)
	if err != nil {
		return nil, err
	}

	return &addresses.ImplementationsContracts{
		OpcmStandardValidatorImpl:        validator,
		OpcmUtilsImpl:                    utils,
		OpcmMigratorImpl:                 migrator,
		OpcmV2Impl:                       opcmAddr,
		OpcmContainerImpl:                container,
		DelayedWethImpl:                  impls.DelayedWETHImpl,
		OptimismPortalImpl:               impls.OptimismPortalImpl,
		EthLockboxImpl:                   impls.ETHLockboxImpl,
		PreimageOracleImpl:               oracle,
		MipsImpl:                         impls.MipsImpl,
		SystemConfigImpl:                 impls.SystemConfigImpl,
		L1CrossDomainMessengerImpl:       impls.L1CrossDomainMessengerImpl,
		L1Erc721BridgeImpl:               impls.L1ERC721BridgeImpl,
		L1StandardBridgeImpl:             impls.L1StandardBridgeImpl,
		OptimismMintableErc20FactoryImpl: impls.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImpl:           impls.DisputeGameFactoryImpl,
		AnchorStateRegistryImpl:          impls.AnchorStateRegistryImpl,
		FaultDisputeGameImpl:             impls.FaultDisputeGameImpl,
		PermissionedDisputeGameImpl:      impls.PermissionedDisputeGameImpl,
		ZkDisputeGameImpl:                impls.ZkDisputeGameImpl,
		StorageSetterImpl:                impls.StorageSetterImpl,
		SP1PlonkAdapterImpl:              impls.SP1PlonkAdapterImpl,
		SuperFaultDisputeGameImpl:        impls.SuperFaultDisputeGameImpl,
		SuperPermissionedDisputeGameImpl: impls.SuperPermissionedDisputeGameImpl,
	}, nil
}
