package compressor

import (
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type FactoryFunc func(Config) (derive.Compressor, error)

type FactoryFlag struct {
	FlagValue   string
	FactoryFunc FactoryFunc
}

var (
	// The Ratio Factory creates new RatioCompressor's (see NewRatioCompressor
	// for a description).
	Ratio = FactoryFlag{
		FlagValue:   "ratio",
		FactoryFunc: NewRatioCompressor,
	}
	// The Shadow Factory creates new ShadowCompressor's (see NewShadowCompressor
	// for a description).
	Shadow = FactoryFlag{
		FlagValue:   "shadow",
		FactoryFunc: NewShadowCompressor,
	}
)

var Factories = []FactoryFlag{
	Ratio,
	Shadow,
}

func FactoryFlags() string {
	var out strings.Builder
	for i, v := range Factories {
		out.WriteString(v.FlagValue)
		if i+1 < len(Factories) {
			out.WriteString(", ")
		}
	}
	return out.String()
}
