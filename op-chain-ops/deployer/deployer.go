package deployer

import (
	"context"
	"fmt"
	"math/big"

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

type Deployment struct {
	Name     string
	Bytecode hexutil.Bytes
	Address  common.Address
}

type Deployer func(*backends.SimulatedBackend, *bind.TransactOpts, Constructor) (*types.Transaction, error)

func NewBackend() *backends.SimulatedBackend {
	return NewBackendWithGenesisTimestamp(0)
}

func NewBackendWithGenesisTimestamp(ts uint64) *backends.SimulatedBackend {
	chainConfig := params.ChainConfig{
		ChainID:             ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
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

	return backends.NewSimulatedBackendWithOpts(
		backends.WithCacheConfig(&core.CacheConfig{
			Preimages: true,
		}),
		backends.WithGenesis(core.Genesis{
			Config:     &chainConfig,
			Timestamp:  ts,
			Difficulty: big.NewInt(0),
			Alloc: core.GenesisAlloc{
				crypto.PubkeyToAddress(TestKey.PublicKey): {Balance: thousandETH},
			},
			GasLimit: 15000000,
		}),
		backends.WithConsensus(beacon.New(ethash.NewFaker())),
	)
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
