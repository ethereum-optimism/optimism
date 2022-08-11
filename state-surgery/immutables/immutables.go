package immutables

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/l2geth/common/hexutil"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// testKey is the same test key that geth uses
var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// chainID is the chain id used for simulated backends
var chainID = big.NewInt(1337)

// TODO(tynes): we are planning on making some constructor arguments
// into immutables before the final deployment of the system. This
// means that this struct will need to be updated with an additional
// parameter: Args []interface{}{} and each step will need to typecast
// each argument before doing the simulated deployment
type Deployment struct {
	Name string
}

// DeploymentResults represents the output of deploying each of the
// contracts so that the immutables can be set properly in the bytecode.
type DeploymentResults map[string]hexutil.Bytes

// TODO(tynes): once there are deploy time config params,
// pass in a config struct to this function that comes from
// a JSON file/cli flags and then populate the Deployment
// Args.
func OptimismBuild() (DeploymentResults, error) {
	deployments := []Deployment{
		{
			Name: "GasPriceOracle",
		},
		{
			Name: "L1Block",
		},
		{
			Name: "L2CrossDomainMessenger",
		},
		{
			Name: "L2StandardBridge",
		},
		{
			Name: "L2ToL1MessagePasser",
		},
		{
			Name: "SequencerFeeVault",
		},
	}
	return Build(deployments)
}

// Build will deploy contracts to a simulated backend so that their immutables
// can be properly set. The bytecode returned in the results is suitable to be
// inserted into the state via state surgery.
func Build(deployments []Deployment) (DeploymentResults, error) {
	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
		},
		15000000,
	)

	results := make(DeploymentResults)

	opts, err := bind.NewKeyedTransactorWithChainID(testKey, chainID)
	if err != nil {
		return nil, err
	}

	for _, deployment := range deployments {
		var addr common.Address
		switch deployment.Name {
		case "GasPriceOracle":
			// The owner of the gas price oracle is not immutable, not required
			// to be set here. It cannot be `address(0)`
			owner := common.Address{1}
			addr, _, _, err = bindings.DeployGasPriceOracle(opts, backend, owner)
			if err != nil {
				return nil, err
			}
		case "L1Block":
			// No arguments required for the L1Block contract
			addr, _, _, err = bindings.DeployL1Block(opts, backend)
			if err != nil {
				return nil, err
			}
		case "L2CrossDomainMessenger":
			// The L1CrossDomainMessenger value is not immutable, no need to set
			// it here correctly
			l1CrossDomainMessenger := common.Address{}
			addr, _, _, err = bindings.DeployL2CrossDomainMessenger(opts, backend, l1CrossDomainMessenger)
			if err != nil {
				return nil, err
			}
		case "L2StandardBridge":
			// The OtherBridge value is not immutable, no need to set
			otherBridge := common.Address{}
			addr, _, _, err = bindings.DeployL2StandardBridge(opts, backend, otherBridge)
		case "L2ToL1MessagePasser":
			// No arguments required for L2ToL1MessagePasser
			addr, _, _, err = bindings.DeployL2ToL1MessagePasser(opts, backend)
		case "SequencerFeeVault":
			// No arguments to SequencerFeeVault
			addr, _, _, err = bindings.DeploySequencerFeeVault(opts, backend)
		default:
			return nil, fmt.Errorf("unknown contract: %s", deployment.Name)
		}

		backend.Commit()
		if addr == (common.Address{}) {
			return nil, fmt.Errorf("no address for %s", deployment.Name)
		}
		code, err := backend.CodeAt(context.Background(), addr, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot fetch code for %s", deployment.Name)
		}
		results[deployment.Name] = code
	}

	return results, nil
}
