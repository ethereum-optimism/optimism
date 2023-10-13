package fault

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
)

var (
	cannonGameType   = uint8(0)
	alphabetGameType = uint8(255)
)

type Registry interface {
	RegisterGameType(gameType uint8, creator scheduler.PlayerCreator)
}

func RegisterGameTypes(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	client bind.ContractCaller) {
	creator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client)
	}
	switch cfg.TraceType {
	case config.TraceTypeCannon:
		registry.RegisterGameType(cannonGameType, creator)
	case config.TraceTypeOutputCannon:
		registry.RegisterGameType(cannonGameType, creator)
	case config.TraceTypeAlphabet:
		registry.RegisterGameType(alphabetGameType, creator)
	}
}
