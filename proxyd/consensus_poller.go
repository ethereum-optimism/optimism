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

// ConsensusPoller checks the consensus state for each member of a BackendGroup
// resolves the highest common block for multiple nodes, and reconciles the consensus
// in case of block hash divergence to minimize re-orgs
type ConsensusPoller struct {
	cancelFunc context.CancelFunc

	backendGroup      *BackendGroup
	backendState      map[*Backend]*backendState
	consensusGroupMux sync.Mutex
	consensusGroup    []*Backend

	tracker      ConsensusTracker
	asyncHandler ConsensusAsyncHandler

	minPeerCount uint64

	banPeriod          time.Duration
	maxUpdateThreshold time.Duration
}

type backendState struct {
	backendStateMux sync.Mutex

	latestBlockNumber hexutil.Uint64
	latestBlockHash   string
	peerCount         uint64

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

// GetConsensusBlockNumber returns the agreed block number in a consensus
func (ct *ConsensusPoller) GetConsensusBlockNumber() hexutil.Uint64 {
	return ct.tracker.GetConsensusBlockNumber()
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
	bs := cp.backendState[be]
	if time.Now().Before(bs.bannedUntil) {
		log.Warn("skipping backend banned", "backend", be.Name, "bannedUntil", bs.bannedUntil)
		return
	}

	// if backend it not online or not in a health state we'll only resume checkin it after ban
	if !be.Online() || !be.IsHealthy() {
		log.Warn("backend banned - not online or not healthy", "backend", be.Name, "bannedUntil", bs.bannedUntil)
		bs.bannedUntil = time.Now().Add(cp.banPeriod)
	}

	// if backend it not in sync we'll check again after ban
	inSync, err := cp.isInSync(ctx, be)
	if err != nil || !inSync {
		log.Warn("backend banned - not in sync", "backend", be.Name, "bannedUntil", bs.bannedUntil)
		bs.bannedUntil = time.Now().Add(cp.banPeriod)
	}

	// if backend exhausted rate limit we'll skip it for now
	if be.IsRateLimited() {
		return
	}

	peerCount, err := cp.getPeerCount(ctx, be)
	if err != nil {
		log.Warn("error updating backend", "name", be.Name, "err", err)
		return
	}

	latestBlockNumber, latestBlockHash, err := cp.fetchBlock(ctx, be, "latest")
	if err != nil {
		log.Warn("error updating backend", "name", be.Name, "err", err)
		return
	}

	changed := cp.setBackendState(be, peerCount, latestBlockNumber, latestBlockHash)

	if changed {
		RecordBackendLatestBlock(be, latestBlockNumber)
		log.Info("backend state updated", "name", be.Name, "state", bs)
	}
}

// UpdateBackendGroupConsensus resolves the current group consensus based on the state of the backends
func (cp *ConsensusPoller) UpdateBackendGroupConsensus(ctx context.Context) {
	var lowestBlock hexutil.Uint64
	var lowestBlockHash string

	currentConsensusBlockNumber := cp.GetConsensusBlockNumber()

	for _, be := range cp.backendGroup.Backends {
		peerCount, backendLatestBlockNumber, backendLatestBlockHash, lastUpdate := cp.getBackendState(be)

		if peerCount < cp.minPeerCount {
			continue
		}
		if lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now()) {
			continue
		}

		if lowestBlock == 0 || backendLatestBlockNumber < lowestBlock {
			lowestBlock = backendLatestBlockNumber
			lowestBlockHash = backendLatestBlockHash
		}
	}

	// no block to propose (i.e. initializing consensus)
	if lowestBlock == 0 {
		return
	}

	proposedBlock := lowestBlock
	proposedBlockHash := lowestBlockHash
	hasConsensus := false

	// check if everybody agrees on the same block hash
	consensusBackends := make([]*Backend, 0, len(cp.backendGroup.Backends))
	consensusBackendsNames := make([]string, 0, len(cp.backendGroup.Backends))
	filteredBackendsNames := make([]string, 0, len(cp.backendGroup.Backends))

	if lowestBlock > currentConsensusBlockNumber {
		log.Info("validating consensus on block", lowestBlock)
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
				- not rate limited
				- online
				- not banned
				- with minimum peer count
				- updated recently
			*/
			bs := cp.backendState[be]
			notUpdated := bs.lastUpdate.Add(cp.maxUpdateThreshold).Before(time.Now())
			isBanned := time.Now().Before(bs.bannedUntil)
			notEnoughPeers := bs.peerCount < cp.minPeerCount
			if !be.IsHealthy() || be.IsRateLimited() || !be.Online() || notUpdated || isBanned || notEnoughPeers {
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
			log.Info("no consensus, now trying", "block:", proposedBlock)
		}
	}

	if broken {
		// propagate event to other interested parts, such as cache invalidator
		log.Info("consensus broken", "currentConsensusBlockNumber", currentConsensusBlockNumber, "proposedBlock", proposedBlock, "proposedBlockHash", proposedBlockHash)
	}

	cp.tracker.SetConsensusBlockNumber(proposedBlock)
	RecordGroupConsensusLatestBlock(cp.backendGroup, proposedBlock)
	cp.consensusGroupMux.Lock()
	cp.consensusGroup = consensusBackends
	cp.consensusGroupMux.Unlock()

	log.Info("group state", "proposedBlock", proposedBlock, "consensusBackends", strings.Join(consensusBackendsNames, ", "), "filteredBackends", strings.Join(filteredBackendsNames, ", "))
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

// isSyncing Convenient wrapper to check if the backend is syncing from the network
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

func (cp *ConsensusPoller) getBackendState(be *Backend) (peerCount uint64, blockNumber hexutil.Uint64, blockHash string, lastUpdate time.Time) {
	bs := cp.backendState[be]
	bs.backendStateMux.Lock()
	peerCount = bs.peerCount
	blockNumber = bs.latestBlockNumber
	blockHash = bs.latestBlockHash
	lastUpdate = bs.lastUpdate
	bs.backendStateMux.Unlock()
	return
}

func (cp *ConsensusPoller) setBackendState(be *Backend, peerCount uint64, blockNumber hexutil.Uint64, blockHash string) (changed bool) {
	bs := cp.backendState[be]
	bs.backendStateMux.Lock()
	changed = bs.latestBlockHash != blockHash
	bs.peerCount = peerCount
	bs.latestBlockNumber = blockNumber
	bs.latestBlockHash = blockHash
	bs.lastUpdate = time.Now()
	bs.backendStateMux.Unlock()
	return
}
