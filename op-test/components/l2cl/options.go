package l2cl

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l2el"
)

type Settings struct {
	L2EL l2el.L2EL

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

func L2EL(el l2el.L2EL) Option {
	return OptionFn(func(settings *Settings) error {
		settings.L2EL = el
		return nil
	})
}

// TODO verif conf depth

// TODO sequencer conf depth

// TODO sequencer mode / key
// TODO: requires health-check at start
// replaces active sequencer (if temp / controlled)
// matches existing sequencer (if handsoff)
