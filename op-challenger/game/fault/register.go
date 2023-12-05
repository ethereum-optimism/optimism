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
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	cannonGameType         = uint8(0)
	outputCannonGameType   = uint8(1)
	outputAlphabetGameType = uint8(254)
	alphabetGameType       = uint8(255)
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
	caller *batching.MultiCaller,
) (CloseFunc, error) {
	var closer CloseFunc
	var l2Client *ethclient.Client
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) || cfg.TraceTypeEnabled(config.TraceTypeOutputCannon) {
		l2, err := ethclient.DialContext(ctx, cfg.CannonL2)
		if err != nil {
			return nil, fmt.Errorf("dial l2 client %v: %w", cfg.CannonL2, err)
		}
		l2Client = l2
		closer = l2Client.Close
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputCannon) {
		creator := newOutputCannonCreator(ctx, logger, m, cfg, txMgr, caller, l2Client)
		registry.RegisterGameType(outputCannonGameType, creator)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputAlphabet) {
		creator := newOutputAlphabetCreator(ctx, logger, m, cfg, txMgr, caller)
		registry.RegisterGameType(outputAlphabetGameType, creator)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		creator := newCannonCreator(ctx, logger, m, cfg, txMgr, caller, l2Client)
		registry.RegisterGameType(cannonGameType, creator)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		creator := newAlphabetCreator(ctx, logger, m, cfg, txMgr, caller)
		registry.RegisterGameType(alphabetGameType, creator)
	}
	return closer, nil
}

func newOutputAlphabetCreator(
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller) scheduler.PlayerCreator {
	return func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewOutputBisectionGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
			if err != nil {
				return nil, err
			}
			splitDepth, err := contract.GetSplitDepth(ctx)
			if err != nil {
				return nil, err
			}
			prestateProvider, err := outputs.NewPrestateProvider(ctx, logger, cfg.RollupRpc, prestateBlock)
			if err != nil {
				return nil, err
			}
			if err := ValidateGenesisOutputRoot(ctx, prestateProvider, contract.GetGenesisOutputRoot); err != nil {
				return nil, err
			}
			if err := ValidateAbsolutePrestate(ctx, prestateProvider, contract.GetAbsolutePrestateHash); err != nil {
				return nil, err
			}
			accessor, err := outputs.NewOutputAlphabetTraceAccessor(ctx, logger, prestateProvider, m, cfg, gameDepth, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, creator)
	}
}

func newOutputCannonCreator(
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) scheduler.PlayerCreator {
	return func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewOutputBisectionGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
			if err != nil {
				return nil, err
			}
			splitDepth, err := contract.GetSplitDepth(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load split depth: %w", err)
			}
			prestateProvider, err := outputs.NewPrestateProvider(ctx, logger, cfg.RollupRpc, prestateBlock)
			if err != nil {
				return nil, err
			}
			if err := ValidateGenesisOutputRoot(ctx, prestateProvider, contract.GetGenesisOutputRoot); err != nil {
				return nil, err
			}
			if err := ValidateAbsolutePrestate(ctx, prestateProvider, contract.GetAbsolutePrestateHash); err != nil {
				return nil, err
			}
			accessor, err := outputs.NewOutputCannonTraceAccessor(ctx, logger, prestateProvider, m, cfg, l2Client, contract, dir, gameDepth, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, creator)
	}
}

func newCannonCreator(
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) scheduler.PlayerCreator {
	return func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			localInputs, err := cannon.FetchLocalInputs(ctx, contract, l2Client)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch cannon local inputs: %w", err)
			}
			provider := cannon.NewTraceProvider(logger, m, cfg, faultTypes.NoLocalContext, localInputs, dir, gameDepth)
			if err := ValidateAbsolutePrestate(ctx, provider, contract.GetAbsolutePrestateHash); err != nil {
				return nil, err
			}
			return trace.NewSimpleTraceAccessor(provider), nil
		}
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, creator)
	}
}

func newAlphabetCreator(
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller) scheduler.PlayerCreator {
	return func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			provider := alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
			if err := ValidateAbsolutePrestate(ctx, provider, contract.GetAbsolutePrestateHash); err != nil {
				return nil, err
			}
			return trace.NewSimpleTraceAccessor(provider), nil
		}
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, creator)
	}
}
