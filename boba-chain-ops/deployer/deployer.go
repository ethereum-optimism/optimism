package deployer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/accounts/abi/bind"
	"github.com/ledgerwatch/erigon/accounts/abi/bind/backends"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/ledgerwatch/erigon/params"
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
	Bytecode string
	Address  common.Address
}

type Deployer func(*backends.SimulatedBackend, *bind.TransactOpts, Constructor) (types.Transaction, error)

func NewBackend() *backends.SimulatedBackend {
	return NewBackendWithGenesisTimestamp(0)
}

func NewBackendWithGenesisTimestamp(ts uint64) *backends.SimulatedBackend {
	chainConfig := chain.Config{
		ChainID:               ChainID,
		Consensus:             chain.EtHashConsensus,
		Ethash:                new(chain.EthashConfig),
		HomesteadBlock:        big.NewInt(0),
		DAOForkBlock:          big.NewInt(0),
		ByzantiumBlock:        big.NewInt(0),
		ConstantinopleBlock:   big.NewInt(0),
		TangerineWhistleBlock: big.NewInt(0),
		SpuriousDragonBlock:   big.NewInt(0),
		PetersburgBlock:       big.NewInt(0),
		IstanbulBlock:         big.NewInt(0),
		MuirGlacierBlock:      big.NewInt(0),
		BerlinBlock:           big.NewInt(0),
		LondonBlock:           big.NewInt(0),
		ArrowGlacierBlock:     big.NewInt(0),
		GrayGlacierBlock:      big.NewInt(0),
	}

	return backends.NewSimulatedBackendWithConfig(
		types.GenesisAlloc{
			crypto.PubkeyToAddress(TestKey.PublicKey): types.GenesisAccount{
				Balance: thousandETH,
			},
		},
		&chainConfig,
		15000000,
	)
}

func Deploy(backend *backends.SimulatedBackend, constructors []Constructor, cb Deployer) ([]Deployment, error) {
	results := make([]Deployment, len(constructors))

	opts, err := bind.NewKeyedTransactorWithChainID(TestKey, ChainID)
	if err != nil {
		return nil, err
	}

	opts.GasPrice = big.NewInt(1000000000)
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
			Bytecode: hexutility.Bytes(code).String(),
			Address:  addr,
		}
	}

	return results, nil
}
