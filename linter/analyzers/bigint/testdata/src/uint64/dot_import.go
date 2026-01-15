package a

import . "math/big"

func checkDotImport() {
	var value Int
	_ = value.Uint64() // want "calling Uint64 on big.Int is forbidden"

	_ = NewInt(2).Uint64() // want "calling Uint64 on big.Int is forbidden"
}
