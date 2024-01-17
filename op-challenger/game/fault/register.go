package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
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
	cannonGameType   = uint8(0)
	alphabetGameType = uint8(255)
)

type CloseFunc func()

type Registry interface {
	RegisterGameType(gameType uint8, creator scheduler.PlayerCreator, oracle types.LargePreimageOracle)
}

func RegisterGameTypes(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	rollupClient outputs.OutputRollupClient,
	txMgr txmgr.TxManager,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
) (CloseFunc, error) {
	var closer CloseFunc
	var l2Client *ethclient.Client
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		l2, err := ethclient.DialContext(ctx, cfg.CannonL2)
		if err != nil {
			return nil, fmt.Errorf("dial l2 client %v: %w", cfg.CannonL2, err)
		}
		l2Client = l2
		closer = l2Client.Close
	}
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		if err := registerCannon(registry, ctx, logger, m, cfg, rollupClient, txMgr, gameFactory, caller, l2Client); err != nil {
			return nil, fmt.Errorf("failed to register cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		if err := registerAlphabet(registry, ctx, logger, m, rollupClient, txMgr, gameFactory, caller); err != nil {
			return nil, fmt.Errorf("failed to register alphabet game type: %w", err)
		}
	}
	return closer, nil
}

func registerAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	rollupClient outputs.OutputRollupClient,
	txMgr txmgr.TxManager,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
) error {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
		if err != nil {
			return nil, err
		}
		prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, err
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			accessor, err := outputs.NewOutputAlphabetTraceAccessor(logger, m, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		genesisValidator := NewPrestateValidator(contract.GetGenesisOutputRoot, prestateProvider)
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, []Validator{prestateValidator, genesisValidator}, creator)
	}
	oracle, err := createOracle(ctx, gameFactory, caller)
	if err != nil {
		return err
	}
	registry.RegisterGameType(alphabetGameType, playerCreator, oracle)
	return nil
}

func createOracle(ctx context.Context, gameFactory *contracts.DisputeGameFactoryContract, caller *batching.MultiCaller) (*contracts.PreimageOracleContract, error) {
	implAddr, err := gameFactory.GetGameImpl(ctx, alphabetGameType)
	if err != nil {
		return nil, fmt.Errorf("failed to load alphabet game implementation: %w", err)
	}
	contract, err := contracts.NewFaultDisputeGameContract(implAddr, caller)
	if err != nil {
		return nil, err
	}
	oracle, err := contract.GetOracle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load oracle address: %w", err)
	}
	return oracle, nil
}

func registerCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	rollupClient outputs.OutputRollupClient,
	txMgr txmgr.TxManager,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource,
) error {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
		if err != nil {
			return nil, err
		}
		prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			splitDepth, err := contract.GetSplitDepth(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load split depth: %w", err)
			}
			accessor, err := outputs.NewOutputCannonTraceAccessor(logger, m, cfg, l2Client, contract, prestateProvider, rollupClient, dir, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		genesisValidator := NewPrestateValidator(contract.GetGenesisOutputRoot, prestateProvider)
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, []Validator{prestateValidator, genesisValidator}, creator)
	}
	oracle, err := createOracle(ctx, gameFactory, caller)
	if err != nil {
		return err
	}
	registry.RegisterGameType(cannonGameType, playerCreator, oracle)
	return nil
}
