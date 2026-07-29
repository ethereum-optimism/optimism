package sysgo

import (
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Proposer struct {
	name    string
	chainID eth.ChainID
	service *ps.ProposerService
	userRPC string
}

type ProposerOption func(target ComponentTarget, cfg *ps.CLIConfig)
