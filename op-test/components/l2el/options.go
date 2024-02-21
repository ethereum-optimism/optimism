package l2el

import (
	test "github.com/ethereum-optimism/optimism/op-test"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
)

type Settings struct {
	L2 l2.L2

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

func L2Chain(ch l2.L2) Option {
	return OptionFn(func(settings *Settings) error {
		settings.L2 = ch
		return nil
	})
}

// TODO config stuff like pending-gas-limit, archive-rpc, etc. for legacy tests
