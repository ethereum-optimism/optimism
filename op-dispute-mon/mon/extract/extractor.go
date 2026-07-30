package extract

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrIgnored = errors.New("ignored")
)

type (
	CreateGameCaller        func(ctx context.Context, game gameTypes.GameMetadata) (GameCaller, error)
	FactoryGameFetcher      func(ctx context.Context, blockHash common.Hash, earliestTimestamp uint64) ([]gameTypes.GameMetadata, error)
	ParentGameStatusFetcher func(ctx context.Context, index uint64, block rpcblock.Block) (gameTypes.GameStatus, error)
)

// CommonEnricher adds data shared by every enriched game variant.
type CommonEnricher interface {
	Enrich(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.CommonGameData) error
}

// FaultEnricher adds data that exists only for fault games.
type FaultEnricher interface {
	Enrich(ctx context.Context, block rpcblock.Block, caller FaultGameCaller, game *monTypes.FaultGameData) error
}

type Extractor struct {
	logger            log.Logger
	clock             clock.Clock
	createContract    CreateGameCaller
	fetchGames        FactoryGameFetcher
	fetchParentStatus ParentGameStatusFetcher
	maxConcurrency    int
	commonEnrichers   []CommonEnricher
	faultEnrichers    []FaultEnricher
	zkAgreement       CommonEnricher
	ignoredGames      map[common.Address]bool
	latestGameData    map[common.Address]monTypes.EnrichedGame
}

func NewExtractor(
	logger log.Logger,
	cl clock.Clock,
	creator CreateGameCaller,
	fetchGames FactoryGameFetcher,
	fetchParentStatus ParentGameStatusFetcher,
	ignoredGames []common.Address,
	maxConcurrency uint,
	commonEnrichers []CommonEnricher,
	faultEnrichers []FaultEnricher,
	zkAgreement CommonEnricher,
) *Extractor {
	ignored := make(map[common.Address]bool)
	for _, game := range ignoredGames {
		ignored[game] = true
	}
	return &Extractor{
		logger:            logger,
		clock:             cl,
		createContract:    creator,
		fetchGames:        fetchGames,
		fetchParentStatus: fetchParentStatus,
		maxConcurrency:    int(maxConcurrency),
		commonEnrichers:   commonEnrichers,
		faultEnrichers:    faultEnrichers,
		zkAgreement:       zkAgreement,
		ignoredGames:      ignored,
	}
}

func (e *Extractor) Extract(ctx context.Context, blockHash common.Hash, minTimestamp uint64) ([]monTypes.EnrichedGame, int, int, error) {
	games, err := e.fetchGames(ctx, blockHash, minTimestamp)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to load games: %w", err)
	}
	enriched, ignored, failed := e.enrichGames(ctx, blockHash, games)
	return enriched, ignored, failed, nil
}

func (e *Extractor) enrichGames(ctx context.Context, blockHash common.Hash, games []gameTypes.GameMetadata) ([]monTypes.EnrichedGame, int, int) {
	var ignored atomic.Int32
	var failed atomic.Int32

	var wg sync.WaitGroup
	wg.Add(e.maxConcurrency)
	gameCh := make(chan gameTypes.GameMetadata, e.maxConcurrency)
	// Create a channel for enriched games. Must have enough capacity to hold all games.
	enrichedCh := make(chan monTypes.EnrichedGame, len(games))
	// Spin up multiple goroutines to enrich game data
	for i := 0; i < e.maxConcurrency; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case game, ok := <-gameCh:
					if !ok {
						e.logger.Debug("Enriching complete")
						// Channel closed
						return
					}
					e.logger.Trace("Enriching game", "game", game.Proxy)
					enrichedGame, err := e.enrichGame(ctx, blockHash, game)
					if errors.Is(err, ErrIgnored) {
						ignored.Add(1)
						e.logger.Warn("Ignoring game", "game", game.Proxy)
						continue
					} else if err != nil {
						failed.Add(1)
						e.logger.Error("Failed to fetch game data", "game", game.Proxy, "err", err)
						continue
					}
					enrichedCh <- enrichedGame
				}
			}
		}()
	}

	// Create a new store for game data. This ensures any games no longer in the monitoring set are dropped.
	updatedGameData := make(map[common.Address]monTypes.EnrichedGame)
	// Push each game into the channel and store the latest cached game data as a default if fetching fails
	for _, game := range games {
		previousData := e.latestGameData[game.Proxy]
		if previousData != nil {
			updatedGameData[game.Proxy] = previousData
		}
		gameCh <- game
	}
	close(gameCh)
	// Wait for games to finish being enriched then close enrichedCh since no future results will be published
	wg.Wait()
	close(enrichedCh)

	// Read the results
	for enrichedGame := range enrichedCh {
		updatedGameData[enrichedGame.Common().Proxy] = enrichedGame
	}
	e.latestGameData = updatedGameData
	return slices.Collect(maps.Values(updatedGameData)), int(ignored.Load()), int(failed.Load())
}

func (e *Extractor) enrichGame(ctx context.Context, blockHash common.Hash, game gameTypes.GameMetadata) (monTypes.EnrichedGame, error) {
	if e.ignoredGames[game.Proxy] {
		return nil, ErrIgnored
	}
	caller, err := e.createContract(ctx, game)
	if err != nil {
		return nil, fmt.Errorf("failed to create contracts: %w", err)
	}
	block := rpcblock.ByHash(blockHash)
	switch gameTypes.GameType(game.GameType) {
	case gameTypes.SuperPermissionedGameType:
		return e.enrichSuperPermissionedGame(ctx, block, caller, game)
	case gameTypes.ZKDisputeGameType:
		return e.enrichZKGame(ctx, block, caller, game)
	case gameTypes.CannonGameType,
		gameTypes.PermissionedGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.AlphabetGameType,
		gameTypes.FastGameType,
		gameTypes.SuperCannonKonaGameType:
		return e.enrichFaultGame(ctx, block, caller, game)
	default:
		return nil, fmt.Errorf("unsupported game type: %d", game.GameType)
	}
}

func (e *Extractor) enrichFaultGame(
	ctx context.Context,
	block rpcblock.Block,
	caller GameCaller,
	game gameTypes.GameMetadata,
) (monTypes.EnrichedGame, error) {
	faultCaller, ok := caller.(FaultGameCaller)
	if !ok {
		return nil, fmt.Errorf("game caller %T does not support fault game extraction", caller)
	}
	meta, err := faultCaller.GetExtendedMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	claims, err := faultCaller.GetAllClaims(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game claims: %w", err)
	}
	enrichedClaims := make([]monTypes.EnrichedClaim, len(claims))
	for i, claim := range claims {
		enrichedClaims[i] = monTypes.EnrichedClaim{Claim: claim}
	}
	enrichedGame := &monTypes.FaultGameData{
		CommonGameData:        e.newCommonGameData(game, meta.L1Head, meta.L2SequenceNum, meta.RootClaim, meta.Status),
		MaxClockDuration:      meta.MaxClockDuration,
		BlockNumberChallenged: meta.L2BlockNumberChallenged,
		BlockNumberChallenger: meta.L2BlockNumberChallenger,
		Claims:                enrichedClaims,
	}
	for _, enricher := range e.faultEnrichers {
		if err := enricher.Enrich(ctx, block, faultCaller, enrichedGame); err != nil {
			return nil, fmt.Errorf("failed to enrich fault game: %w", err)
		}
	}
	if err := e.applyCommonEnrichers(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	return enrichedGame, nil
}

func (e *Extractor) enrichSuperPermissionedGame(
	ctx context.Context,
	block rpcblock.Block,
	caller GameCaller,
	game gameTypes.GameMetadata,
) (monTypes.EnrichedGame, error) {
	metadataCaller, ok := caller.(MetadataCaller)
	if !ok {
		return nil, fmt.Errorf("game caller %T does not support common game extraction", caller)
	}
	meta, err := metadataCaller.GetExtendedMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	enrichedGame := &monTypes.SuperPermissionedGameData{
		CommonGameData: e.newCommonGameData(game, meta.L1Head, meta.L2SequenceNum, meta.RootClaim, meta.Status),
	}
	if err := e.applyCommonEnrichers(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	return enrichedGame, nil
}

func (e *Extractor) enrichZKGame(
	ctx context.Context,
	block rpcblock.Block,
	caller GameCaller,
	game gameTypes.GameMetadata,
) (monTypes.EnrichedGame, error) {
	zkCaller, ok := caller.(ZKGameCaller)
	if !ok {
		return nil, fmt.Errorf("game caller %T does not support ZK game extraction", caller)
	}
	meta, err := zkCaller.GetMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	challengerMeta, err := zkCaller.GetChallengerMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZK challenger metadata: %w", err)
	}
	if meta.ProposedRoot != challengerMeta.ProposedRoot {
		return nil, fmt.Errorf("inconsistent ZK root claim: generic %s, challenger %s", meta.ProposedRoot, challengerMeta.ProposedRoot)
	}
	if meta.L2SequenceNum != challengerMeta.L2SequenceNumber {
		return nil, fmt.Errorf("inconsistent ZK sequence number: generic %d, challenger %d", meta.L2SequenceNum, challengerMeta.L2SequenceNumber)
	}
	proposalInProgress := challengerMeta.ProposalStatus != contracts.ProposalStatusResolved
	if (meta.Status == gameTypes.GameStatusInProgress) != proposalInProgress {
		return nil, fmt.Errorf("inconsistent ZK global status %s and proposal status %s", meta.Status, challengerMeta.ProposalStatus)
	}

	hasParent := challengerMeta.ParentIndex != math.MaxUint32
	parentStatus := gameTypes.GameStatusInProgress
	if hasParent {
		if e.fetchParentStatus == nil {
			return nil, errors.New("parent game status fetcher is required for ZK child games")
		}
		parentStatus, err = e.fetchParentStatus(ctx, uint64(challengerMeta.ParentIndex), block)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ZK parent game %d status: %w", challengerMeta.ParentIndex, err)
		}
		if meta.Status != gameTypes.GameStatusInProgress && parentStatus == gameTypes.GameStatusInProgress {
			return nil, fmt.Errorf("terminal ZK child has in-progress parent %d", challengerMeta.ParentIndex)
		}
	}

	enrichedGame := &monTypes.ZKGameData{
		CommonGameData: e.newCommonGameData(game, meta.L1Head, meta.L2SequenceNum, meta.ProposedRoot, meta.Status),
		ParentIndex:    challengerMeta.ParentIndex,
		HasParent:      hasParent,
		ParentStatus:   parentStatus,
		ProposalStatus: challengerMeta.ProposalStatus,
	}
	if err := e.applyCommonEnrichers(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	if e.zkAgreement == nil {
		return nil, errors.New("ZK agreement enricher is required")
	}
	if err := e.zkAgreement.Enrich(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich ZK agreement: %w", err)
	}
	return enrichedGame, nil
}

func (e *Extractor) newCommonGameData(
	game gameTypes.GameMetadata,
	l1Head common.Hash,
	l2SequenceNumber uint64,
	rootClaim common.Hash,
	status gameTypes.GameStatus,
) monTypes.CommonGameData {
	return monTypes.CommonGameData{
		LastUpdateTime:             e.clock.Now(),
		GameMetadata:               game,
		L1Head:                     l1Head,
		L2SequenceNumber:           l2SequenceNumber,
		RootClaim:                  rootClaim,
		Status:                     status,
		NodeEndpointErrors:         make(map[string]bool),
		NodeEndpointErrorCount:     0,
		NodeEndpointNotFoundCount:  0,
		NodeEndpointOutOfSyncCount: 0,
		NodeEndpointTotalCount:     0,
		NodeEndpointSafeCount:      0,
		NodeEndpointUnsafeCount:    0,
		NodeEndpointDifferentRoots: false,
	}
}

func (e *Extractor) applyCommonEnrichers(ctx context.Context, block rpcblock.Block, caller GameCaller, game *monTypes.CommonGameData) error {
	for _, enricher := range e.commonEnrichers {
		if err := enricher.Enrich(ctx, block, caller, game); err != nil {
			return err
		}
	}
	return nil
}
