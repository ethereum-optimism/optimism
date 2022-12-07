package op_batcher

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/ethereum-optimism/optimism/op-batcher/sequencer"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// BatchSubmitter encapsulates a service responsible for submitting L2 tx
// batches to L1 for availability.
type BatchSubmitter struct {
	txMgr *TransactionManager
	addr  common.Address
	cfg   sequencer.Config
	wg    sync.WaitGroup
	done  chan struct{}
	log   log.Logger

	ctx    context.Context
	cancel context.CancelFunc

	// lastStoredBlock is the last block loaded into `state`. If it is empty it should be set to the l2 safe head.
	lastStoredBlock eth.BlockID

	state *channelManager
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed during operation.
func NewBatchSubmitter(cfg Config, l log.Logger) (*BatchSubmitter, error) {
	ctx := context.Background()

	var err error
	var sequencerPrivKey *ecdsa.PrivateKey
	var addr common.Address

	if cfg.PrivateKey != "" && cfg.Mnemonic != "" {
		return nil, errors.New("cannot specify both a private key and a mnemonic")
	}

	if cfg.PrivateKey == "" {
		// Parse wallet private key that will be used to submit L2 txs to the batch
		// inbox address.
		wallet, err := hdwallet.NewFromMnemonic(cfg.Mnemonic)
		if err != nil {
			return nil, err
		}

		acc := accounts.Account{
			URL: accounts.URL{
				Path: cfg.SequencerHDPath,
			},
		}
		addr, err = wallet.Address(acc)
		if err != nil {
			return nil, err
		}

		sequencerPrivKey, err = wallet.PrivateKey(acc)
		if err != nil {
			return nil, err
		}
	} else {
		sequencerPrivKey, err = crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
		if err != nil {
			return nil, err
		}

		addr = crypto.PubkeyToAddress(sequencerPrivKey.PublicKey)
	}

	batchInboxAddress, err := parseAddress(cfg.SequencerBatchInboxAddress)
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

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	sequencerBalance, err := l1Client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, err
	}

	log.Info("starting batch submitter", "submitter_addr", addr, "submitter_bal", sequencerBalance)

	txManagerConfig := txmgr.Config{
		Log:                       l,
		Name:                      "Batch Submitter",
		ResubmissionTimeout:       cfg.ResubmissionTimeout,
		ReceiptQueryInterval:      time.Second,
		NumConfirmations:          cfg.NumConfirmations,
		SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
	}

	batcherCfg := sequencer.Config{
		Log:               l,
		Name:              "Batch Submitter",
		L1Client:          l1Client,
		L2Client:          l2Client,
		RollupNode:        rollupClient,
		MinL1TxSize:       cfg.MinL1TxSize,
		MaxL1TxSize:       cfg.MaxL1TxSize,
		BatchInboxAddress: batchInboxAddress,
		ChannelTimeout:    cfg.ChannelTimeout,
		ChainID:           chainID,
		PrivKey:           sequencerPrivKey,
		PollInterval:      cfg.PollInterval,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BatchSubmitter{
		cfg:   batcherCfg,
		addr:  addr,
		txMgr: NewTransactionManager(l, txManagerConfig, batchInboxAddress, chainID, sequencerPrivKey, l1Client),
		done:  make(chan struct{}),
		log:   l,
		state: NewChannelManager(l, cfg.ChannelTimeout),
		// TODO: this context only exists because the even loop doesn't reach done
		// if the tx manager is blocking forever due to e.g. insufficient balance.
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (l *BatchSubmitter) Start() error {
	l.wg.Add(1)
	go l.loop()
	return nil
}

func (l *BatchSubmitter) Stop() {
	l.cancel()
	close(l.done)
	l.wg.Wait()
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
		l.log.Trace("was not able to calculate L2 block range", "err", err)
		return
	}

	// Add all blocks to "state"
	for i := start.Number + 1; i < end.Number+1; i++ {
		id, err := l.loadBlockIntoState(ctx, i)
		if errors.Is(err, ErrReorg) {
			l.log.Warn("Found L2 reorg", "block_number", i)
			l.state.Clear()
			l.lastStoredBlock = eth.BlockID{}
			return
		} else if err != nil {
			l.log.Warn("failed to load block into state", "err", err)
			return
		}
		l.lastStoredBlock = id
	}
}

// loadBlockIntoState fetches & stores a single block into `state`. It returns the block it loaded.
func (l *BatchSubmitter) loadBlockIntoState(ctx context.Context, blockNumber uint64) (eth.BlockID, error) {
	ctx, cancel := context.WithTimeout(ctx, networkTimeout)
	block, err := l.cfg.L2Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	cancel()
	if err != nil {
		return eth.BlockID{}, err
	}
	if err := l.state.AddL2Block(block); err != nil {
		return eth.BlockID{}, err
	}
	id := eth.ToBlockID(block)
	l.log.Info("added L2 block to local state", "block", id, "tx_count", len(block.Transactions()), "time", block.Time())
	return id, nil
}

// calculateL2BlockRangeToStore determines the range (start,end] that should be loaded into the local state.
// It also takes care of initializing some local state (i.e. will modify l.lastStoredBlock in certain conditions)
func (l *BatchSubmitter) calculateL2BlockRangeToStore(ctx context.Context) (eth.BlockID, eth.BlockID, error) {
	childCtx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	syncStatus, err := l.cfg.RollupNode.SyncStatus(childCtx)
	// Ensure that we have the sync status
	if err != nil {
		return eth.BlockID{}, eth.BlockID{}, fmt.Errorf("failed to get sync status: %w", err)
	}
	if syncStatus.HeadL1 == (eth.L1BlockRef{}) {
		return eth.BlockID{}, eth.BlockID{}, errors.New("empty sync status")
	}

	// Check last stored to see if it needs to be set on startup OR set if is lagged behind.
	// It lagging implies that the op-node processed some batches that where submitted prior to the current instance of the batcher being alive.
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

	ticker := time.NewTicker(l.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.loadBlocksIntoState(l.ctx)

			// Empty the state after loading into it on every iteration.
		blockLoop:
			for {
				// Collect the output frame
				data, id, err := l.state.TxData(eth.L1BlockRef{})
				if err == io.EOF {
					l.log.Trace("no transaction data available")
					break // local for loop
				} else if err != nil {
					l.log.Error("unable to get tx data", "err", err)
					break
				}
				// Record TX Status
				if receipt, err := l.txMgr.SendTransaction(l.ctx, data); err != nil {
					l.log.Error("Failed to send transaction", "err", err)
					l.state.TxFailed(id)
				} else {
					l.log.Info("Transaction confirmed", "tx_hash", receipt.TxHash, "status", receipt.Status, "block_hash", receipt.BlockHash, "block_number", receipt.BlockNumber)
					l.state.TxConfirmed(id, eth.BlockID{Number: receipt.BlockNumber.Uint64(), Hash: receipt.BlockHash})
				}

				// hack to exit this loop. Proper fix is to do request another send tx or parallel tx sending
				// from the channel manager rather than sending the channel in a loop. This stalls b/c if the
				// context is cancelled while sending, it will never fuilly clearing the pending txns.
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
