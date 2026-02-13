package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

// WeiToEther divides the wei value by 10^18 to get a number in ether as a float64
func WeiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
