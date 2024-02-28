package l2proposer

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

// TODO: requires health-check at start
// replaces active proposer (if temp / controlled)
// matches existing proposer (if handsoff)
