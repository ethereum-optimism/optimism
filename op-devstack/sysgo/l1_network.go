package sysgo

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1Network struct {
	name      string
	chainID   eth.ChainID
	genesis   *core.Genesis
	blockTime uint64
}

func (n *L1Network) Name() string {
	return n.name
}

func (n *L1Network) ChainID() eth.ChainID {
	return n.chainID
}

func (n *L1Network) ChainConfig() *params.ChainConfig {
	return n.genesis.Config
}
