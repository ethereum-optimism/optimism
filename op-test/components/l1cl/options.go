package l1cl

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l1el"
)

type Settings struct {
	el l1el.L1EL

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

func Kind(kind test.BackendKind) Option {
	return OptionFn(func(settings *Settings) error {
		settings.Kind = kind
		return nil
	})
}

func L1EL(el l1el.L1EL) Option {
	return OptionFn(func(settings *Settings) error {
		settings.el = el
		return nil
	})
}

func BlockBuilding(v bool) Option {
	return OptionFn(func(settings *Settings) error {
		settings.BlockBuilding = v
		return nil
	})
}
