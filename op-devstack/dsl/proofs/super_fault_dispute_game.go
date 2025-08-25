package proofs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

type SuperFaultDisputeGame struct {
	*FaultDisputeGame
}

func NewSuperFaultDisputeGame(t devtest.T, require *require.Assertions, addr common.Address, helperProvider gameHelperProvider, game *bindings.FaultDisputeGame) *SuperFaultDisputeGame {
	fdg := NewFaultDisputeGame(t, require, addr, helperProvider, game)
	return &SuperFaultDisputeGame{
		FaultDisputeGame: fdg,
	}
}
