package fuzzerutils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	fuzz "github.com/google/gofuzz"
)

// AddFuzzerFunctions takes a fuzz.Fuzzer and adds a list of functions to handle different
// data types in a fuzzing campaign. It adds support for commonly used types throughout the
// application.
func AddFuzzerFunctions(fuzzer *fuzz.Fuzzer) {
	fuzzer.Funcs(
		func(e *big.Int, c fuzz.Continue) {
			var temp [32]byte
			c.Fuzz(&temp)
			e.SetBytes(temp[:])
		},
		func(e *common.Hash, c fuzz.Continue) {
			var temp [32]byte
			c.Fuzz(&temp)
			e.SetBytes(temp[:])
		},
		func(e *common.Address, c fuzz.Continue) {
			var temp [20]byte
			c.Fuzz(&temp)
			e.SetBytes(temp[:])
		},
	)
}
