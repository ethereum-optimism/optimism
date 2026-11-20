// Package cfastlz wraps the C FastLZ library, the original implementation that fees.FlzCompressLen,
// Solady's LibZip and op-revm port, so tests can compare them with it.
package cfastlz

// #include "fastlz.h"
import "C"

import "unsafe"

// Compress compresses input with FastLZ level 1, the level the ports implement. The library is
// vendored from https://github.com/ariya/FastLZ/commit/344eb4025f9ae866ebf7a2ec48850f7113a97a42,
// the commit Solady's LibZip follows; the compression ratio varies between FastLZ versions.
// Inputs under the 16-byte minimum that fastlz.h documents come out of level 1 as literal runs,
// which the ports reproduce.
func Compress(input []byte) []byte {
	if len(input) == 0 {
		return nil
	}
	// fastlz.h asks for an output buffer at least 5% larger than the input and no smaller than
	// 66 bytes.
	out := make([]byte, max(66, 2*len(input)))
	n := C.fastlz_compress_level(1, unsafe.Pointer(&input[0]), C.int(len(input)), unsafe.Pointer(&out[0]))
	return out[:n]
}
