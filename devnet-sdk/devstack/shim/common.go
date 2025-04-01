package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

// CommonConfig provides common inputs for creating a new component
type CommonConfig struct {
	Log log.Logger
	T   stack.T
}

// CommonConfigFromSetup is a convenience method to build the config common between all components.
// Note that component constructors will decorate the logger with metadata for internal use,
// the caller of the component constructor can generally leave the logger as-is.
func CommonConfigFromSetup(setup *stack.Setup) CommonConfig {
	return CommonConfig{
		Log: setup.Log,
		T:   setup.T,
	}
}

type commonImpl struct {
	log log.Logger
	t   stack.T
	req *require.Assertions
}

var _ interface {
	stack.Common
	require() *require.Assertions
} = (*commonImpl)(nil)

// newCommon creates an object to hold on to common component data, safe to embed in other structs
func newCommon(cfg CommonConfig) commonImpl {
	return commonImpl{
		log: cfg.Log,
		t:   cfg.T,
		req: require.New(cfg.T),
	}
}

func (c *commonImpl) Logger() log.Logger {
	return c.log
}

func (c *commonImpl) require() *require.Assertions {
	return c.req
}
