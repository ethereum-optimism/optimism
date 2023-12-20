package deployer

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestKey is the same test key that geth uses
var TestKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// ChainID is the chain id used for simulated backends
var ChainID = big.NewInt(1337)

var TestAddress = crypto.PubkeyToAddress(TestKey.PublicKey)

var thousandETH = new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(1000))

type Constructor struct {
	Name string
	Args []interface{}
}

type SuperchainPredeploy struct {
	Name     string
	CodeHash common.Hash
}

type Deployment struct {
	Name     string
	Bytecode hexutil.Bytes
	Address  common.Address
}

type Deployer func(*backends.SimulatedBackend, *bind.TransactOpts, Constructor) (*types.Transaction, error)

// NewL1Backend returns a SimulatedBackend suitable for L1. It has
// the latest L1 hardforks enabled.
func NewL1Backend() (*backends.SimulatedBackend, error) {
	backend, err := NewBackendWithGenesisTimestamp(ChainID, 0, true, nil)
	return backend, err
}

// NewL2Backend returns a SimulatedBackend suitable for L2.
// It has the latest L2 hardforks enabled.
func NewL2Backend() (*backends.SimulatedBackend, error) {
	backend, err := NewBackendWithGenesisTimestamp(ChainID, 0, false, nil)
	return backend, err
}

// NewL2BackendWithChainIDAndPredeploys returns a SimulatedBackend suitable for L2.
// It has the latest L2 hardforks enabled, and allows for the configuration of the network's chain ID and predeploys.
func NewL2BackendWithChainIDAndPredeploys(chainID *big.Int, predeploys map[string]*common.Address) (*backends.SimulatedBackend, error) {
	backend, err := NewBackendWithGenesisTimestamp(chainID, 0, false, predeploys)
	return backend, err
}

func NewBackendWithGenesisTimestamp(chainID *big.Int, ts uint64, shanghai bool, predeploys map[string]*common.Address) (*backends.SimulatedBackend, error) {
	chainConfig := params.ChainConfig{
		ChainID:             chainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		// Activated proof of stake. We manually build/commit blocks in the simulator anyway,
		// and the timestamp verification of PoS is not against the wallclock,
		// preventing blocks from getting stuck temporarily in the future-blocks queue, decreasing setup time a lot.
		MergeNetsplitBlock:            big.NewInt(0),
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
	}

	if shanghai {
		chainConfig.ShanghaiTime = u64ptr(0)
	}

	alloc := core.GenesisAlloc{
		crypto.PubkeyToAddress(TestKey.PublicKey): core.GenesisAccount{
			Balance: thousandETH,
		},
	}
	for name, address := range predeploys {
		bytecode, err := bindings.GetDeployedBytecode(name)
		if err != nil {
			return nil, err
		}
		alloc[*address] = core.GenesisAccount{
			Code: bytecode,
		}
	}

	return backends.NewSimulatedBackendWithOpts(
		backends.WithCacheConfig(&core.CacheConfig{
			Preimages: true,
		}),
		backends.WithGenesis(core.Genesis{
			Config:     &chainConfig,
			Timestamp:  ts,
			Difficulty: big.NewInt(0),
			Alloc:      alloc,
			GasLimit:   30_000_000,
		}),
		backends.WithConsensus(beacon.New(ethash.NewFaker())),
	), nil
}

func Deploy(backend *backends.SimulatedBackend, constructors []Constructor, cb Deployer) ([]Deployment, error) {
	results := make([]Deployment, len(constructors))

	opts, err := bind.NewKeyedTransactorWithChainID(TestKey, ChainID)
	if err != nil {
		return nil, err
	}

	opts.GasLimit = 15_000_000

	ctx := context.Background()
	for i, deployment := range constructors {
		tx, err := cb(backend, opts, deployment)
		if err != nil {
			return nil, err
		}

		// The simulator performs asynchronous processing,
		// so we need to both commit the change here as
		// well as wait for the transaction receipt.
		backend.Commit()
		addr, err := bind.WaitDeployed(ctx, backend, tx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", deployment.Name, err)
		}

		if addr == (common.Address{}) {
			return nil, fmt.Errorf("no address for %s", deployment.Name)
		}
		code, err := backend.CodeAt(context.Background(), addr, nil)
		if len(code) == 0 {
			return nil, fmt.Errorf("no code found for %s", deployment.Name)
		}
		if err != nil {
			return nil, fmt.Errorf("cannot fetch code for %s", deployment.Name)
		}
		results[i] = Deployment{
			Name:     deployment.Name,
			Bytecode: code,
			Address:  addr,
		}
	}

	return results, nil
}

// DeployWithDeterministicDeployer deploys a smart contract on a simulated Ethereum blockchain using a deterministic deployment proxy (Arachnid's).
//
// Parameters:
// - backend: A pointer to backends.SimulatedBackend, representing the simulated Ethereum blockchain.
// Expected to have Arachnid's proxy deployer predeploys at 0x4e59b44847b379578588920cA78FbF26c0B4956C, NewL2BackendWithChainIDAndPredeploys handles this for you.
// - contractName: A string representing the name of the contract to be deployed.
//
// Returns:
// - []byte: The deployed bytecode of the contract.
// - error: An error object indicating any issues encountered during the deployment process.
//
// The function logs a fatal error and exits if there are any issues with transaction mining, if the deployment fails,
// or if the deployed bytecode is not found at the computed address.
func DeployWithDeterministicDeployer(backend *backends.SimulatedBackend, contractName string) ([]byte, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(TestKey, backend.Blockchain().Config().ChainID)
	if err != nil {
		return nil, err
	}

	deployerAddress, err := bindings.GetDeployerAddress(contractName)
	if err != nil {
		return nil, err
	}

	deploymentSalt, err := bindings.GetDeploymentSalt(contractName)
	if err != nil {
		return nil, err
	}

	initBytecode, err := bindings.GetInitBytecode(contractName)
	if err != nil {
		return nil, err
	}

	transactor, err := bindings.NewDeterministicDeploymentProxyTransactor(common.BytesToAddress(deployerAddress), backend)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize deployment proxy transactor at %s: %w", deployerAddress, err)
	}

	tx, err := transactor.Fallback(opts, append(deploymentSalt, initBytecode...))
	if err != nil {
		return nil, err
	}

	backend.Commit()

	receipt, err := bind.WaitMined(context.Background(), backend, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status == 0 {
		return nil, errors.New("failed to deploy contract using proxy deployer")
	}

	address := create2Address(
		deployerAddress,
		deploymentSalt,
		initBytecode,
	)

	code, _ := backend.CodeAt(context.Background(), address, nil)
	if len(code) == 0 {
		return nil, fmt.Errorf("no code found for %s at: %s", contractName, address)
	}

	return code, nil
}

func u64ptr(n uint64) *uint64 {
	return &n
}

// create2Address computes the Ethereum address for a contract created using the CREATE2 opcode.
//
// The CREATE2 opcode allows for more deterministic address generation in Ethereum, as it computes the
// address based on the creator's address, a salt value, and the contract's initialization code.
//
// Parameters:
// - creatorAddress: A byte slice representing the address of the account creating the contract.
// - salt: A byte slice representing the salt used in the address generation process. This can be any 32-byte value.
// - initCode: A byte slice representing the contract's initialization bytecode.
//
// Returns:
// - common.Address: The Ethereum address calculated using the CREATE2 opcode logic.
func create2Address(creatorAddress, salt, initCode []byte) common.Address {
	payload := append([]byte{0xff}, creatorAddress...)
	payload = append(payload, salt...)
	initCodeHash := crypto.Keccak256(initCode)
	payload = append(payload, initCodeHash...)

	return common.BytesToAddress(crypto.Keccak256(payload)[12:])
}
