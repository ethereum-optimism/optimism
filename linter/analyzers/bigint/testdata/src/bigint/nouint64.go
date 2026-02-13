package a

import "math/big"

type MyInt struct{}

func (m MyInt) Uint64() uint64 { return 0 }

func checkUint64() {
	var value big.Int
	_ = value.Uint64() // want "use bigs.Uint64Strict instead of big.Int.Uint64"

	ptr := new(big.Int)
	_ = ptr.Uint64() // want "use bigs.Uint64Strict instead of big.Int.Uint64"

	_ = big.NewInt(1).Uint64() // want "use bigs.Uint64Strict instead of big.Int.Uint64"

	var custom MyInt
	_ = custom.Uint64()
}
