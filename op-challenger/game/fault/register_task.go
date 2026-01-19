package fault

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type RegisterTask struct {
	gameType               gameTypes.GameType
	skipPrestateValidation bool

	syncValidator gameTypes.SyncValidator

	getTopPrestateProvider    func(ctx context.Context, prestateBlock uint64) (faultTypes.PrestateProvider, error)
	getBottomPrestateProvider func(ctx context.Context, prestateHash common.Hash) (faultTypes.PrestateProvider, error)
	newTraceAccessor          func(
		logger log.Logger,
		m metrics.Metricer,
		prestateProvider faultTypes.PrestateProvider,
		vmPrestateProvider faultTypes.PrestateProvider,
		dir string,
		l1Head eth.BlockID,
		splitDepth faultTypes.Depth,
		prestateBlock uint64,
		poststateBlock uint64) (*trace.Accessor, error)
}

func NewSuperCannonRegisterTask(gameType gameTypes.GameType, cfg *config.Config, m caching.Metrics, serverExecutor vm.OracleServerExecutor, rootProvider *sources.SupervisorClient, superNodeProvider *sources.SuperNodeClient, syncValidator gameTypes.SyncValidator) *RegisterTask {
	return newSuperCannonVMRegisterTaskWithConfig(gameType, cfg, m, serverExecutor, rootProvider, superNodeProvider, syncValidator, cfg.Cannon, cfg.CannonAbsolutePreStateBaseURL, cfg.CannonAbsolutePreState)
}

func NewSuperCannonKonaRegisterTask(gameType gameTypes.GameType, cfg *config.Config, m caching.Metrics, serverExecutor vm.OracleServerExecutor, rootProvider *sources.SupervisorClient, superNodeProvider *sources.SuperNodeClient, syncValidator gameTypes.SyncValidator) *RegisterTask {
	return newSuperCannonVMRegisterTaskWithConfig(gameType, cfg, m, serverExecutor, rootProvider, superNodeProvider, syncValidator, cfg.CannonKona, cfg.CannonKonaAbsolutePreStateBaseURL, cfg.CannonKonaAbsolutePreState)
}

func newSuperCannonVMRegisterTaskWithConfig(
	gameType gameTypes.GameType,
	cfg *config.Config,
	m caching.Metrics,
	serverExecutor vm.OracleServerExecutor,
	rootProvider *sources.SupervisorClient,
	superNodeProvider *sources.SuperNodeClient,
	syncValidator gameTypes.SyncValidator,
	vmCfg vm.Config,
	preStateBaseURL *url.URL,
	preState string,
) *RegisterTask {
	stateConverter := cannon.NewStateConverter(vmCfg)
	return &RegisterTask{
		gameType:               gameType,
		syncValidator:          syncValidator,
		skipPrestateValidation: gameType == gameTypes.SuperPermissionedGameType,
		getTopPrestateProvider: func(ctx context.Context, prestateTimestamp uint64) (faultTypes.PrestateProvider, error) {
			return super.NewSuperRootPrestateProvider(rootProvider, prestateTimestamp), nil
		},
		getBottomPrestateProvider: cachePrestates(
			gameType,
			stateConverter,
			m,
			preStateBaseURL,
			preState,
			filepath.Join(cfg.Datadir, "super-cannon-prestates"),
			func(ctx context.Context, path string) faultTypes.PrestateProvider {
				return vm.NewPrestateProvider(path, stateConverter)
			}),
		newTraceAccessor: func(
			logger log.Logger,
			m metrics.Metricer,
			prestateProvider faultTypes.PrestateProvider,
			vmPrestateProvider faultTypes.PrestateProvider,
			dir string,
			l1Head eth.BlockID,
			splitDepth faultTypes.Depth,
			prestateBlock uint64,
			poststateBlock uint64) (*trace.Accessor, error) {
			provider := vmPrestateProvider.(*vm.PrestateProvider)
			preimagePrestateProvider := prestateProvider.(super.PreimagePrestateProvider)
			return super.NewSuperCannonTraceAccessor(logger, m, vmCfg, serverExecutor, preimagePrestateProvider, rootProvider, superNodeProvider, provider.PrestatePath(), dir, l1Head, splitDepth, prestateBlock, poststateBlock)
		},
	}
}

func NewCannonRegisterTask(gameType gameTypes.GameType, cfg *config.Config, m caching.Metrics, serverExecutor vm.OracleServerExecutor, l2Client utils.L2HeaderSource, rollupClient outputs.OutputRollupClient, syncValidator gameTypes.SyncValidator) *RegisterTask {
	return newCannonVMRegisterTaskWithConfig(gameType, cfg, m, serverExecutor, l2Client, rollupClient, syncValidator, cfg.Cannon, cfg.CannonAbsolutePreStateBaseURL, cfg.CannonAbsolutePreState)
}

func NewCannonKonaRegisterTask(gameType gameTypes.GameType, cfg *config.Config, m caching.Metrics, serverExecutor vm.OracleServerExecutor, l2Client utils.L2HeaderSource, rollupClient outputs.OutputRollupClient, syncValidator gameTypes.SyncValidator) *RegisterTask {
	return newCannonVMRegisterTaskWithConfig(gameType, cfg, m, serverExecutor, l2Client, rollupClient, syncValidator, cfg.CannonKona, cfg.CannonKonaAbsolutePreStateBaseURL, cfg.CannonKonaAbsolutePreState)
}

func newCannonVMRegisterTaskWithConfig(
	gameType gameTypes.GameType,
	cfg *config.Config,
	m caching.Metrics,
	serverExecutor vm.OracleServerExecutor,
	l2Client utils.L2HeaderSource,
	rollupClient outputs.OutputRollupClient,
	syncValidator gameTypes.SyncValidator,
	vmCfg vm.Config,
	preStateBaseURL *url.URL,
	preState string,
) *RegisterTask {
	stateConverter := cannon.NewStateConverter(cfg.Cannon)
	return &RegisterTask{
		gameType:      gameType,
		syncValidator: syncValidator,
		// Don't validate the absolute prestate or genesis output root for permissioned games
		// Only trusted actors participate in these games so they aren't expected to reach the step() call and
		// are often configured without valid prestates but the challenger should still resolve the games.
		skipPrestateValidation: gameType == gameTypes.PermissionedGameType,
		getTopPrestateProvider: func(ctx context.Context, prestateBlock uint64) (faultTypes.PrestateProvider, error) {
			return outputs.NewPrestateProvider(rollupClient, prestateBlock), nil
		},
		getBottomPrestateProvider: cachePrestates(
			gameType,
			stateConverter,
			m,
			preStateBaseURL,
			preState,
			filepath.Join(cfg.Datadir, vmCfg.VmType.String()+"-prestates"),
			func(ctx context.Context, path string) faultTypes.PrestateProvider {
				return vm.NewPrestateProvider(path, stateConverter)
			}),
		newTraceAccessor: func(
			logger log.Logger,
			m metrics.Metricer,
			prestateProvider faultTypes.PrestateProvider,
			vmPrestateProvider faultTypes.PrestateProvider,
			dir string,
			l1Head eth.BlockID,
			splitDepth faultTypes.Depth,
			prestateBlock uint64,
			poststateBlock uint64) (*trace.Accessor, error) {
			provider := vmPrestateProvider.(*vm.PrestateProvider)
			return outputs.NewOutputCannonTraceAccessor(logger, m, vmCfg, serverExecutor, l2Client, prestateProvider, provider.PrestatePath(), rollupClient, dir, l1Head, splitDepth, prestateBlock, poststateBlock)
		},
	}
}

func NewAlphabetRegisterTask(gameType gameTypes.GameType, l2Client utils.L2HeaderSource, rollupClient outputs.OutputRollupClient, syncValidator gameTypes.SyncValidator) *RegisterTask {
	return &RegisterTask{
		gameType:      gameType,
		syncValidator: syncValidator,
		getTopPrestateProvider: func(ctx context.Context, prestateBlock uint64) (faultTypes.PrestateProvider, error) {
			return outputs.NewPrestateProvider(rollupClient, prestateBlock), nil
		},
		getBottomPrestateProvider: func(_ context.Context, _ common.Hash) (faultTypes.PrestateProvider, error) {
			return alphabet.PrestateProvider, nil
		},
		newTraceAccessor: func(
			logger log.Logger,
			m metrics.Metricer,
			prestateProvider faultTypes.PrestateProvider,
			vmPrestateProvider faultTypes.PrestateProvider,
			dir string,
			l1Head eth.BlockID,
			splitDepth faultTypes.Depth,
			prestateBlock uint64,
			poststateBlock uint64) (*trace.Accessor, error) {
			return outputs.NewOutputAlphabetTraceAccessor(logger, m, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)
		},
	}
}

func cachePrestates(
	gameType gameTypes.GameType,
	stateConverter vm.StateConverter,
	m caching.Metrics,
	prestateBaseURL *url.URL,
	preStatePath string,
	prestateDir string,
	newPrestateProvider func(ctx context.Context, path string) faultTypes.PrestateProvider,
) func(ctx context.Context, prestateHash common.Hash) (faultTypes.PrestateProvider, error) {
	prestateSource := prestates.NewPrestateSource(prestateBaseURL, preStatePath, prestateDir, stateConverter)
	prestateProviderCache := prestates.NewPrestateProviderCache(m, fmt.Sprintf("prestates-%v", gameType),
		func(ctx context.Context, prestateHash common.Hash) (faultTypes.PrestateProvider, error) {
			prestatePath, err := prestateSource.PrestatePath(ctx, prestateHash)
			if err != nil {
				return nil, fmt.Errorf("required prestate %v not available: %w", prestateHash, err)
			}
			return newPrestateProvider(ctx, prestatePath), nil
		})
	return prestateProviderCache.GetOrCreate
}

func (e *RegisterTask) Register(
	ctx context.Context,
	registry Registry,
	oracles OracleRegistry,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l1HeaderSource generic.L1HeaderSource,
	selective bool,
	claimants []common.Address,
	responseDelay time.Duration,
	responseDelayAfter uint64) error {

	playerCreator := func(game gameTypes.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create fault dispute game contracts: %w", err)
		}
		requiredPrestatehash, err := contract.GetAbsolutePrestateHash(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load prestate hash for game %v: %w", game.Proxy, err)
		}

		vmPrestateProvider, err := e.getBottomPrestateProvider(ctx, requiredPrestatehash)
		if err != nil {
			return nil, fmt.Errorf("required prestate %v not available for game %v: %w", requiredPrestatehash, game.Proxy, err)
		}

		oracle, err := contract.GetOracle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load oracle for game %v: %w", game.Proxy, err)
		}
		oracles.RegisterOracle(oracle)
		prestateBlock, poststateBlock, err := contract.GetGameRange(ctx)
		if err != nil {
			return nil, err
		}
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load split depth: %w", err)
		}
		prestateProvider, err := e.getTopPrestateProvider(ctx, prestateBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to create top prestate provider: %w", err)
		}
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, l1HeadID eth.BlockID, dir string) (faultTypes.TraceAccessor, error) {
			accessor, err := e.newTraceAccessor(logger, m, prestateProvider, vmPrestateProvider, dir, l1HeadID, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		var validators []generic.PrestateValidator
		if !e.skipPrestateValidation {
			validators = append(validators, NewPrestateValidator(e.gameType.String(), contract.GetAbsolutePrestateHash, vmPrestateProvider))
			validators = append(validators, NewPrestateValidator("output root", contract.GetStartingRootHash, prestateProvider))
		}
		return generic.NewGenericGamePlayer(
			ctx,
			logger,
			game.Proxy,
			contract,
			e.syncValidator,
			validators,
			l1HeaderSource,
			AgentCreator(systemClock, l1Clock, m, dir, txSender, contract, creator, selective, claimants, responseDelay, responseDelayAfter),
		)
	}
	err := registerOracle(ctx, logger, oracles, gameFactory, e.gameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(e.gameType, playerCreator)

	contractCreator := func(game gameTypes.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
	}
	registry.RegisterBondContract(e.gameType, contractCreator)
	return nil
}

func registerOracle(ctx context.Context, logger log.Logger, oracles OracleRegistry, gameFactory *contracts.DisputeGameFactoryContract, gameType gameTypes.GameType) error {
	// Check that there is an implementation set for this game type and skip if not.
	hasImpl, err := gameFactory.HasGameImpl(ctx, gameType)
	if err != nil {
		return fmt.Errorf("failed to check implementation for game type %v: %w", gameType, err)
	}
	if !hasImpl {
		logger.Warn("No game implementation set for game type", "gameType", gameType)
		return nil
	}
	vmContract, err := gameFactory.GetGameVm(ctx, gameType)
	if err != nil {
		return fmt.Errorf("failed to get vm for game type %v: %w", gameType, err)
	}
	oracle, err := vmContract.Oracle(ctx)
	if err != nil {
		return fmt.Errorf("failed to load oracle address: %w", err)
	}
	oracles.RegisterOracle(oracle)
	return nil
}
