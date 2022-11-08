package immutables

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ImmutableValues represents the values to be set in immutable code.
// The key is the name of the variable and the value is the value to set in
// immutable code.
type ImmutableValues map[string]any

// ImmutableConfig represents the immutable configuration for the L2 predeploy
// contracts.
type ImmutableConfig map[string]ImmutableValues

// DeploymentResults represents the output of deploying each of the
// contracts so that the immutables can be set properly in the bytecode.
type DeploymentResults map[string]hexutil.Bytes

// BuildOptimism will deploy the L2 predeploys so that their immutables are set
// correctly.
func BuildOptimism(immutable ImmutableConfig) (DeploymentResults, error) {
	deployments := []deployer.Constructor{
		{
			Name: "GasPriceOracle",
		},
		{
			Name: "L1Block",
		},
		{
			Name: "L2CrossDomainMessenger",
			Args: []interface{}{
				immutable["L2CrossDomainMessenger"]["otherMessenger"],
			},
		},
		{
			Name: "L2StandardBridge",
			Args: []interface{}{
				immutable["L2StandardBridge"]["otherBridge"],
			},
		},
		{
			Name: "L2ToL1MessagePasser",
		},
		{
			Name: "SequencerFeeVault",
			Args: []interface{}{
				immutable["SequencerFeeVault"]["recipient"],
			},
		},
		{
			Name: "BaseFeeVault",
			Args: []interface{}{
				immutable["BaseFeeVault"]["recipient"],
			},
		},
		{
			Name: "L1FeeVault",
			Args: []interface{}{
				immutable["L1FeeVault"]["recipient"],
			},
		},
		{
			Name: "OptimismMintableERC20Factory",
		},
		{
			Name: "DeployerWhitelist",
		},
		{
			Name: "LegacyMessagePasser",
		},
		{
			Name: "L1BlockNumber",
		},
		{
			Name: "L2ERC721Bridge",
			Args: []interface{}{
				predeploys.L2CrossDomainMessengerAddr,
				immutable["L2ERC721Bridge"]["otherBridge"],
			},
		},
		{
			Name: "OptimismMintableERC721Factory",
			Args: []interface{}{
				predeploys.L2ERC721BridgeAddr,
				immutable["OptimismMintableERC721Factory"]["remoteChainId"],
			},
		},
	}
	return BuildL2(deployments)
}

// BuildL2 will deploy contracts to a simulated backend so that their immutables
// can be properly set. The bytecode returned in the results is suitable to be
// inserted into the state via state surgery.
func BuildL2(constructors []deployer.Constructor) (DeploymentResults, error) {
	deployments, err := deployer.Deploy(deployer.NewBackend(), constructors, l2Deployer)
	if err != nil {
		return nil, err
	}
	results := make(DeploymentResults)
	for _, dep := range deployments {
		results[dep.Name] = dep.Bytecode
	}
	return results, nil
}

func l2Deployer(backend *backends.SimulatedBackend, opts *bind.TransactOpts, deployment deployer.Constructor) (*types.Transaction, error) {
	var tx *types.Transaction
	var err error
	switch deployment.Name {
	case "GasPriceOracle":
		_, tx, _, err = bindings.DeployGasPriceOracle(opts, backend)
	case "L1Block":
		// No arguments required for the L1Block contract
		_, tx, _, err = bindings.DeployL1Block(opts, backend)
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
	case "L2ToL1MessagePasser":
		// No arguments required for L2ToL1MessagePasser
		_, tx, _, err = bindings.DeployL2ToL1MessagePasser(opts, backend)
	case "SequencerFeeVault":
		recipient, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for recipient")
		}
		_, tx, _, err = bindings.DeploySequencerFeeVault(opts, backend, recipient)
	case "BaseFeeVault":
		recipient, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for recipient")
		}
		_, tx, _, err = bindings.DeployBaseFeeVault(opts, backend, recipient)
	case "L1FeeVault":
		recipient, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for recipient")
		}
		_, tx, _, err = bindings.DeployL1FeeVault(opts, backend, recipient)
	case "OptimismMintableERC20Factory":
		_, tx, _, err = bindings.DeployOptimismMintableERC20Factory(opts, backend, predeploys.L2StandardBridgeAddr)
	case "DeployerWhitelist":
		_, tx, _, err = bindings.DeployDeployerWhitelist(opts, backend)
	case "LegacyMessagePasser":
		_, tx, _, err = bindings.DeployLegacyMessagePasser(opts, backend)
	case "L1BlockNumber":
		_, tx, _, err = bindings.DeployL1BlockNumber(opts, backend)
	case "L2ERC721Bridge":
		// TODO(tynes): messenger should be hardcoded in the contract
		messenger, ok := deployment.Args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for messenger")
		}
		otherBridge, ok := deployment.Args[1].(common.Address)
		if !ok {
			return nil, fmt.Errorf("invalid type for otherBridge")
		}
		_, tx, _, err = bindings.DeployL2ERC721Bridge(opts, backend, messenger, otherBridge)
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
	default:
		return tx, fmt.Errorf("unknown contract: %s", deployment.Name)
	}

	return tx, err
}
