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
	// Boba Goerli genesis block hash
	BobaGoerliGenesisRoot = "0x36c808dc3bb586c14bebde3ca630a4d49a1fdad0b01d7e58f96f2fcd1aa0003d"

	// Bobabeam
	BobabeamChainId = big.NewInt(1294)
	// Bobabeam genesis gas limit
	BobabeamGenesisGasLimit = 11000000
	// Bobabeam genesis block coinbase
	BobabeamGenesisCoinbase = "0x0000000000000000000000000000000000000000"
	// Bobabeam genesis block extra data
	BobabeamGenesisExtraData = "0x000000000000000000000000000000000000000000000000000000000000000000000398232e2064f896018496b4b44b3d62751f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	// Bobabeam genesis block hash
	BobabeamGenesisRoot = "0xc15008e5d48a63706baa38cc16207be66e7596da0d413367376140f5a2ed4197"

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
	return ""
}
