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
// Logger is explicitly omitted as each component should have additional context applied to its logging.
type common struct {
	// Ctx is the context for test execution.
	ctx context.Context
	// T is a minimal test interface for panic-checks / assertions.
	t stack.T
	// Require is a helper around the above T, ready to assert against.
	require *require.Assertions
}

type System struct {
	common
	log log.Logger
	sys stack.System
}

func (s *System) Supervisor(id stack.SupervisorID) *Supervisor {
	super := s.sys.Supervisor(id)
	return newSupervisor(s.common, s.log.New("component", "supervisor"), super)
}

func Hydrate(t stack.T, setup *stack.Setup) *System {
	return &System{
		common: common{
			ctx:     setup.Ctx,
			t:       t,
			require: require.New(t),
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
