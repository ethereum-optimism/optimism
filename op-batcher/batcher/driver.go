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
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// BatchSubmitter encapsulates a service responsible for submitting L2 tx
// batches to L1 for availability.
type BatchSubmitter struct {
	Config // directly embed the config + sources

	txMgr txmgr.TxManager
	wg    sync.WaitGroup

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

// NewBatchSubmitterFromCLIConfig initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitterFromCLIConfig(cfg CLIConfig, l log.Logger, m metrics.Metricer) (*BatchSubmitter, error) {
	ctx := context.Background()

	// Connect to L1 and L2 providers. Perform these last since they are the
	// most expensive.
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	l2Client, err := dialEthClientWithTimeout(ctx, cfg.L2EthRpc)
	if err != nil {
		return nil, err
	}

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	rcfg, err := rollupClient.RollupConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying rollup config: %w", err)
	}

	txManager, err := txmgr.NewSimpleTxManager("batcher", l, m, cfg.TxMgrConfig)
	if err != nil {
		return nil, err
	}

	batcherCfg := Config{
		L1Client:               l1Client,
		L2Client:               l2Client,
		RollupNode:             rollupClient,
		PollInterval:           cfg.PollInterval,
		MaxPendingTransactions: cfg.MaxPendingTransactions,
		NetworkTimeout:         cfg.TxMgrConfig.NetworkTimeout,
		TxManager:              txManager,
		Rollup:                 rcfg,
		Channel: ChannelConfig{
			SeqWindowSize:      rcfg.SeqWindowSize,
			ChannelTimeout:     rcfg.ChannelTimeout,
			MaxChannelDuration: cfg.MaxChannelDuration,
			SubSafetyMargin:    cfg.SubSafetyMargin,
			MaxFrameSize:       cfg.MaxL1TxSize - 1,    // subtract 1 byte for version
			TargetFrameSize:    cfg.TargetL1TxSize - 1, // subtract 1 byte for version
			TargetNumFrames:    cfg.TargetNumFrames,
			ApproxComprRatio:   cfg.ApproxComprRatio,
		},
	}

	// Validate the batcher config
	if err := batcherCfg.Check(); err != nil {
		return nil, err
	}

	return NewBatchSubmitter(ctx, batcherCfg, l, m)
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitter(ctx context.Context, cfg Config, l log.Logger, m metrics.Metricer) (*BatchSubmitter, error) {
	balance, err := cfg.L1Client.BalanceAt(ctx, cfg.TxManager.From(), nil)
	if err != nil {
		return nil, err
	}

	cfg.log = l
	cfg.log.Info("creating batch submitter", "submitter_addr", cfg.TxManager.From(), "submitter_bal", balance)

	cfg.metr = m

	return &BatchSubmitter{
		Config: cfg,
		txMgr:  cfg.TxManager,
		state:  NewChannelManager(l, m, cfg.Channel),
	}, nil

}

func (l *BatchSubmitter) Start() error {
	l.log.Info("Starting Batch Submitter")

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.running {
		return errors.New("batcher is already running")
	}
	l.running = true

	l.shutdownCtx, l.cancelShutdownCtx = context.WithCancel(context.Background())
	l.killCtx, l.cancelKillCtx = context.WithCancel(context.Background())
	l.state.Clear()
	l.lastStoredBlock = eth.BlockID{}

	l.wg.Add(1)
	go l.loop()

	l.log.Info("Batch Submitter started")

	return nil
}

func (l *BatchSubmitter) StopIfRunning(ctx context.Context) {
	_ = l.Stop(ctx)
}

func (l *BatchSubmitter) Stop(ctx context.Context) error {
	l.log.Info("Stopping Batch Submitter")

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.running {
		return errors.New("batcher is not running")
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

	l.log.Info("Batch Submitter stopped")

	return nil
}

// loadBlocksIntoState loads all blocks since the previous stored block
// It does the following:
// 1. Fetch the sync status of the sequencer
// 2. Check if the sync status is valid or if we are all the way up to date
// 3. Check if it needs to initialize state OR it is lagging (todo: lagging just means race condition?)
// 4. Load all new blocks into the local state.
func (l *BatchSubmitter) loadBlocksIntoState(ctx context.Context) {
	start, end, err := l.calculateL2BlockRangeToStore(ctx)
	if err != nil {
		l.log.Warn("Error calculating L2 block range", "err", err)
		return
	} else if start.Number >= end.Number {
		return
	}

	var latestBlock *types.Block
	// Add all blocks to "state"
	for i := start.Number + 1; i < end.Number+1; i++ {
		block, err := l.loadBlockIntoState(ctx, i)
		if errors.Is(err, ErrReorg) {
			l.log.Warn("Found L2 reorg", "block_number", i)
			l.state.Clear()
			l.lastStoredBlock = eth.BlockID{}
			return
		} else if err != nil {
			l.log.Warn("failed to load block into state", "err", err)
			return
		}
		l.lastStoredBlock = eth.ToBlockID(block)
		latestBlock = block
	}

	l2ref, err := derive.L2BlockToBlockRef(latestBlock, &l.Rollup.Genesis)
	if err != nil {
		l.log.Warn("Invalid L2 block loaded into state", "err", err)
		return
	}

	l.metr.RecordL2BlocksLoaded(l2ref)
}

// loadBlockIntoState fetches & stores a single block into `state`. It returns the block it loaded.
func (l *BatchSubmitter) loadBlockIntoState(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, l.NetworkTimeout)
	defer cancel()
	block, err := l.L2Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("getting L2 block: %w", err)
	}

	if err := l.state.AddL2Block(block); err != nil {
		return nil, fmt.Errorf("adding L2 block to state: %w", err)
	}

	l.log.Info("added L2 block to local state", "block", eth.ToBlockID(block), "tx_count", len(block.Transactions()), "time", block.Time())
	return block, nil
}

// calculateL2BlockRangeToStore determines the range (start,end] that should be loaded into the local state.
// It also takes care of initializing some local state (i.e. will modify l.lastStoredBlock in certain conditions)
func (l *BatchSubmitter) calculateL2BlockRangeToStore(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	ctx, cancel := context.WithTimeout(ctx, l.NetworkTimeout)
	defer cancel()
	syncStatus, err := l.RollupNode.SyncStatus(ctx)
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
		l.log.Info("Starting batch-submitter work at safe-head", "safe", syncStatus.SafeL2)
		l.lastStoredBlock = syncStatus.SafeL2.ID()
	} else if l.lastStoredBlock.Number < syncStatus.SafeL2.Number {
		l.log.Warn("last submitted block lagged behind L2 safe head: batch submission will continue from the safe head now", "last", l.lastStoredBlock, "safe", syncStatus.SafeL2)
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

	ticker := time.NewTicker(l.PollInterval)
	defer ticker.Stop()

	receiptsCh := make(chan txmgr.TxReceipt[txData])
	queue := txmgr.NewQueue[txData](l.killCtx, l.txMgr, l.MaxPendingTransactions)

	for {
		select {
		case <-ticker.C:
			l.loadBlocksIntoState(l.shutdownCtx)
			l.publishStateToL1(queue, receiptsCh, false)
		case r := <-receiptsCh:
			l.handleReceipt(r)
		case <-l.shutdownCtx.Done():
			err := l.state.Close()
			if err != nil {
				l.log.Error("error closing the channel manager", "err", err)
			}
			l.publishStateToL1(queue, receiptsCh, true)
			return
		}
	}
}

// publishStateToL1 loops through the block data loaded into `state` and
// submits the associated data to the L1 in the form of channel frames.
func (l *BatchSubmitter) publishStateToL1(queue *txmgr.Queue[txData], receiptsCh chan txmgr.TxReceipt[txData], drain bool) {
	txDone := make(chan struct{})
	// send/wait and receipt reading must be on a separate goroutines to avoid deadlocks
	go func() {
		defer func() {
			if drain {
				// if draining, we wait for all transactions to complete
				queue.Wait()
			}
			close(txDone)
		}()
		for {
			err := l.publishTxToL1(l.killCtx, queue, receiptsCh)
			if err != nil {
				if drain && err != io.EOF {
					l.log.Error("error sending tx while draining state", "err", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case r := <-receiptsCh:
			l.handleReceipt(r)
		case <-txDone:
			return
		}
	}
}

// publishTxToL1 submits a single state tx to the L1
func (l *BatchSubmitter) publishTxToL1(ctx context.Context, queue *txmgr.Queue[txData], receiptsCh chan txmgr.TxReceipt[txData]) error {
	// send all available transactions
	l1tip, err := l.l1Tip(ctx)
	if err != nil {
		l.log.Error("Failed to query L1 tip", "error", err)
		return err
	}
	l.recordL1Tip(l1tip)

	// Collect next transaction data
	txdata, err := l.state.TxData(l1tip.ID())
	if err == io.EOF {
		l.log.Trace("no transaction data available")
		return err
	} else if err != nil {
		l.log.Error("unable to get tx data", "err", err)
		return err
	}

	l.sendTransaction(txdata, queue, receiptsCh)
	return nil
}

// sendTransaction creates & submits a transaction to the batch inbox address with the given `data`.
// It currently uses the underlying `txmgr` to handle transaction sending & price management.
// This is a blocking method. It should not be called concurrently.
func (l *BatchSubmitter) sendTransaction(txdata txData, queue *txmgr.Queue[txData], receiptsCh chan txmgr.TxReceipt[txData]) {
	// Do the gas estimation offline. A value of 0 will cause the [txmgr] to estimate the gas limit.
	data := txdata.Bytes()
	intrinsicGas, err := core.IntrinsicGas(data, nil, false, true, true, false)
	if err != nil {
		l.log.Error("Failed to calculate intrinsic gas", "error", err)
		return
	}

	candidate := txmgr.TxCandidate{
		To:       &l.Rollup.BatchInboxAddress,
		TxData:   data,
		GasLimit: intrinsicGas,
	}
	queue.Send(txdata, candidate, receiptsCh)
}

func (l *BatchSubmitter) handleReceipt(r txmgr.TxReceipt[txData]) {
	// Record TX Status
	if r.Err != nil {
		l.log.Warn("unable to publish tx", "err", r.Err, "data_size", r.ID.Len())
		l.recordFailedTx(r.ID.ID(), r.Err)
	} else {
		l.log.Info("tx successfully published", "tx_hash", r.Receipt.TxHash, "data_size", r.ID.Len())
		l.recordConfirmedTx(r.ID.ID(), r.Receipt)
	}
}

func (l *BatchSubmitter) recordL1Tip(l1tip eth.L1BlockRef) {
	if l.lastL1Tip == l1tip {
		return
	}
	l.lastL1Tip = l1tip
	l.metr.RecordLatestL1Block(l1tip)
}

func (l *BatchSubmitter) recordFailedTx(id txID, err error) {
	l.log.Warn("Failed to send transaction", "err", err)
	l.state.TxFailed(id)
}

func (l *BatchSubmitter) recordConfirmedTx(id txID, receipt *types.Receipt) {
	l.log.Info("Transaction confirmed", "tx_hash", receipt.TxHash, "status", receipt.Status, "block_hash", receipt.BlockHash, "block_number", receipt.BlockNumber)
	l1block := eth.BlockID{Number: receipt.BlockNumber.Uint64(), Hash: receipt.BlockHash}
	l.state.TxConfirmed(id, l1block)
}

// l1Tip gets the current L1 tip as a L1BlockRef. The passed context is assumed
// to be a lifetime context, so it is internally wrapped with a network timeout.
func (l *BatchSubmitter) l1Tip(ctx context.Context) (eth.L1BlockRef, error) {
	tctx, cancel := context.WithTimeout(ctx, l.NetworkTimeout)
	defer cancel()
	head, err := l.L1Client.HeaderByNumber(tctx, nil)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("getting latest L1 block: %w", err)
	}
	return eth.InfoToL1BlockRef(eth.HeaderBlockInfo(head)), nil
}
