package test

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/deposit"
	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/l1block"
	rollupNode "github.com/ethereum-optimism/optimistic-specs/opnode/node"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// deriveAddress returns the address associated derivation path for the wallet.
// It will panic if the derivation path is not correctly formatted.
func deriveAddress(w accounts.Wallet, path string) common.Address {
	return deriveAccount(w, path).Address
}

// deriveAccount returns the account associated derivation path for the wallet.
// It will panic if the derivation path is not correctly formatted.
func deriveAccount(w accounts.Wallet, path string) accounts.Account {
	derivPath := hdwallet.MustParseDerivationPath(path)
	account, err := w.Derive(derivPath, true)
	if err != nil {
		panic(err)
	}
	return account
}

type SystemConfig struct {
	Mnemonic                   string
	Premine                    map[string]int // Derivation path -> amount in ETH (not wei)
	CliqueSignerDerivationPath string
	BatchSubmitterHDPath       string
	DepositContractAddress     common.Address
	L1InfoPredeployAddress     common.Address

	L1WsAddr     string
	L1WsPort     int
	L1ChainID    *big.Int
	L2ChainID    *big.Int
	Nodes        map[string]rollupNode.Config // Per node config. Don't use populate rollup.Config
	Loggers      map[string]log.Logger
	RollupConfig rollup.Config // Shared rollup configs
}

type System struct {
	cfg SystemConfig

	// Retain wallet
	wallet *hdwallet.Wallet

	// Connections to running nodes
	nodes        map[string]*node.Node
	backends     map[string]*eth.Ethereum
	Clients      map[string]*ethclient.Client
	RolupGenesis rollup.Genesis
	rollupNodes  map[string]*rollupNode.OpNode
}

func precompileAlloc() core.GenesisAlloc {
	alloc := make(map[common.Address]core.GenesisAccount)
	var addr [common.AddressLength]byte
	for i := 0; i < 256; i++ {
		addr[common.AddressLength-1] = byte(i)
		alloc[addr] = core.GenesisAccount{Balance: common.Big1}
	}
	return alloc
}

func cliqueExtraData(w accounts.Wallet, signers []string) []byte {
	// 32 Empty bytes
	ret := make([]byte, 32)
	// Signer addresses
	for _, signer := range signers {
		address := deriveAddress(w, signer)
		// Was not able to automatically do this
		for i := 0; i < len(address); i++ {
			ret = append(ret, address[i])
		}
	}
	// 65 Empty bytes
	t := make([]byte, 65)
	return append(ret, t...)
}

func (sys *System) Close() {
	for _, node := range sys.rollupNodes {
		node.Stop()
	}
	for _, node := range sys.nodes {
		node.Close()
	}
}

func (cfg SystemConfig) start() (*System, error) {
	sys := &System{
		cfg:         cfg,
		nodes:       make(map[string]*node.Node),
		backends:    make(map[string]*eth.Ethereum),
		Clients:     make(map[string]*ethclient.Client),
		rollupNodes: make(map[string]*rollupNode.OpNode),
	}
	didErrAfterStart := false
	defer func() {
		if didErrAfterStart {
			for _, node := range sys.rollupNodes {
				node.Stop()
			}
			for _, node := range sys.nodes {
				node.Close()
			}
		}
	}()

	// Wallet
	wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
	if err != nil {
		return nil, fmt.Errorf("Failed to create wallet: %w", err)
	}
	sys.wallet = wallet

	// Create the BSS and set it's config here because it needs to be derived from the accounts
	bssPrivKey, err := wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: cfg.BatchSubmitterHDPath,
		},
	})
	if err != nil {
		return nil, err
	}
	submitterAddress := crypto.PubkeyToAddress(bssPrivKey.PublicKey)

	// Genesis
	eth := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	l1Alloc := precompileAlloc()
	l2Alloc := precompileAlloc()

	for path, amt := range cfg.Premine {
		balance := big.NewInt(int64(amt))
		balance.Mul(balance, eth)
		addr := deriveAddress(wallet, path)
		l1Alloc[addr] = core.GenesisAccount{Balance: balance}
		l2Alloc[addr] = core.GenesisAccount{Balance: balance}
	}

	l1Alloc[cfg.DepositContractAddress] = core.GenesisAccount{Code: common.FromHex(deposit.DepositDeployedBin), Balance: common.Big0}
	l2Alloc[cfg.L1InfoPredeployAddress] = core.GenesisAccount{Code: common.FromHex(l1block.L1blockDeployedBin), Balance: common.Big0}

	genesisTimestamp := uint64(time.Now().Unix())

	l1Genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             cfg.L1ChainID,
			HomesteadBlock:      common.Big0,
			EIP150Block:         common.Big0,
			EIP155Block:         common.Big0,
			EIP158Block:         common.Big0,
			ByzantiumBlock:      common.Big0,
			ConstantinopleBlock: common.Big0,
			PetersburgBlock:     common.Big0,
			IstanbulBlock:       common.Big0,
			BerlinBlock:         common.Big0,
			LondonBlock:         common.Big0,
			Clique: &params.CliqueConfig{
				Period: 2,
				Epoch:  30000,
			},
		},
		Alloc:      l1Alloc,
		Difficulty: common.Big1,
		ExtraData:  cliqueExtraData(wallet, []string{cfg.CliqueSignerDerivationPath}),
		GasLimit:   5000000,
		Nonce:      4660,
		Timestamp:  genesisTimestamp,
		BaseFee:    big.NewInt(7),
	}
	l2Genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:                 cfg.L2ChainID,
			HomesteadBlock:          common.Big0,
			EIP150Block:             common.Big0,
			EIP155Block:             common.Big0,
			EIP158Block:             common.Big0,
			ByzantiumBlock:          common.Big0,
			ConstantinopleBlock:     common.Big0,
			PetersburgBlock:         common.Big0,
			IstanbulBlock:           common.Big0,
			BerlinBlock:             common.Big0,
			LondonBlock:             common.Big0,
			MergeForkBlock:          common.Big0,
			TerminalTotalDifficulty: common.Big0,
		},
		Alloc:      l2Alloc,
		Difficulty: common.Big1,
		GasLimit:   5000000,
		Nonce:      4660,
		// must be equal (or higher, while within bounds) as the L1 anchor point of the rollup
		Timestamp: genesisTimestamp,
		BaseFee:   big.NewInt(7),
	}

	// Initialize nodes
	l1Node, l1Backend, err := initL1Geth(&cfg, wallet, l1Genesis)
	if err != nil {
		return nil, err
	}
	sys.nodes["l1"] = l1Node
	sys.backends["l1"] = l1Backend
	for name, l2Cfg := range cfg.Nodes {
		node, backend, err := initL2Geth(name, l2Cfg.L2EngineAddrs[0], cfg.L2ChainID, l2Genesis)
		if err != nil {
			return nil, err
		}
		sys.nodes[name] = node
		sys.backends[name] = backend
	}

	// Start
	err = l1Node.Start()
	if err != nil {
		didErrAfterStart = true
		return nil, err
	}
	err = l1Backend.StartMining(1)
	if err != nil {
		didErrAfterStart = true
		return nil, err
	}
	for name, node := range sys.nodes {
		if name == "l1" {
			continue
		}
		err = node.Start()
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
	}

	// Geth Clients
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	l1Client, err := ethclient.DialContext(ctx, fmt.Sprintf("ws://%s:%d", cfg.L1WsAddr, cfg.L1WsPort))
	if err != nil {
		didErrAfterStart = true
		return nil, err
	}
	sys.Clients["l1"] = l1Client
	for name, node := range sys.nodes {
		client, err := ethclient.DialContext(ctx, node.WSEndpoint())
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		sys.Clients[name] = client
	}

	// Rollup Genesis
	l1GenesisID, _ := getGenesisInfo(l1Client)
	var l2Client *ethclient.Client
	for name, client := range sys.Clients {
		if name != "l1" {
			l2Client = client
			break
		}
	}
	l2GenesisID, l2GenesisTime := getGenesisInfo(l2Client)

	sys.RolupGenesis = rollup.Genesis{
		L1:     l1GenesisID,
		L2:     l2GenesisID,
		L2Time: l2GenesisTime,
	}

	sys.cfg.RollupConfig.Genesis = sys.RolupGenesis
	sys.cfg.RollupConfig.BatchSenderAddress = submitterAddress

	// Rollup nodes
	for name, nodeConfig := range cfg.Nodes {
		c := nodeConfig
		c.Rollup = sys.cfg.RollupConfig
		if c.Sequencer {
			c.SubmitterPrivKey = bssPrivKey
		}

		node, err := rollupNode.New(context.Background(), &c, cfg.Loggers[name], "")
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		err = node.Start(context.Background())
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		sys.rollupNodes[name] = node

	}

	return sys, nil
}
