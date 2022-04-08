package util

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ToBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
