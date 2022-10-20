package eth

import "github.com/ethereum/go-ethereum/common"

type L1ConfigData struct {
	// last L1 data that was applied to the l1Config
	Origin BlockID

	BatcherAddr common.Address
	Overhead    [32]byte
	Scalar      [32]byte
}
