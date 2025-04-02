package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const defaultTimeout = 30 * time.Second

// common provides a set of common values and methods inherited by all DSL structs.
// These should be kept very minimal.
// No public methods or fields should be exposed.
type common struct {
	// Ctx is the context for test execution.
	ctx context.Context
	// log is the component-specific logger instance.
	log log.Logger
	// T is a minimal test interface for panic-checks / assertions.
	t stack.T
	// Require is a helper around the above T, ready to assert against.
	require *require.Assertions
}

// commonWithLog copies the specified common, replacing the log instance.
// Not an instance method on common to avoid it being inherited to every component that uses common.
func commonWithLog(c common, log log.Logger) common {
	return common{
		ctx:     c.ctx,
		log:     log,
		t:       c.t,
		require: c.require,
	}
}

type System struct {
	common
	log log.Logger
	sys stack.System
}

func (s *System) Supervisor(id stack.SupervisorID) *Supervisor {
	super := s.sys.Supervisor(id)
	return newSupervisor(commonWithLog(s.common, s.log.New("id", id)), super)
}

func Hydrate(setup *stack.Setup) *System {
	return &System{
		common: common{
			ctx:     setup.Ctx,
			log:     setup.Log,
			t:       setup.T,
			require: setup.Require,
		},
		log: setup.Log,
		sys: setup.System,
	}
}

func applyOpts[Config any](defaultConfig Config, opts ...func(config *Config)) Config {
	for _, opt := range opts {
		opt(&defaultConfig)
	}
	return defaultConfig
}
