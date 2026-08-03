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

// NewZKAgreementEnricher creates a ZK agreement policy using the supplied root sources.
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
	// ZK games intentionally do not classify endpoints as safe or unsafe. Like the challenger,
	// they validate the pinned super root without gating it on VerifiedRequiredL1.
	if len(e.clients) == 0 {
		return fmt.Errorf("%w but required for game type %v", ErrSuperRootRpcRequired, game.GameType)
	}

	game.NodeEndpointTotalCount = len(e.clients)
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

	usable := make([]zkRootResult, 0, len(results))
	found := make([]zkRootResult, 0, len(results))
	for idx, result := range results {
		switch {
		case result.err != nil:
			e.log.Error("Failed to fetch ZK super root", "clientIndex", idx, "l2SequenceNumber", game.L2SequenceNumber, "err", result.err)
			game.NodeEndpointErrors[fmt.Sprintf("client-%d", idx)] = true
			game.NodeEndpointErrorCount++
		case result.outOfSync:
			game.NodeEndpointOutOfSyncCount++
		default:
			usable = append(usable, result)
			if result.notFound {
				game.NodeEndpointNotFoundCount++
			} else {
				found = append(found, result)
			}
		}
	}

	if len(usable) == 0 {
		if game.NodeEndpointOutOfSyncCount == len(e.clients) {
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
	differentRoots := false
	for _, result := range found[1:] {
		if result.root != firstRoot {
			differentRoots = true
			break
		}
	}
	game.NodeEndpointDifferentRoots = differentRoots
	if differentRoots || len(found) != len(usable) {
		e.log.Warn("ZK super root sources disagree",
			"l2SequenceNumber", game.L2SequenceNumber,
			"found", len(found),
			"notFound", game.NodeEndpointNotFoundCount,
			"differentRoots", differentRoots)
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = common.Hash{}
		return nil
	}

	game.ExpectedRootClaim = firstRoot
	game.AgreeWithClaim = game.RootClaim == firstRoot
	return nil
}
