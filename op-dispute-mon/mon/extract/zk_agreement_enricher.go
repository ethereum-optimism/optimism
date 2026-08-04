package extract

import (
	"context"
	"fmt"
	"sync"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ZKAgreementEnricher reconciles a ZK proposal with configured super-root sources.
type ZKAgreementEnricher struct {
	log     log.Logger
	metrics OutputMetrics
	clients []SuperRootProvider
	clock   clock.Clock
}

var _ ZKEnricher = (*ZKAgreementEnricher)(nil)

func NewZKAgreementEnricher(logger log.Logger, metrics OutputMetrics, clients []SuperRootProvider, cl clock.Clock) *ZKAgreementEnricher {
	return &ZKAgreementEnricher{
		log:     logger,
		metrics: metrics,
		clients: clients,
		clock:   cl,
	}
}

type zkRootResult struct {
	root      common.Hash
	notFound  bool
	outOfSync bool
	err       error
}

func (e *ZKAgreementEnricher) Enrich(ctx context.Context, _ rpcblock.Block, _ ZKGameCaller, zkGame *monTypes.ZKGameData) error {
	game := zkGame.Common()
	if len(e.clients) == 0 {
		return fmt.Errorf("%w but required for game type %v", ErrSuperRootRpcRequired, game.GameType)
	}

	results := make([]zkRootResult, len(e.clients))
	var wg sync.WaitGroup
	for i, client := range e.clients {
		wg.Add(1)
		go func(i int, client SuperRootProvider) {
			defer wg.Done()
			response, err := client.SuperRootAtTimestamp(ctx, game.L2SequenceNumber)
			if err != nil {
				results[i] = zkRootResult{err: err}
				return
			}
			// CurrentL1 is trusted as the provider's number-progress boundary. A provider must
			// have processed beyond the proposal's L1 head before its answer is usable.
			if response.CurrentL1.Number <= game.L1HeadNum {
				results[i] = zkRootResult{outOfSync: true}
				return
			}
			if response.Data == nil {
				results[i] = zkRootResult{notFound: true}
				return
			}
			results[i] = zkRootResult{root: common.Hash(response.Data.SuperRoot)}
		}(i, client)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}

	endpointErrors := make(map[string]bool)
	usable := make([]zkRootResult, 0, len(results))
	found := make([]zkRootResult, 0, len(results))
	errorCount := 0
	notFoundCount := 0
	outOfSyncCount := 0
	for idx, result := range results {
		switch {
		case result.err != nil:
			e.log.Error("Failed to fetch ZK super root", "clientIndex", idx, "l2SequenceNumber", game.L2SequenceNumber, "err", result.err)
			endpointErrors[fmt.Sprintf("client-%d", idx)] = true
			errorCount++
		case result.outOfSync:
			outOfSyncCount++
		default:
			usable = append(usable, result)
			if result.notFound {
				notFoundCount++
			} else {
				found = append(found, result)
			}
		}
	}

	game.NodeEndpointTotalCount = len(e.clients)
	game.NodeEndpointErrors = endpointErrors
	game.NodeEndpointErrorCount = errorCount
	game.NodeEndpointNotFoundCount = notFoundCount
	game.NodeEndpointOutOfSyncCount = outOfSyncCount
	game.NodeEndpointSafeCount = 0
	game.NodeEndpointUnsafeCount = 0
	game.NodeEndpointDifferentRoots = false

	if len(usable) == 0 {
		if outOfSyncCount == len(e.clients) {
			return fmt.Errorf("all ZK super root sources are behind game L1 head %d: %w", game.L1HeadNum, gameTypes.ErrNotInSync)
		}
		return fmt.Errorf("failed to get ZK super root at timestamp: %w", ErrAllSuperRootRpcsUnavailable)
	}
	if len(found) == 0 {
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = common.Hash{}
		return nil
	}

	e.metrics.RecordOutputFetchTime(float64(e.clock.Now().Unix()))
	firstRoot := found[0].root
	game.ExpectedRootClaim = firstRoot
	for _, result := range found[1:] {
		if result.root != firstRoot {
			game.NodeEndpointDifferentRoots = true
			break
		}
	}
	if game.NodeEndpointDifferentRoots || len(found) != len(usable) {
		e.log.Warn("ZK super root sources disagree",
			"l2SequenceNumber", game.L2SequenceNumber,
			"firstSuperRoot", firstRoot,
			"found", len(found),
			"notFound", notFoundCount,
			"differentRoots", game.NodeEndpointDifferentRoots)
		game.AgreeWithClaim = false
		return nil
	}

	game.AgreeWithClaim = game.RootClaim == firstRoot
	return nil
}
