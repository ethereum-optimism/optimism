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
	rollupClient outputs.OutputRollupClient,
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
		registerOutputCannon(registry, ctx, logger, m, cfg, rollupClient, txMgr, caller, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputAlphabet) {
		registerOutputAlphabet(registry, ctx, logger, m, rollupClient, txMgr, caller)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		registerCannon(registry, ctx, logger, m, cfg, txMgr, caller, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		registerAlphabet(registry, ctx, logger, m, cfg.AlphabetTrace, txMgr, caller)
	}
	return closer, nil
}

func registerOutputAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	rollupClient outputs.OutputRollupClient,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller) {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewOutputBisectionGameContract(game.Proxy, caller)
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
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
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
	registry.RegisterGameType(outputAlphabetGameType, playerCreator)
}

func registerOutputCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	rollupClient outputs.OutputRollupClient,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewOutputBisectionGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
		if err != nil {
			return nil, err
		}
		prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
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
	registry.RegisterGameType(outputCannonGameType, playerCreator)
}

func registerCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		prestateProvider := cannon.NewPrestateProvider(cfg.CannonAbsolutePreState)
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			localInputs, err := cannon.FetchLocalInputs(ctx, contract, l2Client)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch cannon local inputs: %w", err)
			}
			provider := cannon.NewTraceProvider(logger, m, cfg, faultTypes.NoLocalContext, localInputs, dir, gameDepth)
			return trace.NewSimpleTraceAccessor(provider), nil
		}
		validator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, []Validator{validator}, creator)
	}
	registry.RegisterGameType(cannonGameType, playerCreator)
}

func registerAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	alphabetTrace string,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller) {
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		contract, err := contracts.NewFaultDisputeGameContract(game.Proxy, caller)
		if err != nil {
			return nil, err
		}
		prestateProvider := &alphabet.AlphabetPrestateProvider{}
		creator := func(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
			traceProvider := alphabet.NewTraceProvider(alphabetTrace, gameDepth)
			return trace.NewSimpleTraceAccessor(traceProvider), nil
		}
		validator := NewPrestateValidator(contract.GetAbsolutePrestateHash, prestateProvider)
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, contract, []Validator{validator}, creator)
	}
	registry.RegisterGameType(alphabetGameType, playerCreator)
}
