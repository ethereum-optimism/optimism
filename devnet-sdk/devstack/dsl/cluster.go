package dsl

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

// Cluster wraps a stack.Cluster interface for DSL operations
type Cluster struct {
	commonImpl
	inner stack.Cluster
}

// NewCluster creates a new Cluster DSL wrapper
func NewCluster(inner stack.Cluster) *Cluster {
	return &Cluster{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *Cluster) String() string {
	return c.inner.ID().String()
}

// Escape returns the underlying stack.Cluster
func (c *Cluster) Escape() stack.Cluster {
	return c.inner
}
