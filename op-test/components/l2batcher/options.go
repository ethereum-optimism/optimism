package l2batcher

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l2cl"
)

type Settings struct {
	L2CL l2cl.L2CL

	Kind test.BackendKind
}

type Option interface {
	Apply(settings *Settings) error
}

type OptionFn func(settings *Settings) error

func (fn OptionFn) Apply(settings *Settings) error {
	return fn(settings)
}

func Kind(kind test.BackendKind) Option {
	return OptionFn(func(settings *Settings) error {
		settings.Kind = kind
		return nil
	})
}

func L2CL(cl l2cl.L2CL) Option {
	return OptionFn(func(settings *Settings) error {
		settings.L2CL = cl
		return nil
	})
}

// TODO batcher params option

// TODO DA type option

// TODO: requires health-check at start
// replaces active batcher (if temp / controlled)
// matches existing batcher (if handsoff)
