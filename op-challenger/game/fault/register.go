package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	client *ethclient.Client,
) {
	multiCaller := batching.NewMultiCaller(client.Client(), batching.DefaultBatchSize)

	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		resourceCreator := func(addr common.Address, contract *contracts.FaultDisputeGameContract, gameDepth uint64, dir string) (faultTypes.TraceAccessor, gameValidator, error) {
			logger := logger.New("game", addr)
			provider, err := cannon.NewTraceProvider(ctx, logger, m, cfg, contract, dir, gameDepth)
			if err != nil {
				return nil, nil, fmt.Errorf("create cannon trace provider: %w", err)
			}
			validator := func(ctx context.Context, contract *contracts.FaultDisputeGameContract) error {
				return ValidateAbsolutePrestate(ctx, provider, contract)
			}
			return trace.NewSimpleTraceAccessor(provider), validator, nil
		}
		playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
			gameContract, err := contracts.NewFaultDisputeGameContract(game.Proxy, multiCaller)
			if err != nil {
				return nil, fmt.Errorf("failed to create fault dispute game contract wrapper: %w", err)
			}
			return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, gameContract, resourceCreator)
		}
		registry.RegisterGameType(cannonGameType, playerCreator)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		resourceCreator := func(addr common.Address, contract *contracts.FaultDisputeGameContract, gameDepth uint64, dir string) (faultTypes.TraceAccessor, gameValidator, error) {
			provider := alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
			validator := func(ctx context.Context, contract *contracts.FaultDisputeGameContract) error {
				return ValidateAbsolutePrestate(ctx, provider, contract)
			}
			return trace.NewSimpleTraceAccessor(provider), validator, nil
		}
		playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
			gameContract, err := contracts.NewFaultDisputeGameContract(game.Proxy, multiCaller)
			if err != nil {
				return nil, fmt.Errorf("failed to create fault dispute game contract wrapper: %w", err)
			}
			return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, gameContract, resourceCreator)
		}
		registry.RegisterGameType(alphabetGameType, playerCreator)
	}
}
