package immutables

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
)

// PredeploysImmutableConfig represents the set of L2 predeploys. It includes all
// L2 predeploys - not just ones with immutable values. This is to be very explicit
// about the configuration of the predeploys. It is important that the inner struct
// fields are in the same order as the constructor arguments in the solidity code.
type PredeploysImmutableConfig struct {
	L2ToL1MessagePasser    struct{}
	DeployerWhitelist      struct{}
	WETH9                  struct{}
	L2CrossDomainMessenger struct {
		OtherMessenger common.Address
	}
	L2StandardBridge struct {
		OtherBridge common.Address
		Messenger   common.Address
	}
	SequencerFeeVault struct {
		Recipient           common.Address
		MinWithdrawalAmount *big.Int
		WithdrawalNetwork   uint8
	}
	OptimismMintableERC20Factory struct {
		Bridge common.Address
	}
	L1BlockNumber       struct{}
	GasPriceOracle      struct{}
	L1Block             struct{}
	GovernanceToken     struct{}
	LegacyMessagePasser struct{}
	L2ERC721Bridge      struct {
		OtherBridge common.Address
		Messenger   common.Address
	}
	OptimismMintableERC721Factory struct {
		Bridge        common.Address
		RemoteChainId *big.Int
	}
	ProxyAdmin   struct{}
	BaseFeeVault struct {
		Recipient           common.Address
		MinWithdrawalAmount *big.Int
		WithdrawalNetwork   uint8
	}
	L1FeeVault struct {
		Recipient           common.Address
		MinWithdrawalAmount *big.Int
		WithdrawalNetwork   uint8
	}
	SchemaRegistry struct{}
	EAS            struct {
		Name string
	}
	Create2Deployer              struct{}
	MultiCall3                   struct{}
	Safe_v130                    struct{}
	SafeL2_v130                  struct{}
	MultiSendCallOnly_v130       struct{}
	SafeSingletonFactory         struct{}
	DeterministicDeploymentProxy struct{}
	MultiSend_v130               struct{}
	Permit2                      struct{}
	SenderCreator                struct{}
	EntryPoint                   struct{}
}

// Check will ensure that the required fields are set on the config.
// An error returned by `GetImmutableReferences` means that the solc compiler
// output for the contract has no immutables in it.
func (c *PredeploysImmutableConfig) Check() error {
	return c.ForEach(func(name string, values any) error {
		val := reflect.ValueOf(values)
		if val.NumField() == 0 {
			return nil
		}

		has, err := bindings.HasImmutableReferences(name)
		exists := err == nil && has
		isZero := val.IsZero()

		// There are immutables defined in the solc output and
		// the config is not empty.
		if exists && !isZero {
			return nil
		}
		// There are no immutables defined in the solc output and
		// the config is empty
		if !exists && isZero {
			return nil
		}

		return fmt.Errorf("invalid immutables config: field %s: %w", name, err)
	})
}

// ForEach will iterate over each of the fields in the config and call the callback
// with the value of the field as well as the field's name.
func (c *PredeploysImmutableConfig) ForEach(cb func(string, any) error) error {
	val := reflect.ValueOf(c).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		internalVal := reflect.ValueOf(field.Interface())
		if err := cb(typ.Field(i).Name, internalVal.Interface()); err != nil {
			return err
		}
	}
	return nil
}

// DeploymentResults represents the output of deploying each of the
// contracts so that the immutables can be set properly in the bytecode.
type DeploymentResults map[string]hexutil.Bytes

// Deploy will deploy L2 predeploys that include immutables. This is to prevent the need
// for parsing the solc output to find the correct immutable offsets and splicing in the values.
// Skip any predeploys that do not have immutables as their bytecode will be directly inserted
// into the state. This does not currently support recursive structs.
func Deploy(config *PredeploysImmutableConfig) (DeploymentResults, error) {
	if err := config.Check(); err != nil {
		return DeploymentResults{}, err
	}
	deployments := make([]deployer.Constructor, 0)

	val := reflect.ValueOf(config).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if reflect.ValueOf(field.Interface()).IsZero() {
			continue
		}

		deployment := deployer.Constructor{
			Name: typ.Field(i).Name,
			Args: []any{},
		}

		internalVal := reflect.ValueOf(field.Interface())
		for j := 0; j < internalVal.NumField(); j++ {
			internalField := internalVal.Field(j)
			deployment.Args = append(deployment.Args, internalField.Interface())
		}

		deployments = append(deployments, deployment)
	}

	results, err := deployContractsWithImmutables(deployments)
	if err != nil {
		return nil, fmt.Errorf("cannot deploy contracts with immutables: %w", err)
	}
	return results, nil
}

// deployContractsWithImmutables will deploy contracts to a simulated backend so that their immutables
// can be properly set. The bytecode returned in the results is suitable to be
// inserted into the state via state surgery.
func deployContractsWithImmutables(constructors []deployer.Constructor) (DeploymentResults, error) {
	backend, err := deployer.NewL2Backend()
	if err != nil {
		return nil, err
	}
	deployments, err := deployer.Deploy(backend, constructors, l2ImmutableDeployer)
	if err != nil {
		return nil, err
	}
	results := make(DeploymentResults)
	for _, dep := range deployments {
		results[dep.Name] = dep.Bytecode
	}
	return results, nil
}

// l2ImmutableDeployer will deploy L2 predeploys that contain immutables to the simulated backend.
// It only needs to care about the predeploys that have immutables so that the deployed bytecode
// has the dynamic value set at the correct location in the bytecode.
func l2ImmutableDeployer(backend *backends.SimulatedBackend, opts *bind.TransactOpts, deployment deployer.Constructor) (*types.Transaction, error) {
	var tx *types.Transaction
	var recipient common.Address
	var minimumWithdrawalAmount *big.Int
	var withdrawalNetwork uint8
	var err error

	if has, err := bindings.HasImmutableReferences(deployment.Name); err != nil || !has {
		return nil, fmt.Errorf("%s does not have immutables: %w", deployment.Name, err)
	}

	switch deployment.Name {
	case "L2CrossDomainMessenger":
		otherMessenger, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for otherMessenger")
		}
		_, tx, _, err = bindings.DeployL2CrossDomainMessenger(opts, backend, otherMessenger)
	case "L2StandardBridge":
		otherBridge, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for otherBridge")
		}
		_, tx, _, err = bindings.DeployL2StandardBridge(opts, backend, otherBridge)
	case "SequencerFeeVault":
		recipient, minimumWithdrawalAmount, withdrawalNetwork, err = prepareFeeVaultArguments(deployment)
		if err != nil {
			return nil, err
		}
		_, tx, _, err = bindings.DeploySequencerFeeVault(opts, backend, recipient, minimumWithdrawalAmount, withdrawalNetwork)
	case "BaseFeeVault":
		recipient, minimumWithdrawalAmount, withdrawalNetwork, err = prepareFeeVaultArguments(deployment)
		if err != nil {
			return nil, err
		}
		_, tx, _, err = bindings.DeployBaseFeeVault(opts, backend, recipient, minimumWithdrawalAmount, withdrawalNetwork)
	case "L1FeeVault":
		recipient, minimumWithdrawalAmount, withdrawalNetwork, err = prepareFeeVaultArguments(deployment)
		if err != nil {
			return nil, err
		}
		_, tx, _, err = bindings.DeployL1FeeVault(opts, backend, recipient, minimumWithdrawalAmount, withdrawalNetwork)
	case "OptimismMintableERC20Factory":
		bridge, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for bridge")
		}
		// Sanity check that the argument is correct
		if bridge != predeploys.L2StandardBridgeAddr {
			return nil, fmt.Errorf("invalid bridge address")
		}
		_, tx, _, err = bindings.DeployOptimismMintableERC20Factory(opts, backend, bridge)
	case "L2ERC721Bridge":
		otherBridge, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for otherBridge")
		}
		_, tx, _, err = bindings.DeployL2ERC721Bridge(opts, backend, otherBridge)
	case "OptimismMintableERC721Factory":
		bridge, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for bridge")
		}
		remoteChainId, ok := deployment.Args[1].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("invalid type for remoteChainId")
		}
		_, tx, _, err = bindings.DeployOptimismMintableERC721Factory(opts, backend, bridge, remoteChainId)
	case "EAS":
		_, tx, _, err = bindings.DeployEAS(opts, backend)
	default:
		return tx, fmt.Errorf("unknown contract: %s", deployment.Name)
	}

	return tx, err
}

// prepareFeeVaultArguments is a helper function that parses the arguments for the fee vault contracts.
func prepareFeeVaultArguments(deployment deployer.Constructor) (common.Address, *big.Int, uint8, error) {
	recipient, ok := deployment.Args[0].(common.Address)
	if !ok {
		return common.Address{}, nil, 0, fmt.Errorf("invalid type for recipient")
	}
	minimumWithdrawalAmountHex, ok := deployment.Args[1].(*big.Int)
	if !ok {
		return common.Address{}, nil, 0, fmt.Errorf("invalid type for minimumWithdrawalAmount")
	}
	withdrawalNetwork, ok := deployment.Args[2].(uint8)
	if !ok {
		return common.Address{}, nil, 0, fmt.Errorf("invalid type for withdrawalNetwork")
	}
	return recipient, minimumWithdrawalAmountHex, withdrawalNetwork, nil
}
