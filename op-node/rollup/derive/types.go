package derive

import (
	"fmt"
	"regexp"
)

type CompressionAlgo string

const (
	// compression algo types
	Zlib     CompressionAlgo = "zlib"
	Brotli   CompressionAlgo = "brotli" // default level
	Brotli9  CompressionAlgo = "brotli-9"
	Brotli10 CompressionAlgo = "brotli-10"
	Brotli11 CompressionAlgo = "brotli-11"
)

var CompressionAlgos = []CompressionAlgo{
	Zlib,
	Brotli,
	Brotli9,
	Brotli10,
	Brotli11,
}

var brotliRegexp = regexp.MustCompile(`^brotli(|-(9|10|11))$`)

func (algo CompressionAlgo) String() string {
	return string(algo)
}

func (algo *CompressionAlgo) Set(value string) error {
	if !ValidCompressionAlgo(CompressionAlgo(value)) {
		return fmt.Errorf("unknown compression algo type: %q", value)
	}
	*algo = CompressionAlgo(value)
	return nil
}

func (algo *CompressionAlgo) Clone() any {
	cpy := *algo
	return &cpy
}

func (algo *CompressionAlgo) IsBrotli() bool {
	return brotliRegexp.MatchString(algo.String())
}

func GetBrotliLevel(algo CompressionAlgo) int {
	switch algo {
	case Brotli9:
		return 9
	case Brotli10, Brotli: // make level 10 the default
		return 10
	case Brotli11:
		return 11
	default:
		panic("Unsupported brotli level")
	}
}

func ValidCompressionAlgo(value CompressionAlgo) bool {
	for _, k := range CompressionAlgos {
		if k == value {
			return true
		}
	}
	return false
}
