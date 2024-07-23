package dial

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// GasPriceOracleInterface is an interface for providing
// an GasPriceOracle pre-deployment contract
type GasPriceOracleInterface interface {
	BlobBaseFee(opts *bind.CallOpts) (*big.Int, error)
}
