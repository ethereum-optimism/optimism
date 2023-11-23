package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	cannonGameType       = uint8(0)
	outputCannonGameType = uint8(0) // TODO(client-pod#43): This should be a unique game type
	alphabetGameType     = uint8(255)
)

type CloseFunc func()

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
) (CloseFunc, error) {
	var closer CloseFunc
	createL2Client := func() (cannon.L2HeaderSource, error) {
		l2Client, err := ethclient.DialContext(ctx, cfg.CannonL2)
		if err != nil {
			return nil, fmt.Errorf("dial l2 client %v: %w", cfg.CannonL2, err)
		}
		closer = l2Client.Close
		return l2Client, nil
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputCannon) {
		l2Client, err := createL2Client()
		if err != nil {
			return nil, err
		}
		registerOutputCannon(registry, ctx, logger, m, cfg, txMgr, client, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		l2Client, err := createL2Client()
		if err != nil {
			return nil, err
		}
		registerCannon(registry, ctx, logger, m, cfg, txMgr, client, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		registerAlphabet(registry, ctx, logger, m, cfg, txMgr, client)
	}
	return closer, nil
}

func registerOutputCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	client *ethclient.Client,
	l2Client cannon.L2HeaderSource) {
	resourceCreator := func(addr common.Address, contract *contracts.FaultDisputeGameContract, gameDepth uint64, dir string) (faultTypes.TraceAccessor, gameValidator, error) {
		logger := logger.New("game", addr)
		// TODO(client-pod#43): Updated contracts should expose this as the pre and post state blocks
		agreed, disputed, err := contract.GetProposals(ctx)
		if err != nil {
			return nil, nil, err
		}
		accessor, err := outputs.NewOutputCannonTraceAccessor(ctx, logger, m, cfg, l2Client, contract, dir, gameDepth, agreed.L2BlockNumber.Uint64(), disputed.L2BlockNumber.Uint64())
		if err != nil {
			return nil, nil, err
		}
		// TODO(client-pod#44): Validate absolute pre-state for split games
		noopValidator := func(ctx context.Context, gameContract *contracts.FaultDisputeGameContract) error {
			return nil
		}
		return accessor, noopValidator, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(outputCannonGameType, playerCreator)
}

func registerCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	client *ethclient.Client,
	l2Client cannon.L2HeaderSource) {
	resourceCreator := func(addr common.Address, contract *contracts.FaultDisputeGameContract, gameDepth uint64, dir string) (faultTypes.TraceAccessor, gameValidator, error) {
		logger := logger.New("game", addr)
		provider, err := cannon.NewTraceProvider(ctx, logger, m, cfg, l2Client, contract, faultTypes.NoLocalContext, dir, gameDepth)
		if err != nil {
			return nil, nil, fmt.Errorf("create cannon trace provider: %w", err)
		}
		validator := func(ctx context.Context, contract *contracts.FaultDisputeGameContract) error {
			return ValidateAbsolutePrestate(ctx, provider, contract)
		}
		return trace.NewSimpleTraceAccessor(provider), validator, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(cannonGameType, playerCreator)
}

func registerAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	client *ethclient.Client) {
	resourceCreator := func(addr common.Address, contract *contracts.FaultDisputeGameContract, gameDepth uint64, dir string) (faultTypes.TraceAccessor, gameValidator, error) {
		provider := alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
		validator := func(ctx context.Context, contract *contracts.FaultDisputeGameContract) error {
			return ValidateAbsolutePrestate(ctx, provider, contract)
		}
		return trace.NewSimpleTraceAccessor(provider), validator, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(alphabetGameType, playerCreator)
}
