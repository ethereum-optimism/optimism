package test

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
)

// TODO: Make sure these stay up to date
var depositContractByteCode = "0x60806040526004361061001e5760003560e01c8063fa92670c14610023575b600080fd5b61003d6004803603810190610038919061039d565b61003f565b005b8180156100795750600073ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff1614155b156100b0576040517ff98844ef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60003390503273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461010257731111000000000000000000000000000000001111330190505b8573ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad34888888886040516101679594939291906104da565b60405180910390a3505050505050565b6000604051905090565b600080fd5b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006101b68261018b565b9050919050565b6101c6816101ab565b81146101d157600080fd5b50565b6000813590506101e3816101bd565b92915050565b6000819050919050565b6101fc816101e9565b811461020757600080fd5b50565b600081359050610219816101f3565b92915050565b60008115159050919050565b6102348161021f565b811461023f57600080fd5b50565b6000813590506102518161022b565b92915050565b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6102aa82610261565b810181811067ffffffffffffffff821117156102c9576102c8610272565b5b80604052505050565b60006102dc610177565b90506102e882826102a1565b919050565b600067ffffffffffffffff82111561030857610307610272565b5b61031182610261565b9050602081019050919050565b82818337600083830152505050565b600061034061033b846102ed565b6102d2565b90508281526020810184848401111561035c5761035b61025c565b5b61036784828561031e565b509392505050565b600082601f83011261038457610383610257565b5b813561039484826020860161032d565b91505092915050565b600080600080600060a086880312156103b9576103b8610181565b5b60006103c7888289016101d4565b95505060206103d88882890161020a565b94505060406103e98882890161020a565b93505060606103fa88828901610242565b925050608086013567ffffffffffffffff81111561041b5761041a610186565b5b6104278882890161036f565b9150509295509295909350565b61043d816101e9565b82525050565b61044c8161021f565b82525050565b600081519050919050565b600082825260208201905092915050565b60005b8381101561048c578082015181840152602081019050610471565b8381111561049b576000848401525b50505050565b60006104ac82610452565b6104b6818561045d565b93506104c681856020860161046e565b6104cf81610261565b840191505092915050565b600060a0820190506104ef6000830188610434565b6104fc6020830187610434565b6105096040830186610434565b6105166060830185610443565b818103608083015261052881846104a1565b9050969550505050505056fea2646970667358221220267a1955ffafb2cf1004819eb2b511ad2922e2ca0436708e171243ac92ac637e64736f6c634300080b0033"
var l1InfoContractByteCode = "0x608060405234801561001057600080fd5b50600436106100625760003560e01c806309bd5a60146100675780635cf24969146100855780638381f58a146100a3578063b80777ea146100c1578063c03ba43e146100df578063e591b282146100fb575b600080fd5b61006f610119565b60405161007c91906101fd565b60405180910390f35b61008d61011f565b60405161009a9190610231565b60405180910390f35b6100ab610125565b6040516100b89190610231565b60405180910390f35b6100c961012b565b6040516100d69190610231565b60405180910390f35b6100f960048036038101906100f491906102a9565b610131565b005b6101036101cc565b6040516101109190610351565b60405180910390f35b60035481565b60025481565b60005481565b60015481565b73deaddeaddeaddeaddeaddeaddeaddeaddead000173ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101aa576040517fce8c104800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8360008190555082600181905550816002819055508060038190555050505050565b73deaddeaddeaddeaddeaddeaddeaddeaddead000181565b6000819050919050565b6101f7816101e4565b82525050565b600060208201905061021260008301846101ee565b92915050565b6000819050919050565b61022b81610218565b82525050565b60006020820190506102466000830184610222565b92915050565b600080fd5b61025a81610218565b811461026557600080fd5b50565b60008135905061027781610251565b92915050565b610286816101e4565b811461029157600080fd5b50565b6000813590506102a38161027d565b92915050565b600080600080608085870312156102c3576102c261024c565b5b60006102d187828801610268565b94505060206102e287828801610268565b93505060406102f387828801610268565b925050606061030487828801610294565b91505092959194509250565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b600061033b82610310565b9050919050565b61034b81610330565b82525050565b60006020820190506103666000830184610342565b9291505056fea2646970667358221220ef072e1b541577d5392b0cc2b83515eee307c6b7483589377f51e691ef42552e64736f6c634300080b0033"

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

	l1Alloc[common.HexToAddress(cfg.depositContractAddress)] = core.GenesisAccount{Code: common.FromHex(depositContractByteCode), Balance: common.Big0}
	l2Alloc[common.HexToAddress(cfg.l1InforPredeployAddress)] = core.GenesisAccount{Code: common.FromHex(l1InfoContractByteCode), Balance: common.Big0}

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
