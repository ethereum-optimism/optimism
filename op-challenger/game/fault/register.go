package fault

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/asterisc"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/mtcannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type CloseFunc func()

type Registry interface {
	RegisterGameType(gameType faultTypes.GameType, creator scheduler.PlayerCreator)
	RegisterBondContract(gameType faultTypes.GameType, creator claims.BondContractCreator)
}

type OracleRegistry interface {
	RegisterOracle(oracle keccakTypes.LargePreimageOracle)
}

type PrestateSource interface {
	// PrestatePath returns the path to the prestate file to use for the game.
	// The provided prestateHash may be used to differentiate between different states but no guarantee is made that
	// the returned prestate matches the supplied hash.
	PrestatePath(prestateHash common.Hash) (string, error)
}

type RollupClient interface {
	outputs.OutputRollupClient
	SyncStatusProvider
}

func RegisterGameTypes(
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	registry Registry,
	oracles OracleRegistry,
	rollupClient RollupClient,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) (CloseFunc, error) {
	l2Client, err := ethclient.DialContext(ctx, cfg.L2Rpc)
	if err != nil {
		return nil, fmt.Errorf("dial l2 client %v: %w", cfg.L2Rpc, err)
	}
	syncValidator := newSyncStatusValidator(rollupClient)

	if cfg.TraceTypeEnabled(faultTypes.TraceTypeCannon) {
		if err := registerCannon(registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(faultTypes.TraceTypePermissioned) {
		if err := registerCannon(registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register permissioned cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(faultTypes.TraceTypeAsterisc) {
		if err := registerAsterisc(registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register asterisc game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(faultTypes.TraceTypeMTCannon) {
		if err := registerMTCannon(registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register multi-threaded cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(faultTypes.TraceTypeFast) {
		if err := registerAlphabet(faultTypes.FastGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, syncValidator, rollupClient, l2Client, txSender, gameFactory, caller, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register fast game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(faultTypes.TraceTypeAlphabet) {
		if err := registerAlphabet(faultTypes.AlphabetGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, syncValidator, rollupClient, l2Client, txSender, gameFactory, caller, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register alphabet game type: %w", err)
		}
	}
	return l2Client.Close, nil
}

func registerAlphabet(
	gameType faultTypes.GameType,
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	syncValidator SyncValidator,
	rollupClient RollupClient,
	l2Client utils.L2HeaderSource,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create fault dispute game contract: %w", err)
		}
		oracle, err := contract.GetOracle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load oracle for game %v: %w", game.Proxy, err)
		}
		oracles.RegisterOracle(oracle)
		prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
		if err != nil {
			return nil, err
		}
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, err
		}
		l1Head, err := loadL1Head(contract, ctx, l1HeaderSource)
		if err != nil {
			return nil, err
		}
		prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			accessor, err := outputs.NewOutputAlphabetTraceAccessor(logger, m, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator("alphabet", contract.GetAbsolutePrestateHash, alphabet.PrestateProvider)
		startingValidator := NewPrestateValidator("output root", contract.GetStartingRootHash, prestateProvider)
		return NewGamePlayer(ctx, systemClock, l1Clock, logger, m, dir, game.Proxy, txSender, contract, syncValidator, []Validator{prestateValidator, startingValidator}, creator, l1HeaderSource, selective, claimants)
	}
	err := registerOracle(ctx, m, oracles, gameFactory, caller, gameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(gameType, playerCreator)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
	}
	registry.RegisterBondContract(gameType, contractCreator)
	return nil
}

func registerOracle(ctx context.Context, m metrics.Metricer, oracles OracleRegistry, gameFactory *contracts.DisputeGameFactoryContract, caller *batching.MultiCaller, gameType faultTypes.GameType) error {
	implAddr, err := gameFactory.GetGameImpl(ctx, gameType)
	if err != nil {
		return fmt.Errorf("failed to load implementation for game type %v: %w", gameType, err)
	}
	contract, err := contracts.NewFaultDisputeGameContract(ctx, m, implAddr, caller)
	if err != nil {
		return fmt.Errorf("failed to create fault dispute game contracts: %w", err)
	}
	oracle, err := contract.GetOracle(ctx)
	if err != nil {
		return fmt.Errorf("failed to load oracle address: %w", err)
	}
	oracles.RegisterOracle(oracle)
	return nil
}

func registerAsterisc(
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	syncValidator SyncValidator,
	rollupClient outputs.OutputRollupClient,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client utils.L2HeaderSource,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	registerCfg := &registerGameTypeConfig{
		AbsolutePreStateBaseURL:          cfg.AsteriscAbsolutePreStateBaseURL,
		AbsolutePreState:                 cfg.AsteriscAbsolutePreState,
		Datadir:                          cfg.Datadir,
		VmConfig:                         cfg.Asterisc,
		PreStateProviderFactory:          createPrestateProviderFactory(asterisc.NewPrestateProvider),
		OutputCannonTraceAccessorFactory: outputs.NewOutputAsteriscTraceAccessor,
	}
	return registerGameType(
		faultTypes.AlphabetGameType,
		registry,
		oracles,
		ctx,
		systemClock,
		l1Clock, logger,
		m,
		registerCfg,
		syncValidator,
		rollupClient,
		txSender,
		gameFactory,
		caller,
		l2Client,
		l1HeaderSource,
		selective,
		claimants,
	)
}

func registerCannon(
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	syncValidator SyncValidator,
	rollupClient outputs.OutputRollupClient,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client utils.L2HeaderSource,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	registerCfg := &registerGameTypeConfig{
		AbsolutePreStateBaseURL:          cfg.CannonAbsolutePreStateBaseURL,
		AbsolutePreState:                 cfg.CannonAbsolutePreState,
		Datadir:                          cfg.Datadir,
		VmConfig:                         cfg.Cannon,
		PreStateProviderFactory:          createPrestateProviderFactory(cannon.NewPrestateProvider),
		OutputCannonTraceAccessorFactory: outputs.NewOutputCannonTraceAccessor,
	}
	return registerGameType(
		faultTypes.CannonGameType,
		registry,
		oracles,
		ctx,
		systemClock,
		l1Clock, logger,
		m,
		registerCfg,
		syncValidator,
		rollupClient,
		txSender,
		gameFactory,
		caller,
		l2Client,
		l1HeaderSource,
		selective,
		claimants,
	)
}

func registerMTCannon(
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	syncValidator SyncValidator,
	rollupClient outputs.OutputRollupClient,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client utils.L2HeaderSource,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	registerCfg := &registerGameTypeConfig{
		AbsolutePreStateBaseURL:          cfg.MTCannonAbsolutePreStateBaseURL,
		AbsolutePreState:                 cfg.MTCannonAbsolutePreState,
		Datadir:                          cfg.Datadir,
		VmConfig:                         cfg.MTCannon,
		PreStateProviderFactory:          createPrestateProviderFactory(mtcannon.NewPrestateProvider),
		OutputCannonTraceAccessorFactory: outputs.NewOutputMTCannonTraceAccessor,
	}
	return registerGameType(
		faultTypes.MTCannonGameType,
		registry,
		oracles,
		ctx,
		systemClock,
		l1Clock, logger,
		m,
		registerCfg,
		syncValidator,
		rollupClient,
		txSender,
		gameFactory,
		caller,
		l2Client,
		l1HeaderSource,
		selective,
		claimants,
	)
}

func createPrestateProviderFactory[T faultTypes.PrestateProvider](factory func(string) T) func(prestatePath string) faultTypes.PrestateProvider {
	return func(prestatePath string) faultTypes.PrestateProvider {
		return factory(prestatePath)
	}
}

type registerGameTypeConfig struct {
	AbsolutePreStateBaseURL          *url.URL
	AbsolutePreState                 string
	Datadir                          string
	VmConfig                         vm.Config
	PreStateProviderFactory          func(prestatePath string) faultTypes.PrestateProvider
	OutputCannonTraceAccessorFactory func(
		logger log.Logger,
		m metrics.Metricer,
		cfg vm.Config,
		l2Client utils.L2HeaderSource,
		prestateProvider faultTypes.PrestateProvider,
		cannonPrestate string,
		rollupClient outputs.OutputRollupClient,
		dir string,
		l1Head eth.BlockID,
		splitDepth faultTypes.Depth,
		prestateBlock uint64,
		poststateBlock uint64,
	) (*trace.Accessor, error)
}

func registerGameType(
	gameType faultTypes.GameType,
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *registerGameTypeConfig,
	syncValidator SyncValidator,
	rollupClient outputs.OutputRollupClient,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client utils.L2HeaderSource,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	if cfg.Datadir == "" {
		return errors.New("datadir must be set")
	}
	if cfg.VmConfig == (vm.Config{}) {
		return errors.New("vm config must be set")
	}
	if cfg.PreStateProviderFactory == nil {
		return errors.New("prestate provider factory must be set")
	}

	var prestateSource PrestateSource
	if cfg.AbsolutePreStateBaseURL != nil {
		var dir string
		switch gameType {
		case faultTypes.CannonGameType:
			dir = "cannon-prestates"
		case faultTypes.AsteriscGameType:
			dir = "asterisc-prestates"
		case faultTypes.MTCannonGameType:
			dir = "mtcannon-prestates"
		}
		prestateSource = prestates.NewMultiPrestateProvider(cfg.AbsolutePreStateBaseURL, filepath.Join(cfg.Datadir, dir))
	} else {
		prestateSource = prestates.NewSinglePrestateSource(cfg.AbsolutePreState)
	}
	prestateProviderCache := prestates.NewPrestateProviderCache(m, fmt.Sprintf("prestates-%v", gameType), func(prestateHash common.Hash) (faultTypes.PrestateProvider, error) {
		prestatePath, err := prestateSource.PrestatePath(prestateHash)
		if err != nil {
			return nil, fmt.Errorf("required prestate %v not available: %w", prestateHash, err)
		}
		return cfg.PreStateProviderFactory(prestatePath), nil
	})
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
		if err != nil {
			return nil, fmt.Errorf("failed to create fault dispute game contracts: %w", err)
		}
		requiredPrestatehash, err := contract.GetAbsolutePrestateHash(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load prestate hash for game %v: %w", game.Proxy, err)
		}

		prestateProvider, err := prestateProviderCache.GetOrCreate(requiredPrestatehash)
		if err != nil {
			return nil, fmt.Errorf("required prestate %v not available for game %v: %w", requiredPrestatehash, game.Proxy, err)
		}

		oracle, err := contract.GetOracle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load oracle for game %v: %w", game.Proxy, err)
		}
		oracles.RegisterOracle(oracle)
		prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
		if err != nil {
			return nil, err
		}
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load split depth: %w", err)
		}
		l1HeadID, err := loadL1Head(contract, ctx, l1HeaderSource)
		if err != nil {
			return nil, err
		}
		outputPrestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			prestate, err := prestateSource.PrestatePath(requiredPrestatehash)
			if err != nil {
				return nil, fmt.Errorf("failed to get %s prestate: %w", gameType, err)
			}
			accessor, err := cfg.OutputCannonTraceAccessorFactory(logger, m, cfg.VmConfig, l2Client, outputPrestateProvider, prestate, rollupClient, dir, l1HeadID, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator(gameType.String(), contract.GetAbsolutePrestateHash, prestateProvider)
		startingValidator := NewPrestateValidator("output root", contract.GetStartingRootHash, outputPrestateProvider)
		return NewGamePlayer(ctx, systemClock, l1Clock, logger, m, dir, game.Proxy, txSender, contract, syncValidator, []Validator{prestateValidator, startingValidator}, creator, l1HeaderSource, selective, claimants)
	}
	err := registerOracle(ctx, m, oracles, gameFactory, caller, gameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(gameType, playerCreator)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(ctx, m, game.Proxy, caller)
	}
	registry.RegisterBondContract(gameType, contractCreator)
	return nil
}

func loadL1Head(contract contracts.FaultDisputeGameContract, ctx context.Context, l1HeaderSource L1HeaderSource) (eth.BlockID, error) {
	l1Head, err := contract.GetL1Head(ctx)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to load L1 head: %w", err)
	}
	l1Header, err := l1HeaderSource.HeaderByHash(ctx, l1Head)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to load L1 header: %w", err)
	}
	return eth.HeaderBlockID(l1Header), nil
}
