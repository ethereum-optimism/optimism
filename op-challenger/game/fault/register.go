package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs/source"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type CloseFunc func()

type Registry interface {
	RegisterGameType(gameType uint32, creator scheduler.PlayerCreator, oracle keccakTypes.LargePreimageOracle)
	RegisterBondContract(gameType uint32, creator claims.BondContractCreator)
}

func RegisterGameTypes(
	registry Registry,
	ctx context.Context,
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	rollupClient source.OutputRollupClient,
	txSender types.TxSender,
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
	outputSourceCreator := source.NewOutputSourceCreator(logger, rollupClient)

	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		if err := registerCannon(registry, ctx, cl, logger, m, cfg, outputSourceCreator, txSender, gameFactory, caller, l2Client); err != nil {
			return nil, fmt.Errorf("failed to register cannon game type: %w", err)
		}
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		if err := registerAlphabet(registry, ctx, cl, logger, m, outputSourceCreator, txSender, gameFactory, caller); err != nil {
			return nil, fmt.Errorf("failed to register alphabet game type: %w", err)
		}
	}
	return closer, nil
}

func registerAlphabet(
	registry Registry,
	ctx context.Context,
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	outputSourceCreator *source.OutputSourceCreator,
	txSender types.TxSender,
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
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, err
		}
		l1Head, err := contract.GetL1Head(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load L1 head: %w", err)
		}
		rollupClient, err := outputSourceCreator.ForL1Head(ctx, l1Head)
		if err != nil {
			return nil, fmt.Errorf("failed to create output root source: %w", err)
		}
		prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			accessor, err := outputs.NewOutputAlphabetTraceAccessor(logger, m, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		genesisValidator := NewPrestateValidator(contract.GetGenesisOutputRoot, prestateProvider)
		return NewGamePlayer(ctx, cl, logger, m, dir, game.Proxy, txSender, contract, []Validator{prestateValidator, genesisValidator}, creator)
	}
	oracle, err := createOracle(ctx, gameFactory, caller, faultTypes.AlphabetGameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(faultTypes.AlphabetGameType, playerCreator, oracle)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(game.Proxy, caller)
	}
	registry.RegisterBondContract(faultTypes.AlphabetGameType, contractCreator)
	return nil
}

func createOracle(ctx context.Context, gameFactory *contracts.DisputeGameFactoryContract, caller *batching.MultiCaller, gameType uint32) (*contracts.PreimageOracleContract, error) {
	implAddr, err := gameFactory.GetGameImpl(ctx, gameType)
	if err != nil {
		return nil, fmt.Errorf("failed to load implementation for game type %v: %w", gameType, err)
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
	cl faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	outputSourceCreator *source.OutputSourceCreator,
	txSender types.TxSender,
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
		splitDepth, err := contract.GetSplitDepth(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load split depth: %w", err)
		}
		l1Head, err := contract.GetL1Head(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load L1 head: %w", err)
		}
		rollupClient, err := outputSourceCreator.ForL1Head(ctx, l1Head)
		if err != nil {
			return nil, fmt.Errorf("failed to create output root source: %w", err)
		}
		prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth faultTypes.Depth, dir string) (faultTypes.TraceAccessor, error) {
			accessor, err := outputs.NewOutputCannonTraceAccessor(logger, m, cfg, l2Client, contract, prestateProvider, rollupClient, dir, splitDepth, prestateBlock, poststateBlock)
			if err != nil {
				return nil, err
			}
			return accessor, nil
		}
		prestateValidator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		genesisValidator := NewPrestateValidator(contract.GetGenesisOutputRoot, prestateProvider)
		return NewGamePlayer(ctx, cl, logger, m, dir, game.Proxy, txSender, contract, []Validator{prestateValidator, genesisValidator}, creator)
	}
	oracle, err := createOracle(ctx, gameFactory, caller, faultTypes.CannonGameType)
	if err != nil {
		return err
	}
	registry.RegisterGameType(faultTypes.CannonGameType, playerCreator, oracle)

	contractCreator := func(game types.GameMetadata) (claims.BondContract, error) {
		return contracts.NewFaultDisputeGameContract(game.Proxy, caller)
	}
	registry.RegisterBondContract(faultTypes.CannonGameType, contractCreator)
	return nil
}
