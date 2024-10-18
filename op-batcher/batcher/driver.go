package batcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	_ "net/http/pprof"
	"sync"
	"time"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"
)

var (
	ErrBatcherNotRunning = errors.New("batcher is not running")
	emptyTxData          = txData{
		frames: []frameData{
			{
				data: []byte{},
			},
		},
	}
)

type txRef struct {
	id       txID
	isCancel bool
	isBlob   bool
}

func (r txRef) String() string {
	return r.string(func(id txID) string { return id.String() })
}

func (r txRef) TerminalString() string {
	return r.string(func(id txID) string { return id.TerminalString() })
}

func (r txRef) string(txIDStringer func(txID) string) string {
	if r.isCancel {
		if r.isBlob {
			return "blob-cancellation"
		} else {
			return "calldata-cancellation"
		}
	}
	return txIDStringer(r.id)
}

type L1Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

type L2Client interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type RollupClient interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

// DriverSetup is the collection of input/output interfaces and configuration that the driver operates on.
type DriverSetup struct {
	Log              log.Logger
	Metr             metrics.Metricer
	RollupConfig     *rollup.Config
	Config           BatcherConfig
	Txmgr            txmgr.TxManager
	L1Client         L1Client
	EndpointProvider dial.L2EndpointProvider
	ChannelConfig    ChannelConfigProvider
	AltDA            *altda.DAClient
}

// BatchSubmitter encapsulates a service responsible for submitting L2 tx
// batches to L1 for availability.
type BatchSubmitter struct {
	DriverSetup

	wg sync.WaitGroup

	shutdownCtx       context.Context
	cancelShutdownCtx context.CancelFunc
	killCtx           context.Context
	cancelKillCtx     context.CancelFunc

	mutex   sync.Mutex
	running bool

	txpoolMutex       sync.Mutex // guards txpoolState and txpoolBlockedBlob
	txpoolState       TxPoolState
	txpoolBlockedBlob bool

	// lastStoredBlock is the last block loaded into `state`. If it is empty it should be set to the l2 safe head.
	lastStoredBlock eth.BlockID
	lastL1Tip       eth.L1BlockRef

	state *channelManager
}

// NewBatchSubmitter initializes the BatchSubmitter driver from a preconfigured DriverSetup
func NewBatchSubmitter(setup DriverSetup) *BatchSubmitter {
	return &BatchSubmitter{
		DriverSetup: setup,
		state:       NewChannelManager(setup.Log, setup.Metr, setup.ChannelConfig, setup.RollupConfig),
	}
}

func (l *BatchSubmitter) StartBatchSubmitting() error {
	l.Log.Info("Starting Batch Submitter")

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.running {
		return errors.New("batcher is already running")
	}
	l.running = true

	l.shutdownCtx, l.cancelShutdownCtx = context.WithCancel(context.Background())
	l.killCtx, l.cancelKillCtx = context.WithCancel(context.Background())
	l.clearState(l.shutdownCtx)
	l.lastStoredBlock = eth.BlockID{}

	if l.Config.WaitNodeSync {
		err := l.waitNodeSync()
		if err != nil {
			return fmt.Errorf("error waiting for node sync: %w", err)
		}
	}

	receiptsCh := make(chan txmgr.TxReceipt[txRef])
	receiptsLoopCtx, cancelReceiptsLoopCtx := context.WithCancel(context.Background())
	l.wg.Add(2)
	go l.receiptsLoop(receiptsLoopCtx, receiptsCh)                  // receives from receiptsCh
	go l.mainLoop(l.shutdownCtx, receiptsCh, cancelReceiptsLoopCtx) // sends on receiptsCh

	l.Log.Info("Batch Submitter started")
	return nil
}

func (l *BatchSubmitter) StopBatchSubmittingIfRunning(ctx context.Context) error {
	err := l.StopBatchSubmitting(ctx)
	if errors.Is(err, ErrBatcherNotRunning) {
		return nil
	}
	return err
}

// StopBatchSubmitting stops the batch-submitter loop, and force-kills if the provided ctx is done.
func (l *BatchSubmitter) StopBatchSubmitting(ctx context.Context) error {
	l.Log.Info("Stopping Batch Submitter")

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.running {
		return ErrBatcherNotRunning
	}
	l.running = false

	// go routine will call cancelKill() if the passed in ctx is ever Done
	cancelKill := l.cancelKillCtx
	wrapped, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-wrapped.Done()
		cancelKill()
	}()

	l.cancelShutdownCtx()
	l.wg.Wait()
	l.cancelKillCtx()

	l.Log.Info("Batch Submitter stopped")
	return nil
}

// loadBlocksIntoState loads all blocks since the previous stored block
// It does the following:
//  1. Fetch the sync status of the sequencer
//  2. Check if the sync status is valid or if we are all the way up to date
//  3. Check if it needs to initialize state OR it is lagging (todo: lagging just means race condition?)
//  4. Load all new blocks into the local state.
//  5. Dequeue blocks from local state which are now safe.
//
// If there is a reorg, it will reset the last stored block but not clear the internal state so
// the state can be flushed to L1.
func (l *BatchSubmitter) loadBlocksIntoState(syncStatus eth.SyncStatus, ctx context.Context) error {
	start, end, err := l.calculateL2BlockRangeToStore(syncStatus)
	if err != nil {
		l.Log.Warn("Error calculating L2 block range", "err", err)
		return err
	} else if start.Number >= end.Number {
		return errors.New("start number is >= end number")
	}

	var latestBlock *types.Block
	// Add all blocks to "state"
	for i := start.Number + 1; i < end.Number+1; i++ {
		block, err := l.loadBlockIntoState(ctx, i)
		if errors.Is(err, ErrReorg) {
			l.Log.Warn("Found L2 reorg", "block_number", i)
			l.lastStoredBlock = eth.BlockID{}
			return err
		} else if err != nil {
			l.Log.Warn("Failed to load block into state", "err", err)
			return err
		}
		l.lastStoredBlock = eth.ToBlockID(block)
		latestBlock = block
	}

	l2ref, err := derive.L2BlockToBlockRef(l.RollupConfig, latestBlock)
	if err != nil {
		l.Log.Warn("Invalid L2 block loaded into state", "err", err)
		return err
	}

	l.Metr.RecordL2BlocksLoaded(l2ref)
	return nil
}

// loadBlockIntoState fetches & stores a single block into `state`. It returns the block it loaded.
func (l *BatchSubmitter) loadBlockIntoState(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	l2Client, err := l.EndpointProvider.EthClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting L2 client: %w", err)
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()

	block, err := l2Client.BlockByNumber(cCtx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("getting L2 block: %w", err)
	}

	if err := l.state.AddL2Block(block); err != nil {
		return nil, fmt.Errorf("adding L2 block to state: %w", err)
	}

	l.Log.Info("Added L2 block to local state", "block", eth.ToBlockID(block), "tx_count", len(block.Transactions()), "time", block.Time())
	return block, nil
}

func (l *BatchSubmitter) getSyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	rollupClient, err := l.EndpointProvider.RollupClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting rollup client: %w", err)
	}
	cCtx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()
	return rollupClient.SyncStatus(cCtx)
}

// calculateL2BlockRangeToStore determines the range (start,end] that should be loaded into the local state.
// It also takes care of initializing some local state (i.e. will modify l.lastStoredBlock in certain conditions
// as well as garbage collecting blocks which became safe)
func (l *BatchSubmitter) calculateL2BlockRangeToStore(syncStatus eth.SyncStatus) (eth.BlockID, eth.BlockID, error) {
	if syncStatus.HeadL1 == (eth.L1BlockRef{}) {
		return eth.BlockID{}, eth.BlockID{}, errors.New("empty sync status")
	}

	// Check last stored to see if it needs to be set on startup OR set if is lagged behind.
	// It lagging implies that the op-node processed some batches that were submitted prior to the current instance of the batcher being alive.
	if l.lastStoredBlock == (eth.BlockID{}) {
		l.Log.Info("Starting batch-submitter work at safe-head", "safe", syncStatus.SafeL2)
		l.lastStoredBlock = syncStatus.SafeL2.ID()
	} else if l.lastStoredBlock.Number < syncStatus.SafeL2.Number {
		l.Log.Warn("Last submitted block lagged behind L2 safe head: batch submission will continue from the safe head now", "last", l.lastStoredBlock, "safe", syncStatus.SafeL2)
		l.lastStoredBlock = syncStatus.SafeL2.ID()
	}

	// Check if we should even attempt to load any blocks. TODO: May not need this check
	if syncStatus.SafeL2.Number >= syncStatus.UnsafeL2.Number {
		return eth.BlockID{}, eth.BlockID{}, fmt.Errorf("L2 safe head(%d) ahead of L2 unsafe head(%d)", syncStatus.SafeL2.Number, syncStatus.UnsafeL2.Number)
	}

	return l.lastStoredBlock, syncStatus.UnsafeL2.ID(), nil
}

// The following things occur:
// New L2 block (reorg or not)
// L1 transaction is confirmed
//
// What the batcher does:
// Ensure that channels are created & submitted as frames for an L2 range
//
// Error conditions:
// Submitted batch, but it is not valid
// Missed L2 block somehow.

type TxPoolState int

const (
	// Txpool states.  Possible state transitions:
	//   TxpoolGood -> TxpoolBlocked:
	//     happens when ErrAlreadyReserved is ever returned by the TxMgr.
	//   TxpoolBlocked -> TxpoolCancelPending:
	//     happens once the send loop detects the txpool is blocked, and results in attempting to
	//     send a cancellation transaction.
	//   TxpoolCancelPending -> TxpoolGood:
	//     happens once the cancel transaction completes, whether successfully or in error.
	TxpoolGood TxPoolState = iota
	TxpoolBlocked
	TxpoolCancelPending
)

// setTxPoolState locks the mutex, sets the parameters to the supplied ones, and release the mutex.
func (l *BatchSubmitter) setTxPoolState(txPoolState TxPoolState, txPoolBlockedBlob bool) {
	l.txpoolMutex.Lock()
	l.txpoolState = txPoolState
	l.txpoolBlockedBlob = txPoolBlockedBlob
	l.txpoolMutex.Unlock()
}

// receiptsLoop handles transaction receipts from the DA layer
func (l *BatchSubmitter) receiptsLoop(ctx context.Context, receiptsCh chan txmgr.TxReceipt[txRef]) {
	defer l.wg.Done()
	for {
		select {
		case r := <-receiptsCh:
			if errors.Is(r.Err, txpool.ErrAlreadyReserved) && l.txpoolState == TxpoolGood {
				l.setTxPoolState(TxpoolBlocked, r.ID.isBlob)
				l.Log.Warn("incompatible tx in txpool", "id", r.ID, "is_blob", r.ID.isBlob)
			} else if r.ID.isCancel && l.txpoolState == TxpoolCancelPending {
				// Set state to TxpoolGood even if the cancellation transaction ended in error
				// since the stuck transaction could have cleared while we were waiting.
				l.setTxPoolState(TxpoolGood, l.txpoolBlockedBlob)
				l.Log.Info("txpool may no longer be blocked", "err", r.Err)
			}
			l.Log.Info("Handling receipt", "id", r.ID)
			l.handleReceipt(r)
		case <-ctx.Done():
			l.Log.Info("Receipt processing loop done")
			return
		}
	}
}

// mainLoop periodically:
// -  polls the sequencer,
// -  prunes the channel manager state (i.e. safe blocks)
// -  loads unsafe blocks from the sequencer
// -  drives the creation of channels and frames
// -  sends transactions to the DA layer
func (l *BatchSubmitter) mainLoop(ctx context.Context, receiptsCh chan txmgr.TxReceipt[txRef], receiptsLoopCancel context.CancelFunc) {
	defer l.wg.Done()
	defer receiptsLoopCancel()

	queue := txmgr.NewQueue[txRef](l.killCtx, l.Txmgr, l.Config.MaxPendingTransactions)
	daGroup := &errgroup.Group{}
	// errgroup with limit of 0 means no goroutine is able to run concurrently,
	// so we only set the limit if it is greater than 0.
	if l.Config.MaxConcurrentDARequests > 0 {
		daGroup.SetLimit(int(l.Config.MaxConcurrentDARequests))
	}

	ticker := time.NewTicker(l.Config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			if !l.checkTxpool(queue, receiptsCh) {
				continue
			}

			syncStatus, err := l.getSyncStatus(l.shutdownCtx)
			if err != nil {
				l.Log.Warn("could not get sync status", "err", err)
				continue
			}

			l.state.pruneSafeBlocks(syncStatus.SafeL2)
			l.state.pruneChannels(syncStatus.SafeL2)
			if err := l.loadBlocksIntoState(*syncStatus, l.shutdownCtx); errors.Is(err, ErrReorg) {
				// Wait for any in flight transactions
				// to be ingested by the node before
				// we start loading blocks again.
				err := l.waitNodeSync()
				if err != nil {
					l.Log.Warn("error waiting for node sync", "err", err)
				}
				l.clearState(l.shutdownCtx)
				continue
			}
			l.publishStateToL1(queue, receiptsCh, daGroup)
		case <-ctx.Done():
			l.Log.Info("Main loop done")
			return
		}
	}
}

// waitNodeSync Check to see if there was a batcher tx sent recently that
// still needs more block confirmations before being considered finalized
func (l *BatchSubmitter) waitNodeSync() error {
	ctx := l.shutdownCtx
	rollupClient, err := l.EndpointProvider.RollupClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get rollup client: %w", err)
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()

	l1Tip, err := l.l1Tip(cCtx)
	if err != nil {
		return fmt.Errorf("failed to retrieve l1 tip: %w", err)
	}

	l1TargetBlock := l1Tip.Number
	if l.Config.CheckRecentTxsDepth != 0 {
		l.Log.Info("Checking for recently submitted batcher transactions on L1")
		recentBlock, found, err := eth.CheckRecentTxs(cCtx, l.L1Client, l.Config.CheckRecentTxsDepth, l.Txmgr.From())
		if err != nil {
			return fmt.Errorf("failed checking recent batcher txs: %w", err)
		}
		l.Log.Info("Checked for recently submitted batcher transactions on L1",
			"l1_head", l1Tip, "l1_recent", recentBlock, "found", found)
		l1TargetBlock = recentBlock
	}

	return dial.WaitRollupSync(l.shutdownCtx, l.Log, rollupClient, l1TargetBlock, time.Second*12)
}

// publishStateToL1 queues up all pending TxData to be published to the L1, returning when there is
// no more data to queue for publishing or if there was an error queing the data.
func (l *BatchSubmitter) publishStateToL1(queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef], daGroup *errgroup.Group) {
	for {
		// if the txmgr is closed, we stop the transaction sending
		if l.Txmgr.IsClosed() {
			l.Log.Info("Txmgr is closed, aborting state publishing")
			return
		}
		if !l.checkTxpool(queue, receiptsCh) {
			l.Log.Info("txpool state is not good, aborting state publishing")
			return
		}
		err := l.publishTxToL1(l.killCtx, queue, receiptsCh, daGroup)
		if err != nil {
			if err != io.EOF {
				l.Log.Error("Error publishing tx to l1", "err", err)
			}
			return
		}
	}
}

// clearState clears the state of the channel manager
func (l *BatchSubmitter) clearState(ctx context.Context) {
	l.Log.Info("Clearing state")
	defer l.Log.Info("State cleared")

	clearStateWithL1Origin := func() bool {
		l1SafeOrigin, err := l.safeL1Origin(ctx)
		if err != nil {
			l.Log.Warn("Failed to query L1 safe origin, will retry", "err", err)
			return false
		} else {
			l.Log.Info("Clearing state with safe L1 origin", "origin", l1SafeOrigin)
			l.state.Clear(l1SafeOrigin)
			return true
		}
	}

	// Attempt to set the L1 safe origin and clear the state, if fetching fails -- fall through to an infinite retry
	if clearStateWithL1Origin() {
		return
	}

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if clearStateWithL1Origin() {
				return
			}
		case <-ctx.Done():
			l.Log.Warn("Clearing state cancelled")
			l.state.Clear(eth.BlockID{})
			return
		}
	}
}

// publishTxToL1 submits a single state tx to the L1
func (l *BatchSubmitter) publishTxToL1(ctx context.Context, queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef], daGroup *errgroup.Group) error {
	// send all available transactions
	l1tip, err := l.l1Tip(ctx)
	if err != nil {
		l.Log.Error("Failed to query L1 tip", "err", err)
		return err
	}
	l.recordL1Tip(l1tip)

	// Collect next transaction data. This pulls data out of the channel, so we need to make sure
	// to put it back if ever da or txmgr requests fail, by calling l.recordFailedDARequest/recordFailedTx.
	txdata, err := l.state.TxData(l1tip.ID())

	if err == io.EOF {
		l.Log.Trace("No transaction data available")
		return err
	} else if err != nil {
		l.Log.Error("Unable to get tx data", "err", err)
		return err
	}

	if err = l.sendTransaction(txdata, queue, receiptsCh, daGroup); err != nil {
		return fmt.Errorf("BatchSubmitter.sendTransaction failed: %w", err)
	}
	return nil
}

func (l *BatchSubmitter) safeL1Origin(ctx context.Context) (eth.BlockID, error) {
	c, err := l.EndpointProvider.RollupClient(ctx)
	if err != nil {
		log.Error("Failed to get rollup client", "err", err)
		return eth.BlockID{}, fmt.Errorf("safe l1 origin: error getting rollup client: %w", err)
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()

	status, err := c.SyncStatus(cCtx)
	if err != nil {
		log.Error("Failed to get sync status", "err", err)
		return eth.BlockID{}, fmt.Errorf("safe l1 origin: error getting sync status: %w", err)
	}

	// If the safe L2 block origin is 0, we are at the genesis block and should use the L1 origin from the rollup config.
	if status.SafeL2.L1Origin.Number == 0 {
		return l.RollupConfig.Genesis.L1, nil
	}

	return status.SafeL2.L1Origin, nil
}

// cancelBlockingTx creates an empty transaction of appropriate type to cancel out the incompatible
// transaction stuck in the txpool. In the future we might send an actual batch transaction instead
// of an empty one to avoid wasting the tx fee.
func (l *BatchSubmitter) cancelBlockingTx(queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef], isBlockedBlob bool) {
	var candidate *txmgr.TxCandidate
	var err error
	if isBlockedBlob {
		candidate = l.calldataTxCandidate([]byte{})
	} else if candidate, err = l.blobTxCandidate(emptyTxData); err != nil {
		panic(err) // this error should not happen
	}
	l.Log.Warn("sending a cancellation transaction to unblock txpool", "blocked_blob", isBlockedBlob)
	l.sendTx(txData{}, true, candidate, queue, receiptsCh)
}

// publishToAltDAAndL1 posts the txdata to the DA Provider and then sends the commitment to L1.
func (l *BatchSubmitter) publishToAltDAAndL1(txdata txData, queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef], daGroup *errgroup.Group) {
	// sanity checks
	if nf := len(txdata.frames); nf != 1 {
		l.Log.Crit("Unexpected number of frames in calldata tx", "num_frames", nf)
	}
	if txdata.asBlob {
		l.Log.Crit("Unexpected blob txdata with AltDA enabled")
	}

	// when posting txdata to an external DA Provider, we use a goroutine to avoid blocking the main loop
	// since it may take a while for the request to return.
	goroutineSpawned := daGroup.TryGo(func() error {
		// TODO: probably shouldn't be using the global shutdownCtx here, see https://go.dev/blog/context-and-structs
		// but sendTransaction receives l.killCtx as an argument, which currently is only canceled after waiting for the main loop
		// to exit, which would wait on this DA call to finish, which would take a long time.
		// So we prefer to mimic the behavior of txmgr and cancel all pending DA/txmgr requests when the batcher is stopped.
		comm, err := l.AltDA.SetInput(l.shutdownCtx, txdata.CallData())
		if err != nil {
			l.Log.Error("Failed to post input to Alt DA", "error", err)
			// requeue frame if we fail to post to the DA Provider so it can be retried
			// note: this assumes that the da server caches requests, otherwise it might lead to resubmissions of the blobs
			l.recordFailedDARequest(txdata.ID(), err)
			return nil
		}
		l.Log.Info("Set altda input", "commitment", comm, "tx", txdata.ID())
		candidate := l.calldataTxCandidate(comm.TxData())
		l.sendTx(txdata, false, candidate, queue, receiptsCh)
		return nil
	})
	if !goroutineSpawned {
		// We couldn't start the goroutine because the errgroup.Group limit
		// is already reached. Since we can't send the txdata, we have to
		// return it for later processing. We use nil error to skip error logging.
		l.recordFailedDARequest(txdata.ID(), nil)
	}
}

// sendTransaction creates & queues for sending a transaction to the batch inbox address with the given `txData`.
// This call will block if the txmgr queue is at the  max-pending limit.
// The method will block if the queue's MaxPendingTransactions is exceeded.
func (l *BatchSubmitter) sendTransaction(txdata txData, queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef], daGroup *errgroup.Group) error {
	var err error

	// if Alt DA is enabled we post the txdata to the DA Provider and replace it with the commitment.
	if l.Config.UseAltDA {
		l.publishToAltDAAndL1(txdata, queue, receiptsCh, daGroup)
		// we return nil to allow publishStateToL1 to keep processing the next txdata
		return nil
	}

	var candidate *txmgr.TxCandidate
	if txdata.asBlob {
		if candidate, err = l.blobTxCandidate(txdata); err != nil {
			// We could potentially fall through and try a calldata tx instead, but this would
			// likely result in the chain spending more in gas fees than it is tuned for, so best
			// to just fail. We do not expect this error to trigger unless there is a serious bug
			// or configuration issue.
			return fmt.Errorf("could not create blob tx candidate: %w", err)
		}
	} else {
		// sanity check
		if nf := len(txdata.frames); nf != 1 {
			l.Log.Crit("Unexpected number of frames in calldata tx", "num_frames", nf)
		}
		candidate = l.calldataTxCandidate(txdata.CallData())
	}

	l.sendTx(txdata, false, candidate, queue, receiptsCh)
	return nil
}

// sendTx uses the txmgr queue to send the given transaction candidate after setting its
// gaslimit. It will block if the txmgr queue has reached its MaxPendingTransactions limit.
func (l *BatchSubmitter) sendTx(txdata txData, isCancel bool, candidate *txmgr.TxCandidate, queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef]) {
	intrinsicGas, err := core.IntrinsicGas(candidate.TxData, nil, false, true, true, false)
	if err != nil {
		// we log instead of return an error here because txmgr can do its own gas estimation
		l.Log.Error("Failed to calculate intrinsic gas", "err", err)
	} else {
		candidate.GasLimit = intrinsicGas
	}

	queue.Send(txRef{id: txdata.ID(), isCancel: isCancel, isBlob: txdata.asBlob}, *candidate, receiptsCh)
}

func (l *BatchSubmitter) blobTxCandidate(data txData) (*txmgr.TxCandidate, error) {
	blobs, err := data.Blobs()
	if err != nil {
		return nil, fmt.Errorf("generating blobs for tx data: %w", err)
	}
	size := data.Len()
	lastSize := len(data.frames[len(data.frames)-1].data)
	l.Log.Info("Building Blob transaction candidate",
		"size", size, "last_size", lastSize, "num_blobs", len(blobs))
	l.Metr.RecordBlobUsedBytes(lastSize)
	return &txmgr.TxCandidate{
		To:    &l.RollupConfig.BatchInboxAddress,
		Blobs: blobs,
	}, nil
}

func (l *BatchSubmitter) calldataTxCandidate(data []byte) *txmgr.TxCandidate {
	l.Log.Info("Building Calldata transaction candidate", "size", len(data))
	return &txmgr.TxCandidate{
		To:     &l.RollupConfig.BatchInboxAddress,
		TxData: data,
	}
}

func (l *BatchSubmitter) handleReceipt(r txmgr.TxReceipt[txRef]) {
	// Record TX Status
	if r.Err != nil {
		l.recordFailedTx(r.ID.id, r.Err)
	} else {
		l.recordConfirmedTx(r.ID.id, r.Receipt)
	}
}

func (l *BatchSubmitter) recordL1Tip(l1tip eth.L1BlockRef) {
	if l.lastL1Tip == l1tip {
		return
	}
	l.lastL1Tip = l1tip
	l.Metr.RecordLatestL1Block(l1tip)
}

func (l *BatchSubmitter) recordFailedDARequest(id txID, err error) {
	if err != nil {
		l.Log.Warn("DA request failed", logFields(id, err)...)
	}
	l.state.TxFailed(id)
}

func (l *BatchSubmitter) recordFailedTx(id txID, err error) {
	l.Log.Warn("Transaction failed to send", logFields(id, err)...)
	l.state.TxFailed(id)
}

func (l *BatchSubmitter) recordConfirmedTx(id txID, receipt *types.Receipt) {
	l.Log.Info("Transaction confirmed", logFields(id, receipt)...)
	l1block := eth.ReceiptBlockID(receipt)
	l.state.TxConfirmed(id, l1block)
}

// l1Tip gets the current L1 tip as a L1BlockRef. The passed context is assumed
// to be a lifetime context, so it is internally wrapped with a network timeout.
func (l *BatchSubmitter) l1Tip(ctx context.Context) (eth.L1BlockRef, error) {
	tctx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()
	head, err := l.L1Client.HeaderByNumber(tctx, nil)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("getting latest L1 block: %w", err)
	}
	return eth.InfoToL1BlockRef(eth.HeaderBlockInfo(head)), nil
}

func (l *BatchSubmitter) checkTxpool(queue *txmgr.Queue[txRef], receiptsCh chan txmgr.TxReceipt[txRef]) bool {
	l.txpoolMutex.Lock()
	if l.txpoolState == TxpoolBlocked {
		// txpoolState is set to Blocked only if Send() is returning
		// ErrAlreadyReserved. In this case, the TxMgr nonce should be reset to nil,
		// allowing us to send a cancellation transaction.
		l.txpoolState = TxpoolCancelPending
		isBlob := l.txpoolBlockedBlob
		l.txpoolMutex.Unlock()
		l.cancelBlockingTx(queue, receiptsCh, isBlob)
		return false
	}
	r := l.txpoolState == TxpoolGood
	l.txpoolMutex.Unlock()
	return r
}

func logFields(xs ...any) (fs []any) {
	for _, x := range xs {
		switch v := x.(type) {
		case txID:
			fs = append(fs, "tx_id", v.String())
		case *types.Receipt:
			fs = append(fs, "tx", v.TxHash, "block", eth.ReceiptBlockID(v))
		case error:
			fs = append(fs, "err", v)
		default:
			fs = append(fs, "ERROR", fmt.Sprintf("logFields: unknown type: %T", x))
		}
	}
	return fs
}
