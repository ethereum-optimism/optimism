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

// ConsensusPoller checks the consensus state for each member of a BackendGroup
// resolves the highest common block for multiple nodes, and reconciles the consensus
// in case of block hash divergence to minimize re-orgs
type ConsensusPoller struct {
	cancelFunc context.CancelFunc
	listeners  []OnConsensusBroken

	backendGroup      *BackendGroup
	backendState      map[*Backend]*backendState
	consensusGroupMux sync.Mutex
	consensusGroup    []*Backend

	tracker      ConsensusTracker
	asyncHandler ConsensusAsyncHandler

	minPeerCount uint64

	banPeriod          time.Duration
	maxUpdateThreshold time.Duration
	maxBlockLag        uint64
}

type backendState struct {
	backendStateMux sync.Mutex

	latestBlockNumber hexutil.Uint64
	latestBlockHash   string

	finalizedBlockNumber hexutil.Uint64
	safeBlockNumber      hexutil.Uint64

	peerCount uint64
	inSync    bool

	lastUpdate time.Time

	bannedUntil time.Time
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

// GetFinalizedBlockNumber returns the `finalized` agreed block number in a consensus
func (ct *ConsensusPoller) GetFinalizedBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetFinalizedBlockNumber()
}

// GetSafeBlockNumber returns the `safe` agreed block number in a consensus
func (ct *ConsensusPoller) GetSafeBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetSafeBlockNumber()
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

func WithMinPeerCount(minPeerCount uint64) ConsensusOpt {
	return func(cp *ConsensusPoller) {
		cp.minPeerCount = minPeerCount
	}
}

func NewConsensusPoller(bg *BackendGroup, opts ...ConsensusOpt) *ConsensusPoller {
	ctx, cancelFunc := context.WithCancel(context.Background())

	state := make(map[*Backend]*backendState, len(bg.Backends))
	for _, be := range bg.Backends {
		state[be] = &backendState{}
	}

	cp := &ConsensusPoller{
		cancelFunc:   cancelFunc,
		backendGroup: bg,
		backendState: state,

		banPeriod:          5 * time.Minute,
		maxUpdateThreshold: 30 * time.Second,
		maxBlockLag:        50,
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

	cp.asyncHandler.Init()

	return cp
}

// UpdateBackend refreshes the consensus state of a single backend
func (cp *ConsensusPoller) UpdateBackend(ctx context.Context, be *Backend) {
	banned := cp.IsBanned(be)
	RecordConsensusBackendBanned(be, banned)

	if banned {
		log.Debug("skipping backend - banned", "backend", be.Name)
		return
	}

	// if backend is not healthy state we'll only resume checking it after ban
	if !be.IsHealthy() {
		log.Warn("backend banned - not online or not healthy", "backend", be.Name)
		cp.Ban(be)
		return
	}

	// if backend it not in sync we'll check again after ban
	inSync, err := cp.isInSync(ctx, be)
	RecordConsensusBackendInSync(be, err == nil && inSync)
	if err != nil {
		log.Warn("error updating backend sync state", "name", be.Name, "err", err)
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
		log.Warn("error updating backend", "name", be.Name, "err", err)
	}

	finalizedBlockNumber, _, err := cp.fetchBlock(ctx, be, "finalized")
	if err != nil {
		log.Warn("error updating backend", "name", be.Name, "err", err)
	}

	safeBlockNumber, _, err := cp.fetchBlock(ctx, be, "safe")
	if err != nil {
		log.Warn("error updating backend", "name", be.Name, "err", err)
	}

	changed, updateDelay := cp.setBackendState(be, peerCount, inSync,
		latestBlockNumber, latestBlockHash,
		finalizedBlockNumber, safeBlockNumber)

	if changed {
		RecordBackendLatestBlock(be, latestBlockNumber)
		RecordConsensusBackendUpdateDelay(be, updateDelay)
		log.Debug("backend state updated",
			"name", be.Name,
			"peerCount", peerCount,
			"inSync", inSync,
			"latestBlockNumber", latestBlockNumber,
			"latestBlockHash", latestBlockHash,
			"finalizedBlockNumber", finalizedBlockNumber,
			"safeBlockNumber", safeBlockNumber,
			"updateDelay", updateDelay)
	}
}

// UpdateBackendGroupConsensus resolves the current group consensus based on the state of the backends
func (cp *ConsensusPoller) UpdateBackendGroupConsensus(ctx context.Context) {
	var highestLatestBlock hexutil.Uint64

	var lowestLatestBlock hexutil.Uint64
	var lowestLatestBlockHash string

	var lowestFinalizedBlock hexutil.Uint64
	var lowestSafeBlock hexutil.Uint64

	currentConsensusBlockNumber := cp.GetLatestBlockNumber()

	// find the highest block, in order to use it defining the highest non-lagging ancestor block
	for _, be := range cp.backendGroup.Backends {
		peerCount, inSync, backendLatestBlockNumber, _, _, _, lastUpdate, _ := cp.getBackendState(be)

		if !be.skipPeerCountCheck && peerCount < cp.minPeerCount {
			continue
		}
		if !inSync {
			continue
		}
		if lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now()) {
			continue
		}

		if backendLatestBlockNumber > highestLatestBlock {
			highestLatestBlock = backendLatestBlockNumber
		}
	}

	// find the highest common ancestor block
	for _, be := range cp.backendGroup.Backends {
		peerCount, inSync, backendLatestBlockNumber, backendLatestBlockHash, backendFinalizedBlockNumber, backendSafeBlockNumber, lastUpdate, _ := cp.getBackendState(be)

		if !be.skipPeerCountCheck && peerCount < cp.minPeerCount {
			continue
		}
		if !inSync {
			continue
		}
		if lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now()) {
			continue
		}

		// check if backend is lagging behind the highest block
		if backendLatestBlockNumber < highestLatestBlock && uint64(highestLatestBlock-backendLatestBlockNumber) > cp.maxBlockLag {
			continue
		}

		if lowestLatestBlock == 0 || backendLatestBlockNumber < lowestLatestBlock {
			lowestLatestBlock = backendLatestBlockNumber
			lowestLatestBlockHash = backendLatestBlockHash
		}

		if lowestFinalizedBlock == 0 || backendFinalizedBlockNumber < lowestFinalizedBlock {
			lowestFinalizedBlock = backendFinalizedBlockNumber
		}

		if lowestSafeBlock == 0 || backendSafeBlockNumber < lowestSafeBlock {
			lowestSafeBlock = backendSafeBlockNumber
		}
	}

	// no block to propose (i.e. initializing consensus)
	if lowestLatestBlock == 0 {
		return
	}

	proposedBlock := lowestLatestBlock
	proposedBlockHash := lowestLatestBlockHash
	hasConsensus := false

	// check if everybody agrees on the same block hash
	consensusBackends := make([]*Backend, 0, len(cp.backendGroup.Backends))
	consensusBackendsNames := make([]string, 0, len(cp.backendGroup.Backends))
	filteredBackendsNames := make([]string, 0, len(cp.backendGroup.Backends))

	if lowestLatestBlock > currentConsensusBlockNumber {
		log.Debug("validating consensus on block", "lowestLatestBlock", lowestLatestBlock)
	}

	broken := false
	for !hasConsensus {
		allAgreed := true
		consensusBackends = consensusBackends[:0]
		filteredBackendsNames = filteredBackendsNames[:0]
		for _, be := range cp.backendGroup.Backends {
			/*
				a serving node needs to be:
				- healthy (network)
				- updated recently
				- not banned
				- with minimum peer count
				- not lagging latest block
				- in sync
			*/

			peerCount, inSync, latestBlockNumber, _, _, _, lastUpdate, bannedUntil := cp.getBackendState(be)
			notUpdated := lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now())
			isBanned := time.Now().Before(bannedUntil)
			notEnoughPeers := !be.skipPeerCountCheck && peerCount < cp.minPeerCount
			lagging := latestBlockNumber < proposedBlock
			if !be.IsHealthy() || notUpdated || isBanned || notEnoughPeers || lagging || !inSync {
				filteredBackendsNames = append(filteredBackendsNames, be.Name)
				continue
			}

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
					log.Warn("backend broke consensus", "name", be.Name, "blockNum", actualBlockNumber, "proposedBlockNum", proposedBlock, "blockHash", actualBlockHash, "proposedBlockHash", proposedBlockHash)
					broken = true
				}
				allAgreed = false
				break
			}
			consensusBackends = append(consensusBackends, be)
			consensusBackendsNames = append(consensusBackendsNames, be.Name)
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

	if broken {
		// propagate event to other interested parts, such as cache invalidator
		for _, l := range cp.listeners {
			l()
		}
		log.Info("consensus broken", "currentConsensusBlockNumber", currentConsensusBlockNumber, "proposedBlock", proposedBlock, "proposedBlockHash", proposedBlockHash)
	}

	cp.tracker.SetLatestBlockNumber(proposedBlock)
	cp.tracker.SetFinalizedBlockNumber(lowestFinalizedBlock)
	cp.tracker.SetSafeBlockNumber(lowestSafeBlock)
	cp.consensusGroupMux.Lock()
	cp.consensusGroup = consensusBackends
	cp.consensusGroupMux.Unlock()

	RecordGroupConsensusLatestBlock(cp.backendGroup, proposedBlock)
	RecordGroupConsensusCount(cp.backendGroup, len(consensusBackends))
	RecordGroupConsensusFilteredCount(cp.backendGroup, len(filteredBackendsNames))
	RecordGroupTotalCount(cp.backendGroup, len(cp.backendGroup.Backends))

	log.Debug("group state", "proposedBlock", proposedBlock, "consensusBackends", strings.Join(consensusBackendsNames, ", "), "filteredBackends", strings.Join(filteredBackendsNames, ", "))
}

// IsBanned checks if a specific backend is banned
func (cp *ConsensusPoller) IsBanned(be *Backend) bool {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	return time.Now().Before(bs.bannedUntil)
}

// Ban bans a specific backend
func (cp *ConsensusPoller) Ban(be *Backend) {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	bs.bannedUntil = time.Now().Add(cp.banPeriod)
}

// Unban remove any bans from the backends
func (cp *ConsensusPoller) Unban() {
	for _, be := range cp.backendGroup.Backends {
		bs := cp.backendState[be]
		bs.backendStateMux.Lock()
		bs.bannedUntil = time.Now().Add(-10 * time.Hour)
		bs.backendStateMux.Unlock()
	}
}

// fetchBlock Convenient wrapper to make a request to get a block directly from the backend
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

// getPeerCount Convenient wrapper to retrieve the current peer count from the backend
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

// isInSync is a convenient wrapper to check if the backend is in sync from the network
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

func (cp *ConsensusPoller) getBackendState(be *Backend) (peerCount uint64, inSync bool,
	latestBlockNumber hexutil.Uint64, latestBlockHash string,
	finalizedBlockNumber hexutil.Uint64,
	safeBlockNumber hexutil.Uint64,
	lastUpdate time.Time, bannedUntil time.Time) {
	bs := cp.backendState[be]
	defer bs.backendStateMux.Unlock()
	bs.backendStateMux.Lock()
	peerCount = bs.peerCount
	inSync = bs.inSync
	latestBlockNumber = bs.latestBlockNumber
	latestBlockHash = bs.latestBlockHash
	finalizedBlockNumber = bs.finalizedBlockNumber
	safeBlockNumber = bs.safeBlockNumber
	lastUpdate = bs.lastUpdate
	bannedUntil = bs.bannedUntil
	return
}

func (cp *ConsensusPoller) setBackendState(be *Backend, peerCount uint64, inSync bool,
	latestBlockNumber hexutil.Uint64, latestBlockHash string,
	finalizedBlockNumber hexutil.Uint64,
	safeBlockNumber hexutil.Uint64) (changed bool, updateDelay time.Duration) {
	bs := cp.backendState[be]
	bs.backendStateMux.Lock()
	changed = bs.latestBlockHash != latestBlockHash
	bs.peerCount = peerCount
	bs.inSync = inSync
	bs.latestBlockNumber = latestBlockNumber
	bs.latestBlockHash = latestBlockHash
	bs.finalizedBlockNumber = finalizedBlockNumber
	bs.safeBlockNumber = safeBlockNumber
	updateDelay = time.Since(bs.lastUpdate)
	bs.lastUpdate = time.Now()
	bs.backendStateMux.Unlock()
	return
}
