package extract

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
	ErrSupervisorRpcRequired         = errors.New("supervisor rpc required")
	ErrAllSupervisorNodesUnavailable = errors.New("all supervisor nodes returned errors")
)

type SuperRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootResponse, error)
}

type SuperAgreementEnricher struct {
	log     log.Logger
	metrics OutputMetrics
	clients []SuperRootProvider
	clock   clock.Clock
}

func NewSuperAgreementEnricher(logger log.Logger, metrics OutputMetrics, clients []SuperRootProvider, cl clock.Clock) *SuperAgreementEnricher {
	return &SuperAgreementEnricher{
		log:     logger,
		metrics: metrics,
		clients: clients,
		clock:   cl,
	}
}

type superRootResult struct {
	superRoot            common.Hash
	isSafe               bool
	notFound             bool
	err                  error
	crossSafeDerivedFrom uint64
}

func (e *SuperAgreementEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if game.UsesOutputRoots() {
		return nil
	}
	if len(e.clients) == 0 {
		return fmt.Errorf("%w but required for game type %v", ErrSupervisorRpcRequired, game.GameType)
	}

	results := make([]superRootResult, len(e.clients))
	var wg sync.WaitGroup
	for i, client := range e.clients {
		wg.Add(1)
		go func(i int, client SuperRootProvider) {
			defer wg.Done()
			response, err := client.SuperRootAtTimestamp(ctx, hexutil.Uint64(game.L2BlockNumber))
			if errors.Is(err, ethereum.NotFound) {
				results[i] = superRootResult{notFound: true}
				return
			}
			if err != nil {
				results[i] = superRootResult{err: err}
				return
			}

			superRoot := common.Hash(response.SuperRoot)
			results[i] = superRootResult{
				superRoot:            superRoot,
				crossSafeDerivedFrom: response.CrossSafeDerivedFrom.Number,
				isSafe:               response.CrossSafeDerivedFrom.Number <= game.L1HeadNum,
			}
		}(i, client)
	}
	wg.Wait()

	validResults := make([]superRootResult, 0, len(results))
	foundResults := make([]superRootResult, 0, len(results))
	for idx, result := range results {
		if result.err != nil {
			e.log.Error("Failed to fetch super root", "clientIndex", idx, "l2BlockNum", game.L2BlockNumber, "err", result.err)
			continue
		}

		validResults = append(validResults, result)

		if !result.notFound {
			foundResults = append(foundResults, result)
		}
	}

	// If all results were errors, return an error
	if len(validResults) == 0 {
		return fmt.Errorf("failed to get super root at timestamp: %w", ErrAllSupervisorNodesUnavailable)
	}

	// If all remaining nodes returned "not found", we disagree with any claim.
	if len(foundResults) == 0 {
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = common.Hash{}
		return nil
	}

	// At least one node returned a super root, record the fetch time.
	e.metrics.RecordOutputFetchTime(float64(e.clock.Now().Unix()))

	// Check for disagreements among nodes.
	// A disagreement is any of:
	// - Mixed "found" and "not found" responses.
	// - Different super roots from nodes that found data.
	firstResult := foundResults[0]
	diverged := len(foundResults) < len(validResults)
	if !diverged {
		for _, result := range foundResults[1:] {
			if result.superRoot != firstResult.superRoot {
				diverged = true
				break
			}
		}
	}

	if diverged {
		e.log.Warn("Supervisor nodes disagree on super root",
			"l2BlockNum", game.L2BlockNumber,
			"firstSuperRoot", firstResult.superRoot,
			"found", len(foundResults),
			"valid", len(validResults))
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = firstResult.superRoot
		return nil
	}

	// All nodes that found a super root agree on the root.
	// Now check if the super root is considered safe by at least one node.
	atLeastOneSafe := false
	for _, result := range foundResults {
		if result.isSafe {
			atLeastOneSafe = true
			break
		}
	}

	// If no node considers the super root safe, we disagree.
	if !atLeastOneSafe {
		game.AgreeWithClaim = false
		if firstResult.superRoot == game.RootClaim {
			game.ExpectedRootClaim = common.Hash{}
		} else {
			game.ExpectedRootClaim = firstResult.superRoot
		}
		return nil
	}

	// All nodes agree and at least one considers the super root safe.
	// We agree with the claim if the game's root claim matches.
	game.ExpectedRootClaim = firstResult.superRoot
	game.AgreeWithClaim = game.RootClaim == firstResult.superRoot
	return nil
}
