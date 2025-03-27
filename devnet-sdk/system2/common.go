package system2

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

type Common interface {
	Logger() log.Logger
	require() *require.Assertions
}

// CommonConfig provides common inputs for creating a new component
type CommonConfig struct {
	Log log.Logger
	T   T
}

type commonImpl struct {
	log log.Logger
	t   T
	req *require.Assertions
}

var _ Common = (*commonImpl)(nil)

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
