package dsl

import (
	"math/big"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
)

type AnchorStateRegistry struct {
	commonImpl
	contract bindings.AnchorStateRegistry
	l1EL     *L1ELNode
}

func NewAnchorStateRegistry(t devtest.T, l2Network *L2Network, l1EL *L1ELNode) *AnchorStateRegistry {
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.DepositContractAddr()),
		bindings.WithTest(t))
	registryAddress, err := contractio.Read(portal.AnchorStateRegistry(), t.Ctx())
	t.Require().NoError(err, "failed to discover AnchorStateRegistry through OptimismPortal2")

	registry := bindings.NewBindings[bindings.AnchorStateRegistry](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(registryAddress),
		bindings.WithTest(t))
	return &AnchorStateRegistry{
		commonImpl: commonFromT(t),
		contract:   registry,
		l1EL:       l1EL,
	}
}

func (a *AnchorStateRegistry) WaitForAnchorGame(expectedGameType gameTypes.GameType, minimumSequence uint64) {
	minimumSequenceBig := new(big.Int).SetUint64(minimumSequence)
	var boundGameAddress common.Address
	var boundGame bindings.FaultDisputeGame

	a.require.Eventually(func() bool {
		gameAddress, err := contractio.Read(a.contract.AnchorGame(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor game", "err", err)
			return false
		}
		if gameAddress == (common.Address{}) {
			a.log.Info("Waiting for anchor game")
			return false
		}
		if gameAddress != boundGameAddress {
			boundGameAddress = gameAddress
			boundGame = bindings.NewBindings[bindings.FaultDisputeGame](
				bindings.WithClient(a.l1EL.EthClient()),
				bindings.WithTo(gameAddress),
				bindings.WithTest(a.t))
		}

		gameType, err := contractio.Read(boundGame.GameType(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor game type", "game", gameAddress, "err", err)
			return false
		}
		gameRoot, err := contractio.Read(boundGame.RootClaim(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor game root", "game", gameAddress, "err", err)
			return false
		}
		gameSequence, err := contractio.Read(boundGame.L2SequenceNumber(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor game L2 sequence number", "game", gameAddress, "err", err)
			return false
		}
		anchor, err := contractio.Read(a.contract.GetAnchorRoot(), a.ctx)
		if err != nil {
			a.log.Debug("Failed to read anchor root", "err", err)
			return false
		}

		a.log.Info("Observed anchor game",
			"game", gameAddress,
			"gameType", gameTypes.GameType(gameType),
			"root", anchor.Root,
			"l2SequenceNumber", anchor.L2SequenceNumber)
		return gameTypes.GameType(gameType) == expectedGameType &&
			minimumSequenceBig.Cmp(gameSequence) <= 0 &&
			anchor.Root == gameRoot &&
			anchor.L2SequenceNumber.Cmp(gameSequence) == 0
	}, 2*time.Minute, 5*time.Second, "AnchorStateRegistry did not advance to a matching anchor game")
}
