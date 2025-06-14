package controller

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ChainIDProvider interface {
	ChainID() eth.ChainID
}

type chainIDState struct {
	chainID eth.ChainID
}

var _ ChainIDProvider = (*chainIDState)(nil)

func (c *chainIDState) Init(id eth.ChainID) {
	c.chainID = id
}

func (c *chainIDState) ChainID() eth.ChainID {
	return c.chainID
}

func OfChain[V ChainIDProvider](chainID eth.ChainID) Predicate[V] {
	return func(v V) bool {
		return v.ChainID() == chainID
	}
}
