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

var ErrIgnored = errors.New("ignored")

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

// ZKEnricher adds data that exists only for ZK games.
type ZKEnricher interface {
	Enrich(ctx context.Context, block rpcblock.Block, caller ZKGameCaller, game *monTypes.ZKGameData) error
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
	zkAgreement       ZKEnricher
	zkBonds           *BondDataEnricher
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
	zkAgreement ZKEnricher,
	zkBonds *BondDataEnricher,
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
		zkBonds:           zkBonds,
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
	enrichedCh := make(chan monTypes.EnrichedGame, len(games))
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
						return
					}
					e.logger.Trace("Enriching game", "game", game.Proxy)
					enrichedGame, err := e.enrichGame(ctx, blockHash, game)
					if errors.Is(err, ErrIgnored) {
						ignored.Add(1)
						e.logger.Warn("Ignoring game", "game", game.Proxy)
						continue
					}
					if errors.Is(err, gameTypes.ErrNotInSync) {
						e.logger.Debug("Waiting for root source to process past the game L1 head", "game", game.Proxy)
						current, currentIsZK := enrichedGame.(*monTypes.ZKGameData)
						cached, cachedIsZK := e.latestGameData[game.Proxy].(*monTypes.ZKGameData)
						if currentIsZK && cachedIsZK {
							enrichedCh <- withEndpointHealth(cached, current)
						}
						continue
					}
					if err != nil {
						failed.Add(1)
						e.logger.Error("Failed to fetch game data", "game", game.Proxy, "err", err)
						continue
					}
					enrichedCh <- enrichedGame
				}
			}
		}()
	}

	updatedGameData := make(map[common.Address]monTypes.EnrichedGame)
	for _, game := range games {
		previousData := e.latestGameData[game.Proxy]
		if previousData != nil {
			updatedGameData[game.Proxy] = previousData
		}
		gameCh <- game
	}
	close(gameCh)
	wg.Wait()
	close(enrichedCh)

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

func (e *Extractor) enrichZKGame(ctx context.Context, block rpcblock.Block, caller GameCaller, game gameTypes.GameMetadata) (monTypes.EnrichedGame, error) {
	zkCaller, ok := caller.(ZKBondGameCaller)
	if !ok {
		return nil, fmt.Errorf("game caller %T does not support ZK game extraction", caller)
	}
	meta, err := zkCaller.GetMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game metadata: %w", err)
	}
	enrichedGame := &monTypes.ZKGameData{
		CommonGameData: e.newCommonGameData(game, meta.L1Head, meta.L2SequenceNum, meta.ProposedRoot, meta.Status),
	}
	if err := e.applyCommonEnrichers(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	if err := e.zkAgreement.Enrich(ctx, block, zkCaller, enrichedGame); err != nil {
		if errors.Is(err, gameTypes.ErrNotInSync) {
			return enrichedGame, fmt.Errorf("failed to enrich ZK agreement: %w", err)
		}
		return nil, fmt.Errorf("failed to enrich ZK agreement: %w", err)
	}

	challengerMeta, err := zkCaller.GetChallengerMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZK challenger metadata: %w", err)
	}
	proposalStatus, err := contracts.ProposalStatusFromUint8(uint8(challengerMeta.ProposalStatus))
	if err != nil {
		return nil, err
	}
	if meta.ProposedRoot != challengerMeta.ProposedRoot {
		return nil, fmt.Errorf("inconsistent ZK root claim: generic %s, challenger %s", meta.ProposedRoot, challengerMeta.ProposedRoot)
	}
	if meta.L2SequenceNum != challengerMeta.L2SequenceNumber {
		return nil, fmt.Errorf("inconsistent ZK sequence number: generic %d, challenger %d", meta.L2SequenceNum, challengerMeta.L2SequenceNumber)
	}
	if err := validateZKStatus(meta.Status, proposalStatus); err != nil {
		return nil, err
	}
	if err := validateZKParticipants(proposalStatus, challengerMeta.Challenger, challengerMeta.Prover); err != nil {
		return nil, err
	}
	anchorStateRegistry, err := zkCaller.GetAnchorStateRegistry(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZK anchor state registry: %w", err)
	}
	if anchorStateRegistry == (common.Address{}) {
		return nil, errors.New("ZK anchor state registry is zero")
	}

	var parentStatus *gameTypes.GameStatus
	if challengerMeta.ParentIndex != math.MaxUint32 {
		status, err := e.fetchParentStatus(ctx, uint64(challengerMeta.ParentIndex), block)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ZK parent game %d status: %w", challengerMeta.ParentIndex, err)
		}
		parentStatus = &status
		if meta.Status != gameTypes.GameStatusInProgress && status == gameTypes.GameStatusInProgress {
			return nil, fmt.Errorf("terminal ZK child has in-progress parent %d", challengerMeta.ParentIndex)
		}
	}

	enrichedGame.AnchorStateRegistry = anchorStateRegistry
	enrichedGame.ParentIndex = challengerMeta.ParentIndex
	enrichedGame.ParentStatus = parentStatus
	enrichedGame.ProposalStatus = proposalStatus
	enrichedGame.Deadline = challengerMeta.Deadline
	enrichedGame.Challenger = challengerMeta.Challenger
	enrichedGame.Prover = challengerMeta.Prover
	bondMeta, err := zkCaller.GetBondMetadata(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZK bond metadata: %w", err)
	}
	enrichedGame.GameCreator = bondMeta.GameCreator
	enrichedGame.TotalBonds = cloneBigInt(bondMeta.TotalBonds)
	enrichedGame.ChallengerBond = cloneBigInt(bondMeta.ChallengerBond)
	if err := e.zkBonds.EnrichZK(ctx, block, zkCaller, enrichedGame); err != nil {
		return nil, fmt.Errorf("failed to enrich ZK game bonds: %w", err)
	}
	return enrichedGame, nil
}

func withEndpointHealth(cached, current *monTypes.ZKGameData) *monTypes.ZKGameData {
	updated := *cached
	updated.NodeEndpointErrors = maps.Clone(current.NodeEndpointErrors)
	updated.NodeEndpointErrorCount = current.NodeEndpointErrorCount
	updated.NodeEndpointNotFoundCount = current.NodeEndpointNotFoundCount
	updated.NodeEndpointOutOfSyncCount = current.NodeEndpointOutOfSyncCount
	updated.NodeEndpointTotalCount = current.NodeEndpointTotalCount
	updated.NodeEndpointSafeCount = current.NodeEndpointSafeCount
	updated.NodeEndpointUnsafeCount = current.NodeEndpointUnsafeCount
	updated.NodeEndpointDifferentRoots = current.NodeEndpointDifferentRoots
	return &updated
}

func validateZKStatus(global gameTypes.GameStatus, proposal contracts.ProposalStatus) error {
	proposalInProgress := proposal != contracts.ProposalStatusResolved
	if (global == gameTypes.GameStatusInProgress) != proposalInProgress {
		return fmt.Errorf("inconsistent ZK global status %s and proposal status %s", global, proposal)
	}
	return nil
}

func validateZKParticipants(status contracts.ProposalStatus, challenger, prover common.Address) error {
	zeroChallenger := challenger == (common.Address{})
	zeroProver := prover == (common.Address{})
	valid := false
	switch status {
	case contracts.ProposalStatusUnchallenged:
		valid = zeroChallenger && zeroProver
	case contracts.ProposalStatusChallenged:
		valid = !zeroChallenger && zeroProver
	case contracts.ProposalStatusUnchallengedAndValidProofProvided:
		valid = zeroChallenger && !zeroProver
	case contracts.ProposalStatusChallengedAndValidProofProvided:
		valid = !zeroChallenger && !zeroProver
	case contracts.ProposalStatusResolved:
		valid = true
	}
	if !valid {
		return fmt.Errorf("invalid ZK participants for proposal status %s: challenger %s, prover %s", status, challenger, prover)
	}
	return nil
}

func (e *Extractor) enrichFaultGame(ctx context.Context, block rpcblock.Block, caller GameCaller, game gameTypes.GameMetadata) (monTypes.EnrichedGame, error) {
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
			return nil, fmt.Errorf("failed to enrich game: %w", err)
		}
	}
	if err := e.applyCommonEnrichers(ctx, block, caller, enrichedGame.Common()); err != nil {
		return nil, fmt.Errorf("failed to enrich game: %w", err)
	}
	return enrichedGame, nil
}

func (e *Extractor) enrichSuperPermissionedGame(ctx context.Context, block rpcblock.Block, caller GameCaller, game gameTypes.GameMetadata) (monTypes.EnrichedGame, error) {
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

func (e *Extractor) newCommonGameData(game gameTypes.GameMetadata, l1Head common.Hash, l2SequenceNumber uint64, rootClaim common.Hash, status gameTypes.GameStatus) monTypes.CommonGameData {
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
