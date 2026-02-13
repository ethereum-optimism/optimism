package testutils

import (
	"context"
	"encoding/hex"
	"log"
	"maps"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

type SimulatedEthClient struct {
	defaultAccountPvtKey string
	defaultAccountAddr   common.Address
	backend              *simulated.Backend
	simulated.Client
}

type SimulatedEthClientConfig struct {
	GenesisAccountsBalances map[common.Address]*big.Int
	BlockGasLimit           uint64
}

func defaultSimulatedEthClientConfig() *SimulatedEthClientConfig {
	return &SimulatedEthClientConfig{
		GenesisAccountsBalances: map[common.Address]*big.Int{},
		BlockGasLimit:           10000000,
	}
}

func WithAccountBalance(address common.Address, balance *big.Int) func(*SimulatedEthClientConfig) {
	return func(c *SimulatedEthClientConfig) {
		c.GenesisAccountsBalances[address] = balance
	}
}

func WithAccountsBalances(balances map[common.Address]*big.Int) func(*SimulatedEthClientConfig) {
	return func(c *SimulatedEthClientConfig) {
		maps.Copy(balances, c.GenesisAccountsBalances)
	}
}

func WithBlockGasLimit(limit uint64) func(*SimulatedEthClientConfig) {
	return func(c *SimulatedEthClientConfig) {
		c.BlockGasLimit = limit
	}
}

func NewSimulatedEthClient(opts ...func(*SimulatedEthClientConfig)) *SimulatedEthClient {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}

	cfg := defaultSimulatedEthClientConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	genesisAlloc := types.GenesisAlloc{
		auth.From: {
			Balance: eth.HundredEther.ToBig(),
		},
	}
	for addr, balance := range cfg.GenesisAccountsBalances {
		genesisAlloc[addr] = types.Account{
			Balance: balance,
		}
	}

	backend := simulated.NewBackend(genesisAlloc, simulated.WithBlockGasLimit(cfg.BlockGasLimit))
	return &SimulatedEthClient{
		defaultAccountAddr:   auth.From,
		defaultAccountPvtKey: hex.EncodeToString(crypto.FromECDSA(privateKey)),
		backend:              backend,
		Client:               backend.Client(),
	}
}

func (c *SimulatedEthClient) RPC() string {
	return "https://localhost:8545/apikeyfoobarbizzbuzz"
}

func (c *SimulatedEthClient) PrivateKey() string {
	return c.defaultAccountPvtKey
}

func (c *SimulatedEthClient) Address() common.Address {
	return c.defaultAccountAddr
}

func (c *SimulatedEthClient) Commit() {
	c.backend.Commit()
}

func (c *SimulatedEthClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return 0, nil
}
