package fault

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/log"
)

func registerSuperPermissionedGameType(
	ctx context.Context,
	logger log.Logger,
	registry interface {
		RegisterGameType(gameTypes.GameType, scheduler.PlayerCreator)
		RegisterBondContract(gameTypes.GameType, claims.BondContractCreator)
	},
	txSender TxSender,
	gameType gameTypes.GameType,
	rootProvider *sources.SupervisorClient,
	superNodeProvider *sources.SuperNodeClient,
	syncValidator gameTypes.SyncValidator,
	caller *batching.MultiCaller,
	l1HeaderSource generic.L1HeaderSource,
	m metrics.Metricer,
) {
	playerCreator := func(game gameTypes.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		_ = dir
		contract := contracts.NewSuperPermissionedDisputeGameContract(m, game.Proxy, caller)
		return generic.NewGenericGamePlayer(
			ctx,
			logger,
			game.Proxy,
			contract,
			syncValidator,
			nil,
			l1HeaderSource,
			func(_ context.Context, logger log.Logger, l1Head eth.BlockID) (generic.Actor, error) {
				return NewSuperPermissionedActor(logger, txSender, contract, rootProvider, superNodeProvider, l1Head), nil
			},
		)
	}
	registry.RegisterGameType(gameType, playerCreator)
	registry.RegisterBondContract(gameType, func(game gameTypes.GameMetadata) (claims.BondContract, error) {
		return contracts.NewSuperPermissionedDisputeGameContract(m, game.Proxy, caller), nil
	})
}
