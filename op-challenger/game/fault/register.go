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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var (
	cannonGameType         = uint8(0)
	outputCannonGameType   = uint8(0) // TODO(client-pod#260): Switch the output cannon game type to 1
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
		registerOutputCannon(registry, ctx, logger, m, cfg, txMgr, caller, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeOutputAlphabet) {
		registerOutputAlphabet(registry, ctx, logger, m, cfg, txMgr, caller, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeCannon) {
		registerCannon(registry, ctx, logger, m, cfg, txMgr, caller, l2Client)
	}
	if cfg.TraceTypeEnabled(config.TraceTypeAlphabet) {
		registerAlphabet(registry, ctx, logger, m, cfg, txMgr, caller)
	}
	return closer, nil
}

func registerOutputAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) {
	resourceCreator := func(addr common.Address) (gameTypeResources, error) {
		contract, err := contracts.NewOutputBisectionGameContract(addr, caller)
		if err != nil {
			return nil, err
		}
		return &outputAlphabetResources{
			m:        m,
			cfg:      cfg,
			l2Client: l2Client,
			contract: contract,
		}, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, resourceCreator)
	}
	registry.RegisterGameType(outputAlphabetGameType, playerCreator)
}

type outputAlphabetResources struct {
	m        metrics.Metricer
	cfg      *config.Config
	l2Client cannon.L2HeaderSource
	contract *contracts.OutputBisectionGameContract
}

func (r *outputAlphabetResources) Contract() GameContract {
	return r.contract
}

func (r *outputAlphabetResources) CreateAccessor(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
	// TODO(client-pod#44): Validate absolute pre-state for split games
	prestateBlock, poststateBlock, err := r.contract.GetBlockRange(ctx)
	if err != nil {
		return nil, err
	}
	splitDepth, err := r.contract.GetSplitDepth(ctx)
	if err != nil {
		return nil, err
	}
	accessor, err := outputs.NewOutputAlphabetTraceAccessor(ctx, logger, r.m, r.cfg, gameDepth, splitDepth, prestateBlock, poststateBlock)
	if err != nil {
		return nil, err
	}
	return accessor, nil
}

func registerOutputCannon(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller,
	l2Client cannon.L2HeaderSource) {
	resourceCreator := func(addr common.Address) (gameTypeResources, error) {
		// Currently still using the old fault dispute game contracts for output_cannon
		// as the output bisection+cannon contract isn't being deployed.
		contract, err := contracts.NewFaultDisputeGameContract(addr, caller)
		if err != nil {
			return nil, err
		}
		return &outputCannonResources{
			m:        m,
			cfg:      cfg,
			l2Client: l2Client,
			contract: contract,
		}, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, resourceCreator)
	}
	registry.RegisterGameType(outputCannonGameType, playerCreator)
}

type outputCannonResources struct {
	m        metrics.Metricer
	cfg      *config.Config
	l2Client cannon.L2HeaderSource
	contract *contracts.FaultDisputeGameContract // TODO(client-pod#260): Use the OutputBisectionGame Contract
}

func (r *outputCannonResources) Contract() GameContract {
	return r.contract
}

func (r *outputCannonResources) CreateAccessor(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
	// TODO(client-pod#44): Validate absolute pre-state for split games
	// TODO(client-pod#43): Updated contracts should expose this as the pre and post state blocks
	agreed, disputed, err := r.contract.GetProposals(ctx)
	if err != nil {
		return nil, err
	}
	accessor, err := outputs.NewOutputCannonTraceAccessor(ctx, logger, r.m, r.cfg, r.l2Client, r.contract, dir, gameDepth, agreed.L2BlockNumber.Uint64(), disputed.L2BlockNumber.Uint64())
	if err != nil {
		return nil, err
	}
	return accessor, nil
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
	resourceCreator := func(addr common.Address) (gameTypeResources, error) {
		contract, err := contracts.NewFaultDisputeGameContract(addr, caller)
		if err != nil {
			return nil, err
		}
		return &cannonResources{
			m:        m,
			cfg:      cfg,
			l2Client: l2Client,
			contract: contract,
		}, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, resourceCreator)
	}
	registry.RegisterGameType(cannonGameType, playerCreator)
}

type cannonResources struct {
	m        metrics.Metricer
	cfg      *config.Config
	l2Client cannon.L2HeaderSource
	contract *contracts.FaultDisputeGameContract
}

func (r *cannonResources) Contract() GameContract {
	return r.contract
}

func (r *cannonResources) CreateAccessor(ctx context.Context, logger log.Logger, gameDepth uint64, dir string) (faultTypes.TraceAccessor, error) {
	localInputs, err := cannon.FetchLocalInputs(ctx, r.contract, r.l2Client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cannon local inputs: %w", err)
	}
	provider := cannon.NewTraceProvider(logger, r.m, r.cfg, faultTypes.NoLocalContext, localInputs, dir, gameDepth)
	if err := ValidateAbsolutePrestate(ctx, provider, r.contract); err != nil {
		return nil, err
	}
	return trace.NewSimpleTraceAccessor(provider), nil
}

func registerAlphabet(
	registry Registry,
	ctx context.Context,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	txMgr txmgr.TxManager,
	caller *batching.MultiCaller) {
	resourceCreator := func(addr common.Address) (gameTypeResources, error) {
		contract, err := contracts.NewFaultDisputeGameContract(addr, caller)
		if err != nil {
			return nil, err
		}
		return &alphabetResources{
			cfg:      cfg,
			contract: contract,
		}, nil
	}
	playerCreator := func(game types.GameMetadata, dir string) (scheduler.GamePlayer, error) {
		return NewGamePlayer(ctx, logger, m, dir, game.Proxy, txMgr, resourceCreator)
	}
	registry.RegisterGameType(alphabetGameType, playerCreator)
}

type alphabetResources struct {
	cfg      *config.Config
	contract *contracts.FaultDisputeGameContract
}

func (r *alphabetResources) Contract() GameContract {
	return r.contract
}

func (r *alphabetResources) CreateAccessor(ctx context.Context, _ log.Logger, gameDepth uint64, _ string) (faultTypes.TraceAccessor, error) {
	provider := alphabet.NewTraceProvider(r.cfg.AlphabetTrace, gameDepth)
	if err := ValidateAbsolutePrestate(ctx, provider, r.contract); err != nil {
		return nil, err
	}
	return trace.NewSimpleTraceAccessor(provider), nil
}
