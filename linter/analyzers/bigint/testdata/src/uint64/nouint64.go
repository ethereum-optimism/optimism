package a

import "math/big"

type MyInt struct{}

func (m MyInt) Uint64() uint64 { return 0 }

func checkUint64() {
	var value big.Int
	_ = value.Uint64() // want "calling Uint64 on big.Int is forbidden"

	ptr := new(big.Int)
	_ = ptr.Uint64() // want "calling Uint64 on big.Int is forbidden"

	_ = big.NewInt(1).Uint64() // want "calling Uint64 on big.Int is forbidden"

	var custom MyInt
	_ = custom.Uint64()
}
