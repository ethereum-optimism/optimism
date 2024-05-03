package derive

import (
	"fmt"
	"regexp"
)

type CompressionAlgo string

const (
	// compression algo types
	Zlib     CompressionAlgo = "zlib"
	Brotli   CompressionAlgo = Brotli10 // default brotli 10
	Brotli9  CompressionAlgo = "brotli-9"
	Brotli10 CompressionAlgo = "brotli-10"
	Brotli11 CompressionAlgo = "brotli-11"
)

var CompressionAlgoTypes = []CompressionAlgo{
	Zlib,
	Brotli,
	Brotli9,
	Brotli10,
	Brotli11,
}

var brotliRegexp = regexp.MustCompile(`^brotli-(9|10|11)$`)

func (kind CompressionAlgo) String() string {
	return string(kind)
}

func (kind *CompressionAlgo) Set(value string) error {
	if !ValidCompressionAlgoType(CompressionAlgo(value)) {
		return fmt.Errorf("unknown compression algo type: %q", value)
	}
	*kind = CompressionAlgo(value)
	return nil
}

func (kind *CompressionAlgo) Clone() any {
	cpy := *kind
	return &cpy
}

func (kind *CompressionAlgo) IsBrotli() bool {
	return brotliRegexp.MatchString(kind.String())
}

func GetBrotliLevel(kind CompressionAlgo) int {
	switch kind {
	case Brotli9:
		return 9
	case Brotli10:
		return 10
	case Brotli11:
		return 11
	default:
		panic("Unsupported brotli level")
	}
}

func ValidCompressionAlgoType(value CompressionAlgo) bool {
	for _, k := range CompressionAlgoTypes {
		if k == value {
			return true
		}
	}
	return false
}
