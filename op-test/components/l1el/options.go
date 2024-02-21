package l1el

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type Settings struct {
	Chain l1.L1

	// active block-building, peering, etc. config
	BlockBuilding bool

	Kind test.BackendKind
}

type Option interface {
	Apply(settings *Settings) error
}

type OptionFn func(settings *Settings) error

func (fn OptionFn) Apply(settings *Settings) error {
	return fn(settings)
}

func L1Chain(chain l1.L1) Option {
	return OptionFn(func(settings *Settings) error {
		settings.Chain = chain
		return nil
	})
}

func Kind(kind test.BackendKind) Option {
	return OptionFn(func(settings *Settings) error {
		settings.Kind = kind
		return nil
	})
}
