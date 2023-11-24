package immutables

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum-optimism/superchain-registry/superchain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

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
		Messenger   common.Address
		OtherBridge common.Address
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
}

var Create2DeployerCodeHash = common.HexToHash("0xb0550b5b431e30d38000efb7107aaa0ade03d48a7198a140edda9d27134468b2")

// Check will ensure that the required fields are set on the config.
func (c *PredeploysImmutableConfig) Check() error {
	if c.L2CrossDomainMessenger.OtherMessenger == (common.Address{}) {
		return errors.New("L2CrossDomainMessenger otherMessenger not set")
	}
	if c.L2StandardBridge.OtherBridge == (common.Address{}) {
		return errors.New("L2StandardBridge otherBridge not set")
	}
	if c.SequencerFeeVault.Recipient == (common.Address{}) {
		return errors.New("SequencerFeeVault recipient not set")
	}
	if c.SequencerFeeVault.MinWithdrawalAmount == nil || c.SequencerFeeVault.MinWithdrawalAmount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("SequencerFeeVault minWithdrawalAmount not set")
	}
	if c.OptimismMintableERC20Factory.Bridge == (common.Address{}) {
		return errors.New("OptimismMintableERC20Factory bridge not set")
	}
	if c.L2ERC721Bridge.Messenger == (common.Address{}) {
		return errors.New("L2ERC721Bridge messenger not set")
	}
	if c.L2ERC721Bridge.OtherBridge == (common.Address{}) {
		return errors.New("L2ERC721Bridge otherBridge not set")
	}
	if c.OptimismMintableERC721Factory.Bridge == (common.Address{}) {
		return errors.New("OptimismMintableERC721Factory bridge not set")
	}
	if c.OptimismMintableERC721Factory.RemoteChainId == nil || c.OptimismMintableERC721Factory.RemoteChainId.Cmp(big.NewInt(0)) == 0 {
		return errors.New("OptimismMintableERC721Factory remoteChainId not set")
	}
	if c.BaseFeeVault.Recipient == (common.Address{}) {
		return errors.New("BaseFeeVault recipient not set")
	}
	if c.BaseFeeVault.MinWithdrawalAmount == nil || c.BaseFeeVault.MinWithdrawalAmount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("BaseFeeVault minWithdrawalAmount not set")
	}
	if c.L1FeeVault.Recipient == (common.Address{}) {
		return errors.New("L1FeeVault recipient not set")
	}
	if c.L1FeeVault.MinWithdrawalAmount == nil || c.L1FeeVault.MinWithdrawalAmount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("L1FeeVault minWithdrawalAmount not set")
	}
	return nil
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

// BuildOptimism will deploy L2 predeploys that include immutables. This is to prevent the need
// for parsing the solc output to find the correct immutable offsets and splicing in the values.
// Skip any predeploys that do not have immutables as their bytecode will be directly inserted
// into the state. This does not currently support recursive structs.
func BuildOptimism(config *PredeploysImmutableConfig) (DeploymentResults, error) {
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
	superchainPredeploys := []deployer.SuperchainPredeploy{
		{
			Name:     "Create2Deployer",
			CodeHash: Create2DeployerCodeHash,
		},
	}
	return BuildL2(deployments, superchainPredeploys)
}

// BuildL2 will deploy contracts to a simulated backend so that their immutables
// can be properly set. The bytecode returned in the results is suitable to be
// inserted into the state via state surgery.
func BuildL2(constructors []deployer.Constructor, superchainPredeploys []deployer.SuperchainPredeploy) (DeploymentResults, error) {
	log.Info("Creating L2 state")
	deployments, err := deployer.Deploy(deployer.NewL2Backend(), constructors, l2Deployer)
	if err != nil {
		return nil, err
	}
	results := make(DeploymentResults)
	for _, dep := range deployments {
		results[dep.Name] = dep.Bytecode
	}
	for _, dep := range superchainPredeploys {
		code, err := superchain.LoadContractBytecode(superchain.Hash(dep.CodeHash))
		if err != nil {
			return nil, err
		}
		results[dep.Name] = code
	}
	return results, nil
}

func l2Deployer(backend *backends.SimulatedBackend, opts *bind.TransactOpts, deployment deployer.Constructor) (*types.Transaction, error) {
	var tx *types.Transaction
	var recipient common.Address
	var minimumWithdrawalAmount *big.Int
	var withdrawalNetwork uint8
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
		_, tx, _, err = bindings.DeployOptimismMintableERC20Factory(opts, backend, predeploys.L2StandardBridgeAddr)
	case "DeployerWhitelist":
		_, tx, _, err = bindings.DeployDeployerWhitelist(opts, backend)
	case "LegacyMessagePasser":
		_, tx, _, err = bindings.DeployLegacyMessagePasser(opts, backend)
	case "L1BlockNumber":
		_, tx, _, err = bindings.DeployL1BlockNumber(opts, backend)
	case "L2ERC721Bridge":
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
	case "LegacyERC20ETH":
		_, tx, _, err = bindings.DeployLegacyERC20ETH(opts, backend)
	case "EAS":
		_, tx, _, err = bindings.DeployEAS(opts, backend)
	case "SchemaRegistry":
		_, tx, _, err = bindings.DeploySchemaRegistry(opts, backend)
	default:
		return tx, fmt.Errorf("unknown contract: %s", deployment.Name)
	}

	return tx, err
}

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
