package fault

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
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
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		registerCannon(registry, ctx, logger, m, cfg, txMgr, client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		registerAlphabet(registry, ctx, logger, m, cfg, txMgr, client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputCannon) {
		registerOutputCannon(registry, ctx, logger, m, cfg, txMgr, client)
	}
}

func registerOutputCannon(registry Registry, ctx context.Context, logger log.Logger, m metrics.Metricer, cfg *config.Config, txMgr txmgr.TxManager, client *ethclient.Client) {
	resourceCreator := func(addr common.Address, gameDepth uint64, dir string) (faultTypes.TraceAccessor, faultTypes.OracleUpdater, absolutePrestateValidator, error) {
		// TODO: Need to be able to get the upper game depth from the contract
		topDepth := gameDepth / 2
		// TODO: Load the block number of the root claim's output root from the contract.
		outputRootBlockNum := uint64(math.MaxUint64)
		outputProvider, err := outputs.NewTraceProvider(ctx, logger, cfg.RollupRpc, topDepth, 0, outputRootBlockNum)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create output root trace provider: %w", err)
		}

		cannonProviderFactory := func(ctx context.Context, pre faultTypes.Claim, post faultTypes.Claim) (faultTypes.TraceProvider, error) {
			providerDir := filepath.Join(dir, fmt.Sprintf("%s-%s", pre.Value.Hex(), post.Value.Hex()))
			preBlockNum, err := outputProvider.BlockNumberAtPosition(pre.Position)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate pre block number: %w", err)
			}
			postBlockNum, err := outputProvider.BlockNumberAtPosition(post.Position)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate post block number: %w", err)
			}
			provider, err := cannon.NewTraceProviderFromOutputRoots(
				ctx,
				logger,
				m,
				cfg,
				client,
				providerDir,
				addr,
				gameDepth-topDepth,
				pre.Value,
				new(big.Int).SetUint64(preBlockNum),
				post.Value,
				new(big.Int).SetUint64(postBlockNum),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create cannon trace provider between claims %v and %v: %w", pre.ContractIndex, post.ContractIndex, err)
			}

			return provider, nil
		}

		traceAccessor := trace.NewSplitTraceAccessor(outputProvider, topDepth, cannonProviderFactory)
		updater, err := cannon.NewOracleUpdater(ctx, logger, txMgr, addr, client)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create oracle updater: %w", err)
		}
		return traceAccessor, updater, noopPrestateValidator, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(cannonGameType, playerCreator)
}

func registerAlphabet(registry Registry, ctx context.Context, logger log.Logger, m metrics.Metricer, cfg *config.Config, txMgr txmgr.TxManager, client *ethclient.Client) {
	resourceCreator := func(addr common.Address, gameDepth uint64, dir string) (faultTypes.TraceAccessor, faultTypes.OracleUpdater, absolutePrestateValidator, error) {
		provider := alphabet.NewTraceProvider(cfg.AlphabetTrace, gameDepth)
		updater := alphabet.NewOracleUpdater(logger)
		return trace.NewSimpleTraceAccessor(provider), updater, newSingleTracePrestateValidator(provider), nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(alphabetGameType, playerCreator)
}

func registerCannon(registry Registry, ctx context.Context, logger log.Logger, m metrics.Metricer, cfg *config.Config, txMgr txmgr.TxManager, client *ethclient.Client) {
	resourceCreator := func(addr common.Address, gameDepth uint64, dir string) (faultTypes.TraceAccessor, faultTypes.OracleUpdater, absolutePrestateValidator, error) {
		provider, err := cannon.NewTraceProvider(ctx, logger, m, cfg, client, dir, addr, gameDepth)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("create cannon trace provider: %w", err)
		}
		updater, err := cannon.NewOracleUpdater(ctx, logger, txMgr, addr, client)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create the cannon updater: %w", err)
		}
		return trace.NewSimpleTraceAccessor(provider), updater, newSingleTracePrestateValidator(provider), nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, cfg, dir, game.Proxy, txMgr, client, resourceCreator)
	}
	registry.RegisterGameType(cannonGameType, playerCreator)
}
