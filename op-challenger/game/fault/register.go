package fault

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/asterisc"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
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
	RegisterGameType(gameType uint32, creator scheduler.PlayerCreator)
	RegisterBondContract(gameType uint32, creator claims.BondContractCreator)
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

	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		if err := registerCannon(faultTypes.CannonGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypePermissioned) {
		if err := registerCannon(faultTypes.PermissionedGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register permissioned cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAsterisc) {
		if err := registerAsterisc(faultTypes.AsteriscGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register asterisc game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeFast) {
		if err := registerAlphabet(faultTypes.FastGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, syncValidator, rollupClient, l2Client, txSender, gameFactory, caller, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register fast game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		if err := registerAlphabet(faultTypes.AlphabetGameType, registry, oracles, ctx, systemClock, l1Clock, logger, m, syncValidator, rollupClient, l2Client, txSender, gameFactory, caller, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register alphabet game type: %w", err)
		}
	}
	return l2Client.Close, nil
}

func registerAlphabet(
	gameType uint32,
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

func registerOracle(ctx context.Context, m metrics.Metricer, oracles OracleRegistry, gameFactory *contracts.DisputeGameFactoryContract, caller *batching.MultiCaller, gameType uint32) error {
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
	gameType uint32,
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
	var prestateSource PrestateSource
	if cfg.AsteriscAbsolutePreStateBaseURL != nil {
		prestateSource = prestates.NewMultiPrestateProvider(cfg.AsteriscAbsolutePreStateBaseURL, filepath.Join(cfg.Datadir, "asterisc-prestates"))
	} else {
		prestateSource = prestates.NewSinglePrestateSource(cfg.AsteriscAbsolutePreState)
	}
	prestateProviderCache := prestates.NewPrestateProviderCache(m, fmt.Sprintf("prestates-%v", gameType), func(prestateHash common.Hash) (faultTypes.PrestateProvider, error) {
		prestatePath, err := prestateSource.PrestatePath(prestateHash)
		if err != nil {
			return nil, fmt.Errorf("required prestate %v not available: %w", prestateHash, err)
		}
		return asterisc.NewPrestateProvider(prestatePath), nil
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
		asteriscPrestateProvider, err := prestateProviderCache.GetOrCreate(requiredPrestatehash)
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
		prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			asteriscPrestate, err := prestateSource.PrestatePath(requiredPrestatehash)
			if err != nil {
				return nil, fmt.Errorf("failed to get asterisc prestate: %w", err)
			}
			accessor, err := outputs.NewOutputAsteriscTraceAccessor(logger, m, cfg, l2Client, prestateProvider, asteriscPrestate, rollupClient, dir, l1HeadID, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator("asterisc", contract.GetAbsolutePrestateHash, asteriscPrestateProvider)
		genesisValidator := NewPrestateValidator("output root", contract.GetStartingRootHash, prestateProvider)
		return NewGamePlayer(ctx, systemClock, l1Clock, logger, m, dir, game.Proxy, txSender, contract, syncValidator, []Validator{prestateValidator, genesisValidator}, creator, l1HeaderSource, selective, claimants)
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

func registerCannon(
	gameType uint32,
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
	var prestateSource PrestateSource
	if cfg.CannonAbsolutePreStateBaseURL != nil {
		prestateSource = prestates.NewMultiPrestateProvider(cfg.CannonAbsolutePreStateBaseURL, filepath.Join(cfg.Datadir, "cannon-prestates"))
	} else {
		prestateSource = prestates.NewSinglePrestateSource(cfg.CannonAbsolutePreState)
	}
	prestateProviderCache := prestates.NewPrestateProviderCache(m, fmt.Sprintf("prestates-%v", gameType), func(prestateHash common.Hash) (faultTypes.PrestateProvider, error) {
		prestatePath, err := prestateSource.PrestatePath(prestateHash)
		if err != nil {
			return nil, fmt.Errorf("required prestate %v not available: %w", prestateHash, err)
		}
		return cannon.NewPrestateProvider(prestatePath), nil
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

		cannonPrestateProvider, err := prestateProviderCache.GetOrCreate(requiredPrestatehash)

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
		prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			cannonPrestate, err := prestateSource.PrestatePath(requiredPrestatehash)
			if err != nil {
				return nil, fmt.Errorf("failed to get cannon prestate: %w", err)
			}
			accessor, err := outputs.NewOutputCannonTraceAccessor(logger, m, cfg, l2Client, prestateProvider, cannonPrestate, rollupClient, dir, l1HeadID, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator("cannon", contract.GetAbsolutePrestateHash, cannonPrestateProvider)
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
