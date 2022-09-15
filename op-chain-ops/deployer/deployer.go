package deployer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

type Deployment struct {
	Name     string
	Bytecode hexutil.Bytes
	Address  common.Address
}

type Deployer func(*backends.SimulatedBackend, *bind.TransactOpts, Constructor) (common.Address, error)

func NewBackend() *backends.SimulatedBackend {
	return backends.NewSimulatedBackendWithCacheConfig(
		&core.CacheConfig{
			Preimages: true,
		},
		core.GenesisAlloc{
			crypto.PubkeyToAddress(TestKey.PublicKey): {Balance: thousandETH},
		},
		15000000,
	)
}

func Deploy(backend *backends.SimulatedBackend, constructors []Constructor, cb Deployer) ([]Deployment, error) {
	results := make([]Deployment, len(constructors))

	opts, err := bind.NewKeyedTransactorWithChainID(TestKey, ChainID)
	if err != nil {
		return nil, err
	}

	opts.GasLimit = 15_000_000

	for i, deployment := range constructors {
		addr, err := cb(backend, opts, deployment)
		if err != nil {
			return nil, err
		}

		backend.Commit()
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
