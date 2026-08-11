package extract

import (
	"context"
	"errors"
	"fmt"
	"sync"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrSuperRootRpcRequired        = errors.New("super root RPC required")
	ErrAllSuperRootRpcsUnavailable = errors.New("all super root RPC sources returned errors")
)

type SuperRootProvider interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
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
	superRoot common.Hash
	isSafe    bool
	notFound  bool
	outOfSync bool
	err       error
}

type superRootAgreementMode uint8

const (
	superGameAgreement superRootAgreementMode = iota
	zkGameAgreement
)

func (e *SuperAgreementEnricher) Enrich(ctx context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.CommonGameData) error {
	switch gameTypes.GameType(game.GameType) {
	case gameTypes.SuperPermissionedGameType, gameTypes.SuperCannonKonaGameType:
	default:
		return nil
	}
	return e.enrich(ctx, game, superGameAgreement)
}

func (e *SuperAgreementEnricher) enrich(ctx context.Context, game *monTypes.CommonGameData, mode superRootAgreementMode) error {
	isZKGame := mode == zkGameAgreement
	if len(e.clients) == 0 {
		return fmt.Errorf("%w but required for game type %v", ErrSuperRootRpcRequired, game.GameType)
	}

	if !isZKGame {
		game.NodeEndpointTotalCount = len(e.clients)
	}

	results := make([]superRootResult, len(e.clients))
	var wg sync.WaitGroup
	for i, client := range e.clients {
		wg.Add(1)
		go func(i int, client SuperRootProvider) {
			defer wg.Done()
			response, err := client.SuperRootAtTimestamp(ctx, game.L2SequenceNumber)
			if err != nil {
				results[i] = superRootResult{err: err}
				return
			}
			// A ZK provider's answer is usable only after it has processed beyond the game L1 head.
			if isZKGame && response.CurrentL1.Number <= game.L1HeadNum {
				results[i] = superRootResult{outOfSync: true}
				return
			}
			if response.Data == nil {
				results[i] = superRootResult{notFound: true}
				return
			}

			superRoot := common.Hash(response.Data.SuperRoot)
			// If the super root that we computed matches the game's root claim, the game could
			// still technically be invalid if the L1 data required to verify the super root was
			// not fully available on the L1 at the time the game was proposed. In this case, "safe"
			// means that all the L1 data needed to verify cross-chain dependencies was available.
			// The game itself is still "safe" from a security/liveness perspective, but the game
			// would be challenged by an honest proposer.
			results[i] = superRootResult{
				superRoot: superRoot,
				isSafe:    response.Data.VerifiedRequiredL1.Number <= game.L1HeadNum,
			}
		}(i, client)
	}
	wg.Wait()
	if isZKGame {
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	endpointErrors := game.NodeEndpointErrors
	errorCount := game.NodeEndpointErrorCount
	notFoundCount := game.NodeEndpointNotFoundCount
	outOfSyncCount := game.NodeEndpointOutOfSyncCount
	safeCount := game.NodeEndpointSafeCount
	unsafeCount := game.NodeEndpointUnsafeCount
	differentRoots := game.NodeEndpointDifferentRoots
	if isZKGame {
		endpointErrors = make(map[string]bool)
		errorCount = 0
		notFoundCount = 0
		outOfSyncCount = 0
		safeCount = 0
		unsafeCount = 0
		differentRoots = false
		game.NodeEndpointTotalCount = len(e.clients)
	}
	validResults := make([]superRootResult, 0, len(results))
	foundResults := make([]superRootResult, 0, len(results))
	for idx, result := range results {
		if result.err != nil {
			if isZKGame {
				e.log.Error("Failed to fetch ZK super root", "clientIndex", idx, "l2SequenceNumber", game.L2SequenceNumber, "err", result.err)
			} else {
				e.log.Error("Failed to fetch super root", "clientIndex", idx, "l2SequenceNumber", game.L2SequenceNumber, "err", result.err)
			}
			endpointID := fmt.Sprintf("client-%d", idx)
			endpointErrors[endpointID] = true
			errorCount++
			continue
		}
		if result.outOfSync {
			outOfSyncCount++
			continue
		}

		validResults = append(validResults, result)

		if result.notFound {
			notFoundCount++
		} else {
			foundResults = append(foundResults, result)
			// Track safety counts only for found results where the super root matches the game's root claim
			if !isZKGame && result.superRoot == game.RootClaim {
				if result.isSafe {
					safeCount++
				} else {
					unsafeCount++
				}
			}
		}
	}
	game.NodeEndpointErrors = endpointErrors
	game.NodeEndpointErrorCount = errorCount
	game.NodeEndpointNotFoundCount = notFoundCount
	game.NodeEndpointOutOfSyncCount = outOfSyncCount
	game.NodeEndpointSafeCount = safeCount
	game.NodeEndpointUnsafeCount = unsafeCount
	game.NodeEndpointDifferentRoots = differentRoots

	// If all results were errors, return an error
	if len(validResults) == 0 {
		if isZKGame && outOfSyncCount == len(e.clients) {
			return fmt.Errorf("all ZK super root sources are behind game L1 head %d: %w", game.L1HeadNum, gameTypes.ErrNotInSync)
		}
		if isZKGame {
			return fmt.Errorf("failed to get ZK super root at timestamp: %w", ErrAllSuperRootRpcsUnavailable)
		}
		return fmt.Errorf("failed to get super root at timestamp: %w", ErrAllSuperRootRpcsUnavailable)
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
	if !diverged || isZKGame {
		for _, result := range foundResults[1:] {
			if result.superRoot != firstResult.superRoot {
				diverged = true
				differentRoots = true
				break
			}
		}
	}
	game.NodeEndpointDifferentRoots = differentRoots

	if diverged {
		if isZKGame {
			e.log.Warn("ZK super root sources disagree",
				"l2SequenceNumber", game.L2SequenceNumber,
				"firstSuperRoot", firstResult.superRoot,
				"found", len(foundResults),
				"notFound", notFoundCount,
				"differentRoots", differentRoots)
		} else {
			e.log.Warn("Super nodes disagree on super root",
				"l2SequenceNumber", game.L2SequenceNumber,
				"firstSuperRoot", firstResult.superRoot,
				"found", len(foundResults),
				"valid", len(validResults))
		}
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = firstResult.superRoot
		return nil
	}
	if isZKGame {
		game.ExpectedRootClaim = firstResult.superRoot
		game.AgreeWithClaim = game.RootClaim == firstResult.superRoot
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
