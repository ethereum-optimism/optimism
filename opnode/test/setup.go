package test

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/deposit"
	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/l1block"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
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
	account, err := w.Derive(derivPath, false)
	if err != nil {
		panic(err)
	}
	return account
}

// gethConfig is configuration for either and L1 or L2 geth node
type gethConfig struct {
	nodeConfig *node.Config
	ethConfig  *ethconfig.Config
}

// systemConfig holds the information necessary to create a L1 <-> Rollup <-> L2 system
type systemConfig struct {
	mnemonic                string
	l1                      gethConfig
	l2                      gethConfig
	premine                 map[string]int // Derivation path -> amount in ETH (not wei)
	cliqueSigners           []string       // derivation path
	depositContractAddress  string
	l1InforPredeployAddress string
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

// initializeGenesis creates a L1 and L2 genesis from the config and places them in l1 and l2 configurations
func initializeGenesis(cfg *systemConfig) {
	wallet, err := hdwallet.NewFromMnemonic(cfg.mnemonic)
	if err != nil {
		panic(fmt.Errorf("Failed to create wallet: %w", err))
	}

	eth := new(big.Int)
	eth = eth.Exp(big.NewInt(10), big.NewInt(10), nil)

	l1Alloc := precompileAlloc()
	l2Alloc := precompileAlloc()

	for path, amt := range cfg.premine {
		balance := big.NewInt(int64(amt))
		balance.Mul(balance, eth)
		addr := deriveAddress(wallet, path)
		l1Alloc[addr] = core.GenesisAccount{Balance: balance}
		l2Alloc[addr] = core.GenesisAccount{Balance: balance}
	}

	l1Alloc[common.HexToAddress(cfg.depositContractAddress)] = core.GenesisAccount{Code: common.FromHex(deposit.DepositDeployedBin), Balance: common.Big0}
	l2Alloc[common.HexToAddress(cfg.l1InforPredeployAddress)] = core.GenesisAccount{Code: common.FromHex(l1block.L1blockDeployedBin), Balance: common.Big0}

	genesisTimestamp := uint64(time.Now().Unix())

	l1Genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             new(big.Int).SetUint64((cfg.l1.ethConfig.NetworkId)),
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
		ExtraData:  cliqueExtraData(wallet, cfg.cliqueSigners),
		GasLimit:   5000000,
		Nonce:      4660,
		Timestamp:  genesisTimestamp,
		BaseFee:    big.NewInt(7),
	}
	l2Genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:                 new(big.Int).SetUint64((cfg.l2.ethConfig.NetworkId)),
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
		Timestamp:  genesisTimestamp,
		BaseFee:    big.NewInt(7),
	}

	cfg.l1.ethConfig.Genesis = l1Genesis
	cfg.l2.ethConfig.Genesis = l2Genesis

}
