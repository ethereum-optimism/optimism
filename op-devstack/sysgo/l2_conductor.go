package sysgo

import (
	opconductor "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Conductor struct {
	name    string
	chainID eth.ChainID

	serverID          string
	consensusEndpoint string
	rpcEndpoint       string
	service           *opconductor.OpConductor
}

func (c *Conductor) ServerID() string {
	return c.serverID
}

func (c *Conductor) ConsensusEndpoint() string {
	return c.consensusEndpoint
}

func (c *Conductor) HTTPEndpoint() string {
	return c.rpcEndpoint
}
