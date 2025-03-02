package extract

import (
	"context"
	"errors"
	"fmt"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrSupervisorRpcRequired = errors.New("supervisor rpc required")
)

type SuperRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error)
}

type SuperAgreementEnricher struct {
	log     log.Logger
	metrics OutputMetrics
	client  SuperRootProvider
	clock   clock.Clock
}

func NewSuperAgreementEnricher(logger log.Logger, metrics OutputMetrics, client SuperRootProvider, cl clock.Clock) *SuperAgreementEnricher {
	return &SuperAgreementEnricher{
		log:     logger,
		metrics: metrics,
		client:  client,
		clock:   cl,
	}
}

func (e *SuperAgreementEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if game.UsesOutputRoots() {
		return nil
	}
	if e.client == nil {
		return fmt.Errorf("%w but required for game type %v", ErrSupervisorRpcRequired, game.GameType)
	}
	response, err := e.client.SuperRootAtTimestamp(ctx, hexutil.Uint64(game.L2BlockNumber))
	if errors.Is(err, ethereum.NotFound) {
		// Super root doesn't exist, so we must disagree with it.
		game.AgreeWithClaim = false
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to retrieve super root at timestamp %v: %w", game.L2BlockNumber, err)
	}
	e.metrics.RecordOutputFetchTime(float64(e.clock.Now().Unix()))
	game.ExpectedRootClaim = common.Hash(response.SuperRoot)
	if game.RootClaim != game.ExpectedRootClaim {
		game.AgreeWithClaim = false
		return nil
	}
	game.AgreeWithClaim = response.CrossSafeDerivedFrom.Number <= game.L1HeadNum
	return nil
}
