package fault

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/preimages"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/responder"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/generic"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type GameInfo interface {
	GetStatus(context.Context) (gameTypes.GameStatus, error)
	GetClaimCount(context.Context) (uint64, error)
}

type L1HeaderSource interface {
	HeaderByHash(context.Context, common.Hash) (*gethTypes.Header, error)
}

type TxSender interface {
	From() common.Address
	SendAndWaitSimple(txPurpose string, txs ...txmgr.TxCandidate) error
}

type GameContract interface {
	preimages.PreimageGameContract
	responder.GameContract
	claims.BondContract
	GameInfo
	ClaimLoader
	GetStatus(ctx context.Context) (gameTypes.GameStatus, error)
	GetMaxGameDepth(ctx context.Context) (types.Depth, error)
	GetMaxClockDuration(ctx context.Context) (time.Duration, error)
	GetOracle(ctx context.Context) (contracts.PreimageOracleContract, error)
	GetL1Head(ctx context.Context) (common.Hash, error)
}

type resourceCreator func(ctx context.Context, logger log.Logger, gameDepth types.Depth, dir string) (types.TraceAccessor, error)

func NewGamePlayer(
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock types.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	dir string,
	addr common.Address,
	txSender TxSender,
	loader GameContract,
	syncValidator generic.SyncValidator,
	validators []generic.PrestateValidator,
	creator resourceCreator,
	l1HeaderSource L1HeaderSource,
	selective bool,
	claimants []common.Address,
	responseDelay time.Duration,
	responseDelayAfter uint64,
) (*generic.GamePlayer, error) {
	return generic.NewGenericGamePlayer(
		ctx,
		logger,
		addr,
		loader,
		syncValidator,
		validators,
		l1HeaderSource,
		func(ctx context.Context, logger log.Logger) (generic.Actor, error) {
			maxClockDuration, err := loader.GetMaxClockDuration(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch the game duration: %w", err)
			}

			gameDepth, err := loader.GetMaxGameDepth(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch the game depth: %w", err)
			}

			accessor, err := creator(ctx, logger, gameDepth, dir)
			if err != nil {
				return nil, fmt.Errorf("failed to create trace accessor: %w", err)
			}

			oracle, err := loader.GetOracle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load oracle: %w", err)
			}

			minLargePreimageSize, err := oracle.MinLargePreimageSize(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to load min large preimage size: %w", err)
			}
			direct := preimages.NewDirectPreimageUploader(logger, txSender, loader)
			large := preimages.NewLargePreimageUploader(logger, l1Clock, txSender, oracle)
			uploader := preimages.NewSplitPreimageUploader(direct, large, minLargePreimageSize)
			responder, err := responder.NewFaultResponder(logger, txSender, loader, uploader, oracle)
			if err != nil {
				return nil, fmt.Errorf("failed to create the responder: %w", err)
			}

			agent := NewAgent(
				m,
				systemClock,
				l1Clock,
				loader,
				gameDepth,
				maxClockDuration,
				accessor,
				responder,
				logger,
				selective,
				claimants,
				responseDelay,
				responseDelayAfter,
			)
			return agent, nil
		})
}
