package l1

import test "github.com/ethereum-optimism/optimism/op-test"

type Settings struct {
	ActiveFork L1Fork
	Kind       test.BackendKind
}

type Option interface {
	Apply(settings *Settings) error
}

type OptionFn func(settings *Settings) error

func (fn OptionFn) Apply(settings *Settings) error {
	return fn(settings)
}

func ActiveFork(fork L1Fork) Option {
	return OptionFn(func(settings *Settings) error {
		settings.ActiveFork = fork
		return nil
	})
}

func Kind(kind test.BackendKind) Option {
	return OptionFn(func(settings *Settings) error {
		settings.Kind = kind
		return nil
	})
}

// TODO schedule-hardfork option

// TODO reservation-type option:
// hands-off: work against any system, no config changes
// temp: temporary config changes
// controlled: breaking setup or operation changes
