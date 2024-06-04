package compressor

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type FactoryFunc func(Config) (derive.Compressor, error)

const (
	RatioKind  = "ratio"
	ShadowKind = "shadow"
	NoneKind   = "none"

	// CloseOverheadZlib is the number of final bytes a [zlib.Writer] call writes
	// to the output buffer.
	CloseOverheadZlib = 9
)

var Kinds = map[string]FactoryFunc{
	RatioKind:  NewRatioCompressor,
	ShadowKind: NewShadowCompressor,
	NoneKind:   NewNonCompressor,
}

var KindKeys []string

func init() {
	for k := range Kinds {
		KindKeys = append(KindKeys, k)
	}
}
