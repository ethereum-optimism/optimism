package superchain

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type Settings struct {
	L1 l1.L1

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

func L1Chain(ch l1.L1) Option {
	return OptionFn(func(settings *Settings) error {
		settings.L1 = ch
		return nil
	})
}

// TODO superchain feature toggles

// TODO superchain contract settings/constraints
