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
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
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
	done  chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

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

	signer, fromAddress, err := opcrypto.SignerFactoryFromConfig(l, cfg.PrivateKey, cfg.Mnemonic, cfg.SequencerHDPath, cfg.SignerConfig)
	if err != nil {
		return nil, err
	}

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

	txManagerConfig := txmgr.Config{
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
		From:                      fromAddress,
		ChainID:                   rcfg.L1ChainID,
		Signer:                    signer(rcfg.L1ChainID),
	}

	batcherCfg := Config{
		L1Client:        l1Client,
		L2Client:        l2Client,
		RollupNode:      rollupClient,
		PollInterval:    cfg.PollInterval,
		TxManagerConfig: txManagerConfig,
		From:            fromAddress,
		Rollup:          rcfg,
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
	balance, err := cfg.L1Client.BalanceAt(ctx, cfg.From, nil)
	if err != nil {
		return nil, err
	}

	cfg.log = l
	cfg.log.Info("creating batch submitter", "submitter_addr", cfg.From, "submitter_bal", balance)

	cfg.metr = m

	return &BatchSubmitter{
		Config: cfg,
		txMgr:  txmgr.NewSimpleTxManager("batcher", l, cfg.TxManagerConfig, cfg.L1Client),
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

	l.done = make(chan struct{})
	// TODO: this context only exists because the event loop doesn't reach done
	// if the tx manager is blocking forever due to e.g. insufficient balance.
	l.ctx, l.cancel = context.WithCancel(context.Background())
	l.state.Clear()
	l.lastStoredBlock = eth.BlockID{}

	l.wg.Add(1)
	go l.loop()

	l.log.Info("Batch Submitter started")

	return nil
}

func (l *BatchSubmitter) StopIfRunning() {
	_ = l.Stop()
}

func (l *BatchSubmitter) Stop() error {
	l.log.Info("Stopping Batch Submitter")

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.running {
		return errors.New("batcher is not running")
	}
	l.running = false

	l.cancel()
	close(l.done)
	l.wg.Wait()

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
	} else if start.Number == end.Number {
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
	ctx, cancel := context.WithTimeout(ctx, txManagerTimeout)
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
	childCtx, cancel := context.WithTimeout(ctx, txManagerTimeout)
	defer cancel()
	syncStatus, err := l.RollupNode.SyncStatus(childCtx)
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
	for {
		select {
		case <-ticker.C:
			l.loadBlocksIntoState(l.ctx)

		blockLoop:
			for {
				l1tip, err := l.l1Tip(l.ctx)
				if err != nil {
					l.log.Error("Failed to query L1 tip", "error", err)
					break
				}
				l.recordL1Tip(l1tip)

				// Collect next transaction data
				txdata, err := l.state.TxData(l1tip.ID())
				if err == io.EOF {
					l.log.Trace("no transaction data available")
					break // local for loop
				} else if err != nil {
					l.log.Error("unable to get tx data", "err", err)
					break
				}

				// Record TX Status
				if receipt, err := l.SendTransaction(l.ctx, txdata.Bytes()); err != nil {
					l.recordFailedTx(txdata.ID(), err)
				} else {
					l.recordConfirmedTx(txdata.ID(), receipt)
				}

				// hack to exit this loop. Proper fix is to do request another send tx or parallel tx sending
				// from the channel manager rather than sending the channel in a loop. This stalls b/c if the
				// context is cancelled while sending, it will never fully clear the pending txns.
				select {
				case <-l.ctx.Done():
					break blockLoop
				default:
				}
			}

		case <-l.done:
			return
		}
	}
}

const networkTimeout = 2 * time.Second // How long a single network request can take. TODO: put in a config somewhere

// fix(refcell):
// combined with above, these config variables should also be replicated in the op-proposer
// along with op-proposer changes to include the updated tx manager
const txManagerTimeout = 2 * time.Minute // How long the tx manager can take to send a transaction.

// SendTransaction creates & submits a transaction to the batch inbox address with the given `data`.
// It currently uses the underlying `txmgr` to handle transaction sending & price management.
// This is a blocking method. It should not be called concurrently.
func (l *BatchSubmitter) SendTransaction(ctx context.Context, data []byte) (*types.Receipt, error) {
	// Do the gas estimation offline. A value of 0 will cause the [txmgr] to estimate the gas limit.
	intrinsicGas, err := core.IntrinsicGas(data, nil, false, true, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate intrinsic gas: %w", err)
	}

	// Create the transaction
	tx, err := l.txMgr.CraftTx(ctx, txmgr.TxCandidate{
		To:       l.Rollup.BatchInboxAddress,
		TxData:   data,
		From:     l.From,
		GasLimit: intrinsicGas,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tx: %w", err)
	}

	// Send the transaction through the txmgr
	ctx, cancel := context.WithTimeout(ctx, txManagerTimeout)
	defer cancel()
	if receipt, err := l.txMgr.Send(ctx, tx); err != nil {
		l.log.Warn("unable to publish tx", "err", err, "data_size", len(data))
		return nil, err
	} else {
		l.log.Info("tx successfully published", "tx_hash", receipt.TxHash, "data_size", len(data))
		return receipt, nil
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
	tctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	head, err := l.L1Client.HeaderByNumber(tctx, nil)
	if err != nil {
		return eth.L1BlockRef{}, fmt.Errorf("getting latest L1 block: %w", err)
	}
	return eth.InfoToL1BlockRef(eth.HeaderBlockInfo(head)), nil
}
