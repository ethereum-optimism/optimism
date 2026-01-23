package a

import . "math/big"

func checkDotImport() {
	var value Int
	_ = value.Uint64() // want "use bigs.Uint64Strict instead of big.Int.Uint64"

	_ = NewInt(2).Uint64() // want "use bigs.Uint64Strict instead of big.Int.Uint64"
}
