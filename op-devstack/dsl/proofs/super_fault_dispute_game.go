package proofs

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

type SuperFaultDisputeGame struct {
	*FaultDisputeGame
}

func NewSuperFaultDisputeGame(t devtest.T, require *require.Assertions, addr common.Address, helperProvider gameHelperProvider, game *bindings.FaultDisputeGame) *SuperFaultDisputeGame {
	honestTraceProvider := func(_ *FaultDisputeGame) types.TraceAccessor {
		require.Fail("Honest trace not supported for super games")
		return nil
	}
	fdg := NewFaultDisputeGame(t, require, addr, helperProvider, honestTraceProvider, game)
	return &SuperFaultDisputeGame{
		FaultDisputeGame: fdg,
	}
}

func (g *SuperFaultDisputeGame) StartingL2SequenceNumber() uint64 {
	return contract.Read(g.game.StartingSequenceNumber())
}
