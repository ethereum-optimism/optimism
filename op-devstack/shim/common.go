package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// CommonConfig provides common inputs for creating a new component
type CommonConfig struct {
	// Log is the logger to use, annotated with metadata.
	// Shim constructors generally add default annotations like the component "id" and "chain"
	Log log.Logger
	T   devtest.T
}

// NewCommonConfig is a convenience method to build the config common between all components.
// Note that component constructors will decorate the logger with metadata for internal use,
// the caller of the component constructor can generally leave the logger as-is.
func NewCommonConfig(t devtest.T) CommonConfig {
	return CommonConfig{
		Log: t.Logger(),
		T:   t,
	}
}

type commonImpl struct {
	log    log.Logger
	t      devtest.T
	req    *require.Assertions
	labels *locks.RWMap[string, string]
}

var _ interface {
	stack.Common
	require() *require.Assertions
} = (*commonImpl)(nil)

// newCommon creates an object to hold on to common component data, safe to embed in other structs
func newCommon(cfg CommonConfig) commonImpl {
	return commonImpl{
		log:    cfg.T.Logger(),
		t:      cfg.T,
		req:    cfg.T.Require(),
		labels: new(locks.RWMap[string, string]),
	}
}

func (c *commonImpl) T() devtest.T {
	return c.t
}

func (c *commonImpl) Logger() log.Logger {
	return c.log
}

func (c *commonImpl) require() *require.Assertions {
	return c.req
}

func (c *commonImpl) Label(key string) string {
	out, _ := c.labels.Get(key)
	return out
}

func (c *commonImpl) SetLabel(key, value string) {
	c.labels.Set(key, value)
}
