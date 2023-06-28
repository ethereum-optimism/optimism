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

	// Bobabeam
	BobabeamChainId = big.NewInt(1294)
	// Bobabeam genesis gas limit
	BobabeamGenesisGasLimit = 11000000
	// Bobabeam genesis block coinbase
	BobabeamGenesisCoinbase = "0x0000000000000000000000000000000000000000"
	// Bobabeam genesis block extra data
	BobabeamGenesisExtraData = "000000000000000000000000000000000000000000000000000000000000000000000398232e2064f896018496b4b44b3d62751f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	// Bobabeam genesis root
	BobabeamGenesisRoot = "0xc15008e5d48a63706baa38cc16207be66e7596da0d413367376140f5a2ed4197"
	// Bobabeam genesis block hash
	BobabeamGenesisBlockHash = "0x0f93a829d1e17036ccef8b7477c59fe8ed039b0995690b70ef76e894e70ba6c2"

	// Mainnet
	BobamainChainId = big.NewInt(288)
	// Bobabeam genesis gas limit
	BobamainGenesisGasLimit = 11000000
	// Bobabeam genesis block coinbase
	BobamainGenesisCoinbase = "0x0000000000000000000000000000000000000000"
	// Bobabeam genesis block extra data
	BobamainGenesisExtraData = "0x000000000000000000000000000000000000000000000000000000000000000000000398232e2064f896018496b4b44b3d62751f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	// Bobabeam genesis root
	BobamainGenesisRoot = "0x7ec54492a4504ff1ef3491825cd55e01e5c75409e4287129170e98d4693848ce"
	// Bobabeam genesis block hash
	BobamainGenesisBlockHash = "0xdcd9e6a8f9973eaa62da2874959cb152faeb4fd6929177bd6335a1a16074ef9c"
	// Mainnet L1 BOBA Address
	BobaTokenMainL1Address = "0x42bBFa2e77757C645eeaAd1655E0911a7553Efbc"

	// error
	ErrInvalidChainID = errors.New("invalid chain id")
)

func IsBobaValidChainId(chainId *big.Int) bool {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return true
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return true
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return true
	}
	return false
}

func GetBobaGenesisGasLimit(chainId *big.Int) int {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisGasLimit
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisGasLimit
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobamainGenesisGasLimit
	}
	return 11000000
}

func GetBobaGenesisCoinbase(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisCoinbase
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisCoinbase
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobamainGenesisCoinbase
	}
	return "0x0000000000000000000000000000000000000000"
}

func GetBobaGenesisExtraData(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisExtraData
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisExtraData
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisExtraData
	}
	return ""
}

func GetBobaGenesisRoot(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisRoot
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisRoot
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobamainGenesisRoot
	}
	return ""
}

func GetBobaGenesisHash(chainId *big.Int) string {
	// Boba Goerli
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaGoerliGenesisBlockHash
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return BobabeamGenesisBlockHash
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobamainGenesisBlockHash
	}
	return ""
}

func GetBobaTokenL1Address(chainId *big.Int) string {
	// Boba Goerli L1
	if BobaGoerliChainId.Cmp(chainId) == 0 {
		return BobaTokenGoerliL1Address
	}
	// Bobabeam
	if BobabeamChainId.Cmp(chainId) == 0 {
		return ""
	}
	// Mainnet
	if BobamainChainId.Cmp(chainId) == 0 {
		return BobaTokenMainL1Address
	}
	return "0x0000000000000000000000000000000000000000"
}
