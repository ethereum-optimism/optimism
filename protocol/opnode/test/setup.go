package test

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/l2os/bindings/l2oo"
	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/deposit"
	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/l1block"
	rollupNode "github.com/ethereum-optimism/optimistic-specs/opnode/node"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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

type L2OOContractConfig struct {
	SubmissionFrequency   *big.Int
	L2StartTime           *big.Int
	L2BlockTime           *big.Int
	GenesisL2Output       [32]byte
	HistoricalTotalBlocks *big.Int
}

type DepositContractConfig struct {
	L2Oracle           common.Address
	FinalizationPeriod *big.Int
}

type SystemConfig struct {
	Mnemonic                   string
	Premine                    map[string]int // Derivation path -> amount in ETH (not wei)
	CliqueSignerDerivationPath string
	L2OutputHDPath             string
	BatchSubmitterHDPath       string
	DeployerHDPath             string
	L1InfoPredeployAddress     common.Address

	L2OOCfg    L2OOContractConfig
	DepositCFG DepositContractConfig

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
	nodes               map[string]*node.Node
	backends            map[string]*eth.Ethereum
	Clients             map[string]*ethclient.Client
	RolupGenesis        rollup.Genesis
	rollupNodes         map[string]*rollupNode.OpNode
	L2OOContractAddr    common.Address
	DepositContractAddr common.Address
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
	batchSubmitterAddr := crypto.PubkeyToAddress(bssPrivKey.PublicKey)

	// Create the L2 Outputsubmitter Address and set it here because it needs to be derived from the accounts
	l2OOSubmitter, err := wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: cfg.L2OutputHDPath,
		},
	})
	if err != nil {
		return nil, err
	}
	l2OutputSubmitterAddr := crypto.PubkeyToAddress(l2OOSubmitter.PublicKey)

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
	sys.cfg.RollupConfig.BatchSenderAddress = batchSubmitterAddr
	sys.cfg.L2OOCfg.L2StartTime = new(big.Int).SetUint64(l2GenesisTime)

	// Deploy Deposit Contract
	deployerPrivKey, err := sys.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: sys.cfg.DeployerHDPath,
		},
	})
	if err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(deployerPrivKey, cfg.L1ChainID)
	if err != nil {
		return nil, err
	}

	// Deploy contracts
	sys.L2OOContractAddr, _, _, err = l2oo.DeployL2OutputOracle(
		opts,
		l1Client,
		sys.cfg.L2OOCfg.SubmissionFrequency,
		sys.cfg.L2OOCfg.L2BlockTime,
		sys.cfg.L2OOCfg.GenesisL2Output,
		sys.cfg.L2OOCfg.HistoricalTotalBlocks,
		sys.cfg.L2OOCfg.L2StartTime,
		l2OutputSubmitterAddr,
	)
	sys.cfg.DepositCFG.L2Oracle = sys.L2OOContractAddr
	if err != nil {
		return nil, err
	}
	var tx *types.Transaction
	sys.DepositContractAddr, tx, _, err = deposit.DeployOptimismPortal(
		opts,
		l1Client,
		sys.cfg.DepositCFG.L2Oracle,
		sys.cfg.DepositCFG.FinalizationPeriod,
	)
	if err != nil {
		return nil, err
	}
	_, err = waitForTransaction(tx.Hash(), l1Client, 4*time.Second)
	if err != nil {
		return nil, fmt.Errorf("waiting for OptimismPortal: %w", err)
	}

	// Rollup nodes
	for name, nodeConfig := range cfg.Nodes {
		c := nodeConfig
		c.Rollup = sys.cfg.RollupConfig
		c.Rollup.DepositContractAddress = sys.DepositContractAddr
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
