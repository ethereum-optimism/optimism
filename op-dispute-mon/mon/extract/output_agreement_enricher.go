package extract

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrRollupRpcRequired = errors.New("rollup rpc required")
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
	SafeHeadAtL1Block(ctx context.Context, blockNum uint64) (*eth.SafeHeadResponse, error)
}

type OutputMetrics interface {
	RecordOutputFetchTime(float64)
}

type OutputAgreementEnricher struct {
	log     log.Logger
	metrics OutputMetrics
	clients []OutputRollupClient
	clock   clock.Clock
}

func NewOutputAgreementEnricher(logger log.Logger, metrics OutputMetrics, clients []OutputRollupClient, cl clock.Clock) *OutputAgreementEnricher {
	return &OutputAgreementEnricher{
		log:     logger,
		metrics: metrics,
		clients: clients,
		clock:   cl,
	}
}

type outputResult struct {
	outputRoot common.Hash
	isSafe     bool
	notFound   bool
	err        error
}

// Enrich validates the specified root claim against the output at the given block number.
func (o *OutputAgreementEnricher) Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.EnrichedGameData) error {
	if !game.UsesOutputRoots() {
		return nil
	}
	if len(o.clients) == 0 {
		return fmt.Errorf("%w but required for game type %v", ErrRollupRpcRequired, game.GameType)
	}
	if game.L2BlockNumber > math.MaxInt64 {
		// The claimed block number is bigger than an int64. The BlockNumber type used by RPCs is an int64 so anything
		// bigger than that can't be a valid block. So we can determine that this proposal invalid just because it
		// has a ridiculously big block number which must be far in the future.
		game.AgreeWithClaim = false
		return nil
	}

	results := make([]outputResult, len(o.clients))
	var wg sync.WaitGroup
	for i, client := range o.clients {
		wg.Add(1)
		go func(i int, client OutputRollupClient) {
			defer wg.Done()
			output, err := client.OutputAtBlock(ctx, game.L2BlockNumber)
			if err != nil {
				// string match as the error comes from the remote server so we can't use Errors.Is sadly.
				if strings.Contains(err.Error(), "not found") {
					results[i] = outputResult{notFound: true}
					return
				}
				results[i] = outputResult{err: err}
				return
			}

			outputRoot := common.Hash(output.OutputRoot)
			results[i] = outputResult{outputRoot: outputRoot}

			// Only check if the output root is safe if it matches the game's root claim
			if outputRoot == game.RootClaim {
				safeHead, err := client.SafeHeadAtL1Block(ctx, game.L1HeadNum)
				if err != nil {
					o.log.Warn("Unable to verify proposed block was safe", "l1HeadNum", game.L1HeadNum, "l2BlockNum", game.L2BlockNumber, "err", err)
					// If safe head data isn't available, assume the output root was safe
					// Avoids making the dispute mon dependent on safe head db being available
					results[i].isSafe = true
					return
				}
				results[i].isSafe = safeHead.SafeHead.Number >= game.L2BlockNumber
			}
		}(i, client)
	}
	wg.Wait()

	// Filter out results with errors (except "not found")
	validResults := make([]outputResult, 0, len(results))
	for _, result := range results {
		if result.err == nil {
			validResults = append(validResults, result)
		}
	}

	// If all results were errors, return an error
	if len(validResults) == 0 {
		return fmt.Errorf("failed to get output at block: all nodes returned errors")
	}

	// If all remaining nodes returned "not found", set game.AgreeWithClaim = false
	allNotFound := true
	for _, result := range validResults {
		if !result.notFound {
			allNotFound = false
			break
		}
	}
	if allNotFound {
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = common.Hash{}
		return nil
	}

	// Filter out nodes that returned "not found" as they're out of sync
	syncedResults := make([]outputResult, 0, len(validResults))
	for _, result := range validResults {
		if !result.notFound {
			syncedResults = append(syncedResults, result)
		}
	}
	if len(syncedResults) < len(validResults) {
		o.log.Warn("Some nodes are out of sync", "totalNodes", len(validResults), "syncedNodes", len(syncedResults))
	}

	// Check if nodes have diverged
	firstOutputRoot := syncedResults[0].outputRoot
	diverged := false
	for _, result := range syncedResults[1:] {
		if result.outputRoot != firstOutputRoot {
			diverged = true
			break
		}
	}

	if diverged {
		o.log.Error("Nodes have diverged", "firstNodeOutput", firstOutputRoot)
		// Use the result from the first node in the list
		game.ExpectedRootClaim = firstOutputRoot
		game.AgreeWithClaim = firstOutputRoot == game.RootClaim && syncedResults[0].isSafe
	} else {
		// All nodes agree on the output root
		game.ExpectedRootClaim = firstOutputRoot
		// Consider the output root "safe" if any node reported it as safe
		isSafe := false
		for _, result := range syncedResults {
			if result.isSafe {
				isSafe = true
				break
			}
		}
		game.AgreeWithClaim = firstOutputRoot == game.RootClaim && isSafe
	}

	o.metrics.RecordOutputFetchTime(float64(o.clock.Now().Unix()))
	return nil
}
