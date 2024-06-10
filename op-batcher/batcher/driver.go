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

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrBatcherNotRunning = errors.New("batcher is not running")

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
	ChannelConfig    ChannelConfig
	PlasmaDA         *plasma.DAClient
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

	l.wg.Add(1)
	go l.loop()

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
// 1. Fetch the sync status of the sequencer
// 2. Check if the sync status is valid or if we are all the way up to date
// 3. Check if it needs to initialize state OR it is lagging (todo: lagging just means race condition?)
// 4. Load all new blocks into the local state.
// If there is a reorg, it will reset the last stored block but not clear the internal state so
// the state can be flushed to L1.
func (l *BatchSubmitter) loadBlocksIntoState(ctx context.Context) error {
	start, end, err := l.calculateL2BlockRangeToStore(ctx)
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
			l.Log.Warn("failed to load block into state", "err", err)
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

	l.Log.Info("added L2 block to local state", "block", eth.ToBlockID(block), "tx_count", len(block.Transactions()), "time", block.Time())
	return block, nil
}

// calculateL2BlockRangeToStore determines the range (start,end] that should be loaded into the local state.
// It also takes care of initializing some local state (i.e. will modify l.lastStoredBlock in certain conditions)
func (l *BatchSubmitter) calculateL2BlockRangeToStore(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	rollupClient, err := l.EndpointProvider.RollupClient(ctx)
	if err != nil {
		return eth.BlockID{}, eth.BlockID{}, fmt.Errorf("getting rollup client: %w", err)
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Config.NetworkTimeout)
	defer cancel()

	syncStatus, err := rollupClient.SyncStatus(cCtx)
	// Ensure that we have the sync status
	if err != nil {
		return eth.BlockID{}, eth.BlockID{}, fmt.Errorf("failed to get sync status: %w", err)
	}
	if syncStatus.HeadL1 == (eth.L1BlockRef{}) {
		return eth.BlockID{}, eth.BlockID{}, errors.New("empty sync status")
	}

	// Check last stored to see if it needs to be set on startup OR set if is lagged behind.
	// It lagging implies that the op-node processed some batches that were submitted prior to the current instance of the batcher being alive.
	if l.lastStoredBlock == (eth.BlockID{}) {
		l.Log.Info("Starting batch-submitter work at safe-head", "safe", syncStatus.SafeL2)
		l.lastStoredBlock = syncStatus.SafeL2.ID()
	} else if l.lastStoredBlock.Number < syncStatus.SafeL2.Number {
		l.Log.Warn("last submitted block lagged behind L2 safe head: batch submission will continue from the safe head now", "last", l.lastStoredBlock, "safe", syncStatus.SafeL2)
		l.lastStoredBlock = syncStatus.SafeL2.ID()
	}

	// Check if we should even attempt to load any blocks. TODO: May not need this check
	if syncStatus.SafeL2.Number >= syncStatus.UnsafeL2.Number {
		return eth.BlockID{}, eth.BlockID{}, errors.New("L2 safe head ahead of L2 unsafe head")
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

func (l *BatchSubmitter) loop() {
	defer l.wg.Done()
	if l.Config.WaitNodeSync {
		err := l.waitNodeSync()
		if err != nil {
			l.Log.Error("Error waiting for node sync", "err", err)
			return
		}
	}

	receiptsCh := make(chan txmgr.TxReceipt[txID])
	queue := txmgr.NewQueue[txID](l.killCtx, l.Txmgr, l.Config.MaxPendingTransactions)

	// start the receipt/result processing loop
	receiptLoopDone := make(chan struct{})
	defer close(receiptLoopDone) // shut down receipt loop
	go func() {
		for {
			select {
			case r := <-receiptsCh:
				l.Log.Info("handling receipt", "id", r.ID)
				l.handleReceipt(r)
			case <-receiptLoopDone:
				l.Log.Info("receipt processing loop done")
				return
			}
		}
	}()

	ticker := time.NewTicker(l.Config.PollInterval)
	defer ticker.Stop()

	publishAndWait := func() {
		l.publishStateToL1(queue, receiptsCh)
		if !l.Txmgr.IsClosed() {
			queue.Wait()
		} else {
			l.Log.Info("Txmgr is closed, remaining channel data won't be sent")
		}
	}

	for {
		select {
		case <-ticker.C:
			if err := l.loadBlocksIntoState(l.shutdownCtx); errors.Is(err, ErrReorg) {
				err := l.state.Close()
				if err != nil {
					if errors.Is(err, ErrPendingAfterClose) {
						l.Log.Warn("Closed channel manager to handle L2 reorg with pending channel(s) remaining - submitting")
					} else {
						l.Log.Error("Error closing the channel manager to handle a L2 reorg", "err", err)
					}
				}
				// on reorg we want to publish all pending state then wait until each result clears before resetting
				// the state.
				publishAndWait()
				l.clearState(l.shutdownCtx)
				continue
			}
			l.publishStateToL1(queue, receiptsCh)
		case <-l.shutdownCtx.Done():
			if l.Txmgr.IsClosed() {
				l.Log.Info("Txmgr is closed, remaining channel data won't be sent")
				return
			}
			// This removes any never-submitted pending channels, so these do not have to be drained with transactions.
			// Any remaining unfinished channel is terminated, so its data gets submitted.
			err := l.state.Close()
			if err != nil {
				if errors.Is(err, ErrPendingAfterClose) {
					l.Log.Warn("Closed channel manager on shutdown with pending channel(s) remaining - submitting")
				} else {
					l.Log.Error("Error closing the channel manager on shutdown", "err", err)
				}
			}
			publishAndWait()
			l.Log.Info("Finished publishing all remaining channel data")
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
func (l *BatchSubmitter) publishStateToL1(queue *txmgr.Queue[txID], receiptsCh chan txmgr.TxReceipt[txID]) {
	for {
		// if the txmgr is closed, we stop the transaction sending
		if l.Txmgr.IsClosed() {
			l.Log.Info("Txmgr is closed, aborting state publishing")
			return
		}
		err := l.publishTxToL1(l.killCtx, queue, receiptsCh)
		if err != nil {
			if err != io.EOF {
				l.Log.Error("error publishing tx to l1", "err", err)
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
func (l *BatchSubmitter) publishTxToL1(ctx context.Context, queue *txmgr.Queue[txID], receiptsCh chan txmgr.TxReceipt[txID]) error {
	// send all available transactions
	l1tip, err := l.l1Tip(ctx)
	if err != nil {
		l.Log.Error("Failed to query L1 tip", "err", err)
		return err
	}
	l.recordL1Tip(l1tip)

	// Collect next transaction data
	txdata, err := l.state.TxData(l1tip.ID())

	if err == io.EOF {
		l.Log.Trace("no transaction data available")
		return err
	} else if err != nil {
		l.Log.Error("unable to get tx data", "err", err)
		return err
	}

	if err = l.sendTransaction(ctx, txdata, queue, receiptsCh); err != nil {
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

// sendTransaction creates & queues for sending a transaction to the batch inbox address with the given `txData`.
// The method will block if the queue's MaxPendingTransactions is exceeded.
func (l *BatchSubmitter) sendTransaction(ctx context.Context, txdata txData, queue *txmgr.Queue[txID], receiptsCh chan txmgr.TxReceipt[txID]) error {
	var err error
	// Do the gas estimation offline. A value of 0 will cause the [txmgr] to estimate the gas limit.

	var candidate *txmgr.TxCandidate
	if l.Config.UseBlobs {
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
			l.Log.Crit("unexpected number of frames in calldata tx", "num_frames", nf)
		}
		data := txdata.CallData()
		// if plasma DA is enabled we post the txdata to the DA Provider and replace it with the commitment.
		if l.Config.UsePlasma {
			comm, err := l.PlasmaDA.SetInput(ctx, data)
			if err != nil {
				l.Log.Error("Failed to post input to Plasma DA", "error", err)
				// requeue frame if we fail to post to the DA Provider so it can be retried
				l.recordFailedTx(txdata.ID(), err)
				return nil
			}
			// signal plasma commitment tx with TxDataVersion1
			data = comm.TxData()
		}
		candidate = l.calldataTxCandidate(data)
	}

	intrinsicGas, err := core.IntrinsicGas(candidate.TxData, nil, false, true, true, false)
	if err != nil {
		// we log instead of return an error here because txmgr can do its own gas estimation
		l.Log.Error("Failed to calculate intrinsic gas", "err", err)
	} else {
		candidate.GasLimit = intrinsicGas
	}

	queue.Send(txdata.ID(), *candidate, receiptsCh)
	return nil
}

func (l *BatchSubmitter) blobTxCandidate(data txData) (*txmgr.TxCandidate, error) {
	blobs, err := data.Blobs()
	if err != nil {
		return nil, fmt.Errorf("generating blobs for tx data: %w", err)
	}
	size := data.Len()
	lastSize := len(data.frames[len(data.frames)-1].data)
	l.Log.Info("building Blob transaction candidate",
		"size", size, "last_size", lastSize, "num_blobs", len(blobs))
	l.Metr.RecordBlobUsedBytes(lastSize)
	return &txmgr.TxCandidate{
		To:    &l.RollupConfig.BatchInboxAddress,
		Blobs: blobs,
	}, nil
}

func (l *BatchSubmitter) calldataTxCandidate(data []byte) *txmgr.TxCandidate {
	l.Log.Info("building Calldata transaction candidate", "size", len(data))
	return &txmgr.TxCandidate{
		To:     &l.RollupConfig.BatchInboxAddress,
		TxData: data,
	}
}

func (l *BatchSubmitter) handleReceipt(r txmgr.TxReceipt[txID]) {
	// Record TX Status
	if r.Err != nil {
		l.recordFailedTx(r.ID, r.Err)
	} else {
		l.recordConfirmedTx(r.ID, r.Receipt)
	}
}

func (l *BatchSubmitter) recordL1Tip(l1tip eth.L1BlockRef) {
	if l.lastL1Tip == l1tip {
		return
	}
	l.lastL1Tip = l1tip
	l.Metr.RecordLatestL1Block(l1tip)
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
