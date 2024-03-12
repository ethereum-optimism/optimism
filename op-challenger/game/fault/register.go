package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
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

type RollupClient interface {
	outputs.OutputRollupClient
	SyncStatusProvider
}

func RegisterGameTypes(
	ctx context.Context,
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	registry Registry,
	oracles OracleRegistry,
	rollupClient RollupClient,
	txSender types.TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) (CloseFunc, error) {
	var closer CloseFunc
	var l2Client *ethclient.Client
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) || cfg.TraceTypeEnabled(config.TraceTypePermissioned) {
		l2, err := ethclient.DialContext(ctx, cfg.CannonL2)
		if err != nil {
			return nil, fmt.Errorf("dial l2 client %v: %w", cfg.CannonL2, err)
		}
		l2Client = l2
		closer = l2Client.Close
	}
	syncValidator := newSyncStatusValidator(rollupClient)

	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		if err := registerCannon(faultTypes.CannonGameType, registry, oracles, ctx, cl, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypePermissioned) {
		if err := registerCannon(faultTypes.PermissionedGameType, registry, oracles, ctx, cl, logger, m, cfg, syncValidator, rollupClient, txSender, gameFactory, caller, l2Client, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register permissioned cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		if err := registerAlphabet(registry, oracles, ctx, cl, logger, m, syncValidator, rollupClient, txSender, gameFactory, caller, l1HeaderSource, selective, claimants); err != nil {
			return nil, fmt.Errorf("failed to register alphabet game type: %w", err)
		}
	}
	return closer, nil
}

func registerAlphabet(
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	syncValidator SyncValidator,
	rollupClient RollupClient,
	txSender types.TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
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
			accessor, err := outputs.NewOutputAlphabetTraceAccessor(logger, m, prestateProvider, rollupClient, l1Head, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator("alphabet", contract.GetAbsolutePrestateHash, alphabet.PrestateProvider)
		startingValidator := NewPrestateValidator("output root", contract.GetStartingRootHash, prestateProvider)
		return NewGamePlayer(ctx, cl, logger, m, dir, game.Proxy, txSender, contract, syncValidator, []Validator{prestateValidator, startingValidator}, creator, l1HeaderSource, selective, claimants)
	}
	err := registerOracle(ctx, oracles, gameFactory, caller, faultTypes.AlphabetGameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(faultTypes.AlphabetGameType, playerCreator)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(game.Proxy, caller)
	}
	registry.RegisterBondContract(faultTypes.AlphabetGameType, contractCreator)
	return nil
}

func registerOracle(ctx context.Context, oracles OracleRegistry, gameFactory *contracts.DisputeGameFactoryContract, caller *batching.MultiCaller, gameType uint32) error {
	implAddr, err := gameFactory.GetGameImpl(ctx, gameType)
	if err != nil {
		return fmt.Errorf("failed to load implementation for game type %v: %w", gameType, err)
	}
	contract, err := contracts.NewFaultDisputeGameContract(implAddr, caller)
	if err != nil {
		return err
	}
	oracle, err := contract.GetOracle(ctx)
	if err != nil {
		return fmt.Errorf("failed to load oracle address: %w", err)
	}
	oracles.RegisterOracle(oracle)
	return nil
}

func registerCannon(
	gameType uint32,
	registry Registry,
	oracles OracleRegistry,
	ctx context.Context,
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	syncValidator SyncValidator,
	rollupClient outputs.OutputRollupClient,
	txSender types.TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
) error {
	cannonPrestateProvider := cannon.NewPrestateProvider(cfg.CannonAbsolutePreState)
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
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
			accessor, err := outputs.NewOutputCannonTraceAccessor(logger, m, cfg, l2Client, prestateProvider, rollupClient, dir, l1HeadID, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator("cannon", contract.GetAbsolutePrestateHash, cannonPrestateProvider)
		startingValidator := NewPrestateValidator("output root", contract.GetStartingRootHash, prestateProvider)
		return NewGamePlayer(ctx, cl, logger, m, dir, game.Proxy, txSender, contract, syncValidator, []Validator{prestateValidator, startingValidator}, creator, l1HeaderSource, selective, claimants)
	}
	err := registerOracle(ctx, oracles, gameFactory, caller, gameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(gameType, playerCreator)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(game.Proxy, caller)
	}
	registry.RegisterBondContract(gameType, contractCreator)
	return nil
}

func loadL1Head(contract *contracts.FaultDisputeGameContract, ctx context.Context, l1HeaderSource L1HeaderSource) (eth.BlockID, error) {
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
