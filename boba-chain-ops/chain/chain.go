package chain

import (
	"errors"
	"math/big"
)

var (
	// Boba Goerli
	BobaGoerliChainId = big.NewInt(2888)
	// Boba Goerli genesis gas limit
	BobaGoerliGenesisGasLimit = 11000000
	// Boba Goerli genesis block coinbase
	BobaGoerliGenesisCoinbase = "0x0000000000000000000000000000000000000000"
	// Boba Goerli genesis block extra data
	BobaGoerliGenesisExtraData = "000000000000000000000000000000000000000000000000000000000000000000000398232e2064f896018496b4b44b3d62751f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	// Boba Goerli genesis root
	BobaGoerliGenesisRoot = "0x36c808dc3bb586c14bebde3ca630a4d49a1fdad0b01d7e58f96f2fcd1aa0003d"
	// Boba Goerli genesis block hash
	BobaGoerliGenesisBlockHash = "0xde36bac664c1215f9a7d87cddd3745594b351d3464e8a624e322eddd59ccacf3"
	// Goerli L1 BOBA Address
	BobaTokenGoerliL1Address = "0xeCCD355862591CBB4bB7E7dD55072070ee3d0fC1"

	// Boba Mainnet
	BobaMainnetChainId = big.NewInt(288)
	// Boba Mainnet genesis gas limit
	BobaMainnetGenesisGasLimit = 11000000
	// Boba Mainnet genesis block coinbase
	BobaMainnetGenesisCoinbase = "0x0000000000000000000000000000000000000000"
	// Boba Mainnet genesis block extra data
	BobaMainnetGenesisExtraData = "000000000000000000000000000000000000000000000000000000000000000000000398232e2064f896018496b4b44b3d62751f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	// Boba Mainnet genesis root
	BobaMainnetGenesisRoot = "0x7ec54492a4504ff1ef3491825cd55e01e5c75409e4287129170e98d4693848ce"
	// Boba Mainnet genesis block hash
	BobaMainnetGenesisBlockHash = "0xdcd9e6a8f9973eaa62da2874959cb152faeb4fd6929177bd6335a1a16074ef9c"
	// Mainnet L1 BOBA Address
	BobaTokenMainnetL1Address = "0x42bBFa2e77757C645eeaAd1655E0911a7553Efbc"

	// error
	ErrInvalidChainID = errors.New("invalid chain id")
)

func IsBobaValidChainId(chainId *big.Int) bool {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return true
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return true
	}
	return false
}

func GetBobaGenesisGasLimit(chainId *big.Int) int {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisGasLimit
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaMainnetGenesisGasLimit
	}
	return 11000000
}

func GetBobaGenesisCoinbase(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisCoinbase
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaMainnetGenesisCoinbase
	}
	return "0x0000000000000000000000000000000000000000"
}

func GetBobaGenesisExtraData(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisExtraData
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaMainnetGenesisExtraData
	}
	return ""
}

func GetBobaGenesisRoot(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisRoot
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaMainnetGenesisRoot
	}
	return ""
}

func GetBobaGenesisHash(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisBlockHash
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaMainnetGenesisBlockHash
	}
	return ""
}

func GetBobaTokenL1Address(chainId *big.Int) string {
	// Boba Goerli L1
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaTokenGoerliL1Address
	}
	// Mainnet
	if BobaMainnetChainId.Cmp(chainId) == 0 {
		return BobaTokenMainnetL1Address
	}
	return "0x0000000000000000000000000000000000000000"
}
