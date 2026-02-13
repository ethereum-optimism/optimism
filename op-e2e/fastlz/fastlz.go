//go:build cgo_test
// +build cgo_test

package fastlz

// #include <stdlib.h>
// #include "fastlz.h"
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// Compress compresses the input data using the FastLZ algorithm.
// The version of FastLZ used is FastLZ level 1 with the implementation from
// this commit: https://github.com/ariya/FastLZ/commit/344eb4025f9ae866ebf7a2ec48850f7113a97a42
// Which is the same commit that Solady uses: https://github.com/Vectorized/solady/blob/main/src/utils/LibZip.sol#L19
// Note the FastLZ compression ratio does vary between different versions of the library.
func Compress(input []byte) ([]byte, error) {
	length := len(input)
	if length == 0 {
		return nil, errors.New("no input provided")
	}

	result := make([]byte, length*2)
	size := C.fastlz_compress(unsafe.Pointer(&input[0]), C.int(length), unsafe.Pointer(&result[0]))

	runtime.KeepAlive(input)

	if size == 0 {
		return nil, errors.New("error compressing data")
	}

	return result[:size], nil
}
