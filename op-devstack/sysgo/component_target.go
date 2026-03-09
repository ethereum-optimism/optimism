package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ComponentTarget struct {
	Name    string
	ChainID eth.ChainID
}

func NewComponentTarget(name string, chainID eth.ChainID) ComponentTarget {
	return ComponentTarget{
		Name:    name,
		ChainID: chainID,
	}
}

func (t ComponentTarget) String() string {
	if t.ChainID == (eth.ChainID{}) {
		return t.Name
	}
	return fmt.Sprintf("%s-%s", t.Name, t.ChainID)
}
