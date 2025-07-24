package proofs

import (
	"github.com/stretchr/testify/require"

	"github.com/HashKeyChain/verse/op-devstack/devtest"
	"github.com/HashKeyChain/verse/op-service/txintent/bindings"
)

type SuperFaultDisputeGame struct {
	*FaultDisputeGame
}

func NewSuperFaultDisputeGame(t devtest.T, require *require.Assertions, game *bindings.FaultDisputeGame) *SuperFaultDisputeGame {
	fdg := NewFaultDisputeGame(t, require, game)
	return &SuperFaultDisputeGame{
		FaultDisputeGame: fdg,
	}
}
