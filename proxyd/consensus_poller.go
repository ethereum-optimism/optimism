package proxyd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/log"
)

const (
	PollerInterval = 1 * time.Second
)

type OnConsensusBroken func()

// ConsensusPoller checks the consensus local for each member of a BackendGroup
// resolves the highest common block for multiple nodes, and reconciles the consensus
// in case of block hash divergence to minimize re-orgs
type ConsensusPoller struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	listeners  []OnConsensusBroken

	backendGroup      *BackendGroup
	backendState      map[*Backend]*backendState
	consensusGroupMux sync.Mutex
	consensusGroup    []*Backend

	tracker      ConsensusTracker
	asyncHandler ConsensusAsyncHandler

	minPeerCount       uint64
	banPeriod          time.Duration
	maxUpdateThreshold time.Duration
	maxBlockLag        uint64
	maxBlockRange      uint64
}

type backendState struct {
	backendStateMux sync.Mutex

	latestBlockNumber    hexutil.Uint64
	latestBlockHash      string
	safeBlockNumber      hexutil.Uint64
	finalizedBlockNumber hexutil.Uint64

	peerCount uint64
	inSync    bool

	lastUpdate time.Time

	bannedUntil time.Time
}

func (bs *backendState) IsBanned() bool {
	return time.Now().Before(bs.bannedUntil)
}

// GetConsensusGroup returns the backend members that are agreeing in a consensus
func (cp *ConsensusPoller) GetConsensusGroup() []*Backend {
	defer cp.consensusGroupMux.Unlock()
	cp.consensusGroupMux.Lock()

	g := make([]*Backend, len(cp.consensusGroup))
	copy(g, cp.consensusGroup)

	return g
}

// GetLatestBlockNumber returns the `latest` agreed block number in a consensus
func (ct *ConsensusPoller) GetLatestBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetLatestBlockNumber()
}

// GetSafeBlockNumber returns the `safe` agreed block number in a consensus
func (ct *ConsensusPoller) GetSafeBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetSafeBlockNumber()
}

// GetFinalizedBlockNumber returns the `finalized` agreed block number in a consensus
func (ct *ConsensusPoller) GetFinalizedBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetFinalizedBlockNumber()
}

func (cp *ConsensusPoller) Shutdown() {
	cp.asyncHandler.Shutdown()
}

// ConsensusAsyncHandler controls the asynchronous polling mechanism, interval and shutdown
type ConsensusAsyncHandler interface {
	Init()
	Shutdown()
}

// NoopAsyncHandler allows fine control updating the consensus
type NoopAsyncHandler struct{}

func NewNoopAsyncHandler() ConsensusAsyncHandler {
	log.Warn("using NewNoopAsyncHandler")
	return &NoopAsyncHandler{}
}
func (ah *NoopAsyncHandler) Init()     {}
func (ah *NoopAsyncHandler) Shutdown() {}

// PollerAsyncHandler asynchronously updates each individual backend and the group consensus
type PollerAsyncHandler struct {
	ctx context.Context
	cp  *ConsensusPoller
}

func NewPollerAsyncHandler(ctx context.Context, cp *ConsensusPoller) ConsensusAsyncHandler {
	return &PollerAsyncHandler{
		ctx: ctx,
		cp:  cp,
	}
}
func (ah *PollerAsyncHandler) Init() {
	// create the individual backend pollers
	for _, be := range ah.cp.backendGroup.Backends {
		go func(be *Backend) {
			for {
				timer := time.NewTimer(PollerInterval)
				ah.cp.UpdateBackend(ah.ctx, be)

				select {
				case <-timer.C:
				case <-ah.ctx.Done():
					timer.Stop()
					return
				}
			}
		}(be)
	}

	// create the group consensus poller
	go func() {
		for {
			timer := time.NewTimer(PollerInterval)
			ah.cp.UpdateBackendGroupConsensus(ah.ctx)

			select {
			case <-timer.C:
			case <-ah.ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}
func (ah *PollerAsyncHandler) Shutdown() {
	ah.cp.cancelFunc()
}

type ConsensusOpt func(cp *ConsensusPoller)

func WithTracker(tracker ConsensusTracker) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.tracker = tracker
	}
}

func WithAsyncHandler(asyncHandler ConsensusAsyncHandler) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.asyncHandler = asyncHandler
	}
}

func WithListener(listener OnConsensusBroken) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.AddListener(listener)
	}
}

func (cp *ConsensusPoller) AddListener(listener OnConsensusBroken) {
	cp.listeners = append(cp.listeners, listener)
}

func (cp *ConsensusPoller) ClearListeners() {
	cp.listeners = []OnConsensusBroken{}
}

func WithBanPeriod(banPeriod time.Duration) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.banPeriod = banPeriod
	}
}

func WithMaxUpdateThreshold(maxUpdateThreshold time.Duration) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.maxUpdateThreshold = maxUpdateThreshold
	}
}

func WithMaxBlockLag(maxBlockLag uint64) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.maxBlockLag = maxBlockLag
	}
}

func WithMaxBlockRange(maxBlockRange uint64) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.maxBlockRange = maxBlockRange
	}
}

func WithMinPeerCount(minPeerCount uint64) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.minPeerCount = minPeerCount
	}
}

func NewConsensusPoller(bg *BackendGroup, opts ...ConsensusOpt) *ConsensusPoller {
	ctx, cancelFunc := context.WithCancel(context.Background())

	state := make(map[*Backend]*backendState, len(bg.Backends))

	cp := &ConsensusPoller{
		ctx:          ctx,
		cancelFunc:   cancelFunc,
		backendGroup: bg,
		backendState: state,

		banPeriod:          5 * time.Minute,
		maxUpdateThreshold: 30 * time.Second,
		maxBlockLag:        8, // 8*12 seconds = 96 seconds ~ 1.6 minutes
		minPeerCount:       3,
	}

	for _, opt := range opts {
		opt(cp)
	}

	if cp.tracker == nil {
		cp.tracker = NewInMemoryConsensusTracker()
	}

	if cp.asyncHandler == nil {
		cp.asyncHandler = NewPollerAsyncHandler(ctx, cp)
	}

	cp.Reset()
	cp.asyncHandler.Init()

	return cp
}

// UpdateBackend refreshes the consensus local of a single backend
func (cp *ConsensusPoller) UpdateBackend(ctx context.Context, be *Backend) {
	bs := cp.getBackendState(be)
	RecordConsensusBackendBanned(be, bs.IsBanned())

	if bs.IsBanned() {
		log.Debug("skipping backend - banned", "backend", be.Name)
		return
	}

	// if backend is not healthy local we'll only resume checking it after ban
	if !be.IsHealthy() {
		log.Warn("backend banned - not healthy", "backend", be.Name)
		cp.Ban(be)
		return
	}

	inSync, err := cp.isInSync(ctx, be)
	RecordConsensusBackendInSync(be, err == nil && inSync)
	if err != nil {
		log.Warn("error updating backend sync local", "name", be.Name, "err", err)
	}

	var peerCount uint64
	if !be.skipPeerCountCheck {
		peerCount, err = cp.getPeerCount(ctx, be)
		if err != nil {
			log.Warn("error updating backend peer count", "name", be.Name, "err", err)
		}
		RecordConsensusBackendPeerCount(be, peerCount)
	}

	latestBlockNumber, latestBlockHash, err := cp.fetchBlock(ctx, be, "latest")
	if err != nil {
		log.Warn("error updating backend - latest block", "name", be.Name, "err", err)
	}

	safeBlockNumber, _, err := cp.fetchBlock(ctx, be, "safe")
	if err != nil {
		log.Warn("error updating backend - safe block", "name", be.Name, "err", err)
	}

	finalizedBlockNumber, _, err := cp.fetchBlock(ctx, be, "finalized")
	if err != nil {
		log.Warn("error updating backend - finalized block", "name", be.Name, "err", err)
	}

	RecordConsensusBackendUpdateDelay(be, bs.lastUpdate)

	changed := cp.setBackendState(be, peerCount, inSync,
		latestBlockNumber, latestBlockHash,
		safeBlockNumber, finalizedBlockNumber)

	RecordBackendLatestBlock(be, latestBlockNumber)
	RecordBackendSafeBlock(be, safeBlockNumber)
	RecordBackendFinalizedBlock(be, finalizedBlockNumber)

	if changed {
		log.Debug("backend local updated",
			"name", be.Name,
			"peerCount", peerCount,
			"inSync", inSync,
			"latestBlockNumber", latestBlockNumber,
			"latestBlockHash", latestBlockHash,
			"safeBlockNumber", safeBlockNumber,
			"finalizedBlockNumber", finalizedBlockNumber,
			"lastUpdate", bs.lastUpdate)
	}

	// sanity check for latest, safe and finalized block tags
	expectedBlockTags := cp.checkExpectedBlockTags(
		latestBlockNumber,
		bs.safeBlockNumber, safeBlockNumber,
		bs.finalizedBlockNumber, finalizedBlockNumber)

	RecordBackendUnexpectedBlockTags(be, !expectedBlockTags)

	if !expectedBlockTags {
		log.Warn("backend banned - unexpected block tags",
			"backend", be.Name,
			"oldFinalized", bs.finalizedBlockNumber,
			"finalizedBlockNumber", finalizedBlockNumber,
			"oldSafe", bs.safeBlockNumber,
			"safeBlockNumber", safeBlockNumber,
			"latestBlockNumber", latestBlockNumber,
		)
		cp.Ban(be)
	}
}

// checkExpectedBlockTags for unexpected conditions on block tags
// - finalized block number should never decrease
// - safe block number should never decrease
// - finalized block should be <= safe block <= latest block
func (cp *ConsensusPoller) checkExpectedBlockTags(
	currentLatest hexutil.Uint64,
	oldSafe hexutil.Uint64, currentSafe hexutil.Uint64,
	oldFinalized hexutil.Uint64, currentFinalized hexutil.Uint64) bool {
	return currentFinalized >= oldFinalized &&
		currentSafe >= oldSafe &&
		currentFinalized <= currentSafe &&
		currentSafe <= currentLatest
}

// UpdateBackendGroupConsensus resolves the current group consensus based on the local of the backends
func (cp *ConsensusPoller) UpdateBackendGroupConsensus(ctx context.Context) {
	// get the latest block number update the tracker
	currentConsensusBlockNumber := cp.GetLatestBlockNumber()

	// get the candidates for the consensus group
	candidates := cp.getConsensusCandidates()

	// update the lowest latest block number and hash
	//        the lowest safe block number
	//        the lowest finalized block number
	var lowestLatestBlock hexutil.Uint64
	var lowestLatestBlockHash string
	var lowestFinalizedBlock hexutil.Uint64
	var lowestSafeBlock hexutil.Uint64
	for _, bs := range candidates {
		if lowestLatestBlock == 0 || bs.latestBlockNumber < lowestLatestBlock {
			lowestLatestBlock = bs.latestBlockNumber
			lowestLatestBlockHash = bs.latestBlockHash
		}
		if lowestFinalizedBlock == 0 || bs.finalizedBlockNumber < lowestFinalizedBlock {
			lowestFinalizedBlock = bs.finalizedBlockNumber
		}
		if lowestSafeBlock == 0 || bs.safeBlockNumber < lowestSafeBlock {
			lowestSafeBlock = bs.safeBlockNumber
		}
	}

	// find the proposed block among the candidates
	// the proposed block needs have the same hash in the entire consensus group
	proposedBlock := lowestLatestBlock
	proposedBlockHash := lowestLatestBlockHash
	hasConsensus := false
	broken := false

	if lowestLatestBlock > currentConsensusBlockNumber {
		log.Debug("validating consensus on block", "lowestLatestBlock", lowestLatestBlock)
	}

	// if there is a block to propose, check if it is the same in all backends
	if proposedBlock > 0 {
		for !hasConsensus {
			allAgreed := true
			for be := range candidates {
				actualBlockNumber, actualBlockHash, err := cp.fetchBlock(ctx, be, proposedBlock.String())
				if err != nil {
					log.Warn("error updating backend", "name", be.Name, "err", err)
					continue
				}
				if proposedBlockHash == "" {
					proposedBlockHash = actualBlockHash
				}
				blocksDontMatch := (actualBlockNumber != proposedBlock) || (actualBlockHash != proposedBlockHash)
				if blocksDontMatch {
					if currentConsensusBlockNumber >= actualBlockNumber {
						log.Warn("backend broke consensus",
							"name", be.Name,
							"actualBlockNumber", actualBlockNumber,
							"actualBlockHash", actualBlockHash,
							"proposedBlock", proposedBlock,
							"proposedBlockHash", proposedBlockHash)
						broken = true
					}
					allAgreed = false
					break
				}
			}
			if allAgreed {
				hasConsensus = true
			} else {
				// walk one block behind and try again
				proposedBlock -= 1
				proposedBlockHash = ""
				log.Debug("no consensus, now trying", "block:", proposedBlock)
			}
		}
	}

	if broken {
		// propagate event to other interested parts, such as cache invalidator
		for _, l := range cp.listeners {
			l()
		}
		log.Info("consensus broken",
			"currentConsensusBlockNumber", currentConsensusBlockNumber,
			"proposedBlock", proposedBlock,
			"proposedBlockHash", proposedBlockHash)
	}

	// update tracker
	cp.tracker.SetLatestBlockNumber(proposedBlock)
	cp.tracker.SetSafeBlockNumber(lowestSafeBlock)
	cp.tracker.SetFinalizedBlockNumber(lowestFinalizedBlock)

	// update consensus group
	group := make([]*Backend, 0, len(candidates))
	consensusBackendsNames := make([]string, 0, len(candidates))
	filteredBackendsNames := make([]string, 0, len(cp.backendGroup.Backends))
	for _, be := range cp.backendGroup.Backends {
		_, exist := candidates[be]
		if exist {
			group = append(group, be)
			consensusBackendsNames = append(consensusBackendsNames, be.Name)
		} else {
			filteredBackendsNames = append(filteredBackendsNames, be.Name)
		}
	}

	cp.consensusGroupMux.Lock()
	cp.consensusGroup = group
	cp.consensusGroupMux.Unlock()

	RecordGroupConsensusLatestBlock(cp.backendGroup, proposedBlock)
	RecordGroupConsensusSafeBlock(cp.backendGroup, lowestSafeBlock)
	RecordGroupConsensusFinalizedBlock(cp.backendGroup, lowestFinalizedBlock)

	RecordGroupConsensusCount(cp.backendGroup, len(group))
	RecordGroupConsensusFilteredCount(cp.backendGroup, len(filteredBackendsNames))
	RecordGroupTotalCount(cp.backendGroup, len(cp.backendGroup.Backends))

	log.Debug("group local",
		"proposedBlock", proposedBlock,
		"consensusBackends", strings.Join(consensusBackendsNames, ", "),
		"filteredBackends", strings.Join(filteredBackendsNames, ", "))
}

// IsBanned checks if a specific backend is banned
func (cp *ConsensusPoller) IsBanned(be *Backend) bool {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	return bs.IsBanned()
}

// Ban bans a specific backend
func (cp *ConsensusPoller) Ban(be *Backend) {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	bs.bannedUntil = time.Now().Add(cp.banPeriod)

	// when we ban a node, we give it the chance to start update any block when it is back
	bs.latestBlockNumber = 0
	bs.safeBlockNumber = 0
	bs.finalizedBlockNumber = 0
}

// Unban removes any bans update the backends
func (cp *ConsensusPoller) Unban(be *Backend) {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	bs.bannedUntil = time.Now().Add(-10 * time.Hour)
}

// Reset reset all backend states
func (cp *ConsensusPoller) Reset() {
	for _, be := range cp.backendGroup.Backends {
		cp.backendState[be] = &backendState{}
	}
}

// fetchBlock is a convenient wrapper to make a request to get a block directly update the backend
func (cp *ConsensusPoller) fetchBlock(ctx context.Context, be *Backend, block string) (blockNumber hexutil.Uint64, blockHash string, err error) {
	var rpcRes RPCRes
	err = be.ForwardRPC(ctx, &rpcRes, "67", "eth_getBlockByNumber", block, false)
	if err != nil {
		return 0, "", err
	}

	jsonMap, ok := rpcRes.Result.(map[string]interface{})
	if !ok {
		return 0, "", fmt.Errorf("unexpected response to eth_getBlockByNumber on backend %s", be.Name)
	}
	blockNumber = hexutil.Uint64(hexutil.MustDecodeUint64(jsonMap["number"].(string)))
	blockHash = jsonMap["hash"].(string)

	return
}

// getPeerCount is a convenient wrapper to retrieve the current peer count update the backend
func (cp *ConsensusPoller) getPeerCount(ctx context.Context, be *Backend) (count uint64, err error) {
	var rpcRes RPCRes
	err = be.ForwardRPC(ctx, &rpcRes, "67", "net_peerCount")
	if err != nil {
		return 0, err
	}

	jsonMap, ok := rpcRes.Result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected response to net_peerCount on backend %s", be.Name)
	}

	count = hexutil.MustDecodeUint64(jsonMap)

	return count, nil
}

// isInSync is a convenient wrapper to check if the backend is in sync update the network
func (cp *ConsensusPoller) isInSync(ctx context.Context, be *Backend) (result bool, err error) {
	var rpcRes RPCRes
	err = be.ForwardRPC(ctx, &rpcRes, "67", "eth_syncing")
	if err != nil {
		return false, err
	}

	var res bool
	switch typed := rpcRes.Result.(type) {
	case bool:
		syncing := typed
		res = !syncing
	case string:
		syncing, err := strconv.ParseBool(typed)
		if err != nil {
			return false, err
		}
		res = !syncing
	default:
		// result is a json when not in sync
		res = false
	}

	return res, nil
}

// getBackendState creates a copy of backend local so that the caller can use it without locking
func (cp *ConsensusPoller) getBackendState(be *Backend) *backendState {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()

	return &backendState{
		latestBlockNumber:    bs.latestBlockNumber,
		latestBlockHash:      bs.latestBlockHash,
		safeBlockNumber:      bs.safeBlockNumber,
		finalizedBlockNumber: bs.finalizedBlockNumber,
		peerCount:            bs.peerCount,
		inSync:               bs.inSync,
		lastUpdate:           bs.lastUpdate,
		bannedUntil:          bs.bannedUntil,
	}
}

func (cp *ConsensusPoller) setBackendState(be *Backend, peerCount uint64, inSync bool,
	latestBlockNumber hexutil.Uint64, latestBlockHash string,
	safeBlockNumber hexutil.Uint64,
	finalizedBlockNumber hexutil.Uint64) bool {
	bs := cp.backendState[be]
	bs.backendStateMux.Lock()
	changed := bs.latestBlockHash != latestBlockHash
	bs.peerCount = peerCount
	bs.inSync = inSync
	bs.latestBlockNumber = latestBlockNumber
	bs.latestBlockHash = latestBlockHash
	bs.finalizedBlockNumber = finalizedBlockNumber
	bs.safeBlockNumber = safeBlockNumber
	bs.lastUpdate = time.Now()
	bs.backendStateMux.Unlock()
	return changed
}

// getConsensusCandidates find out what backends are the candidates to be in the consensus group
// and create a copy of current their local
//
// a candidate is a serving node within the following conditions:
//   - not banned
//   - healthy (network latency and error rate)
//   - with minimum peer count
//   - in sync
//   - updated recently
//   - not lagging latest block
func (cp *ConsensusPoller) getConsensusCandidates() map[*Backend]*backendState {
	candidates := make(map[*Backend]*backendState, len(cp.backendGroup.Backends))

	for _, be := range cp.backendGroup.Backends {
		bs := cp.getBackendState(be)
		if be.forcedCandidate {
			candidates[be] = bs
			continue
		}
		if bs.IsBanned() {
			continue
		}
		if !be.IsHealthy() {
			continue
		}
		if !be.skipPeerCountCheck && bs.peerCount < cp.minPeerCount {
			continue
		}
		if !bs.inSync {
			continue
		}
		if bs.lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now()) {
			continue
		}

		candidates[be] = bs
	}

	// find the highest block, in order to use it defining the highest non-lagging ancestor block
	var highestLatestBlock hexutil.Uint64
	for _, bs := range candidates {
		if bs.latestBlockNumber > highestLatestBlock {
			highestLatestBlock = bs.latestBlockNumber
		}
	}

	// find the highest common ancestor block
	lagging := make([]*Backend, 0, len(candidates))
	for be, bs := range candidates {
		// check if backend is lagging behind the highest block
		if uint64(highestLatestBlock-bs.latestBlockNumber) > cp.maxBlockLag {
			lagging = append(lagging, be)
		}
	}

	// remove lagging backends update the candidates
	for _, be := range lagging {
		delete(candidates, be)
	}

	return candidates
}
