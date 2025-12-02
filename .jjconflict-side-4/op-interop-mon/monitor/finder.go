package monitor

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/buffer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrBlockNotFound = errors.New("block not found")

// JobFilter is a function that turns any executing messages from a slice of receipts
// into a slice of jobs which can be added to an Updater's inbox
type JobFilter func(receipts []*types.Receipt, executingChain eth.ChainID) []*Job

// NewCallback is a function to be called when a new job is created
type NewCallback func(*Job)

// FinalityCallback is a function to be called when the finality of jobs for this chain is updated
type FinalityCallback func(chainID eth.ChainID, block eth.BlockInfo)

// FinderClient is a client that can be used to find new blocks and their receipts
// it is satisfied by the ethclient.Client type
type FinderClient interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error)
}

var _ FinderClient = &sources.EthClient{}

// Finders are responsible for finding new jobs from a chain for an Updater to track
type Finder interface {
	Start(ctx context.Context) error
	Stop() error
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCFinder struct {
	chainID eth.ChainID
	client  FinderClient
	log     log.Logger

	finalityPollInterval time.Duration
	finalityCallback     FinalityCallback

	fetchInterval time.Duration
	next          uint64
	seenBlocks    *buffer.Ring[eth.BlockInfo]
	toJobs        JobFilter
	newCallback   NewCallback

	closed chan struct{}
}

func NewFinder(chainID eth.ChainID,
	client FinderClient,
	toCases JobFilter,
	newCallback NewCallback,
	finalityCallback FinalityCallback,
	bufferSize int,
	log log.Logger) *RPCFinder {
	return &RPCFinder{
		chainID:              chainID,
		client:               client,
		log:                  log.New("component", "rpc_finder", "chain_id", chainID),
		fetchInterval:        1 * time.Second,
		seenBlocks:           buffer.NewRing[eth.BlockInfo](1000),
		toJobs:               toCases,
		newCallback:          newCallback,
		finalityPollInterval: 10 * time.Second,
		finalityCallback:     finalityCallback,
		closed:               make(chan struct{}),
	}
}

func (t *RPCFinder) Start(ctx context.Context) error {
	// seed the seenBlocks buffer with the latest block
	block, err := t.client.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return err
	}
	// static backfill of 100 blocks. to be made configurable
	t.next = uint64(max(0, int64(block.NumberU64())-100))

	go t.Run(ctx)
	return nil
}

func (t *RPCFinder) Run(ctx context.Context) {
	// fetchTicker starts at 100ms to rapidly backfill blocks
	fetchTicker := time.NewTicker(100 * time.Millisecond)
	defer fetchTicker.Stop()
	// finalityTicker tracks finalized L2 blocks of this chain
	finalityTicker := time.NewTicker(t.finalityPollInterval)
	defer finalityTicker.Stop()

	for {
		select {
		case <-t.closed:
			t.log.Info("finder closed")
			return
		case <-fetchTicker.C:
			blockInfo, receipts, err := t.client.FetchReceiptsByNumber(ctx, t.next)
			if errors.Is(err, ethereum.NotFound) {
				t.log.Debug("block not found", "block", t.next)
				// once a block is not found, increase the poll interval to the configured value
				fetchTicker.Reset(t.fetchInterval)
				continue
			} else if err != nil {
				t.log.Error("error getting block", "error", err)
				continue
			}
			err = t.processBlock(blockInfo, receipts)
			if errors.Is(err, ErrBlockNotContiguous) {
				err := t.walkback(ctx)
				if err != nil {
					t.log.Error("error walking back", "error", err)
				}
				continue
			} else if err != nil {
				t.log.Error("error processing block", "error", err)
				continue
			}
		case <-finalityTicker.C:
			t.checkFinality(ctx)
		}
	}
}

// checkFinality checks the latest finalized block on the L2 chain
// and updates the finality callback
func (t *RPCFinder) checkFinality(ctx context.Context) {
	blockInfo, err := t.client.InfoByLabel(ctx, eth.Finalized)
	if err != nil {
		t.log.Error("error getting finalized block", "error", err)
		return
	}
	t.finalityCallback(t.chainID, blockInfo)
}

var ErrBlockNotContiguous = errors.New("blocks are not contiguous")

// processBlock processes a block and its receipts
// it checks if the block is contiguous with the previous block
// if it is:
// it then calls the toJobs function to convert the receipts to jobs
// it then calls the callback with the jobs
// it then adds the block to the seenBlocks buffer
// it returns a sentinel error if the block was not contiguous and
// a generic error any of the steps fail
func (t *RPCFinder) processBlock(blockInfo eth.BlockInfo, receipts types.Receipts) error {
	previous := t.seenBlocks.Peek()
	if previous != nil {
		// check if the blocks being processed are contiguous
		if blockInfo.ParentHash() != previous.Hash() ||
			blockInfo.NumberU64() != previous.NumberU64()+1 {
			t.log.Error("blocks are not contiguous", "previous", eth.InfoToL1BlockRef(previous), "next", eth.InfoToL1BlockRef(blockInfo))
			return ErrBlockNotContiguous
		}
	}
	jobs := t.toJobs([]*types.Receipt(receipts), t.chainID)
	firstSeen := time.Now()
	for _, job := range jobs {
		job.firstSeen = firstSeen
		job.UpdateStatus(jobStatusUnknown)
		t.newCallback(job)
	}
	t.log.Debug("block processed", "block", blockInfo.NumberU64(), "jobs", len(jobs))
	t.seenBlocks.Add(blockInfo)
	t.next++
	return nil
}

// walkback walks back to the last contiguous block which matches on the l2 client
// it will pop blocks from the buffer until it finds a block that matches the hash,
// or until an error occurs, including when the buffer is empty.
func (t *RPCFinder) walkback(ctx context.Context) error {
	for {
		// pop the last block from the buffer
		previous := t.seenBlocks.Pop()
		if previous == nil {
			t.log.Error("no blocks to walk back to")
			return ErrBlockNotFound
		}
		// fetch the block from the client
		block, err := t.client.InfoByNumber(ctx, previous.NumberU64())
		if err != nil {
			t.log.Error("error fetching block", "error", err)
			return err
		}
		if block.Hash() != previous.Hash() {
			t.log.Debug("block hash mismatch", "height", previous.NumberU64(), "expected", previous.Hash(), "got", block.Hash())
			continue
		}
		// if the block is contiguous, add it back to the buffer
		t.log.Info("walked back to common ancestor", "block", eth.InfoToL1BlockRef(block))
		t.seenBlocks.Add(block)
		t.next = block.NumberU64() + 1
		return nil
	}
}

func (t *RPCFinder) Stop() error {
	close(t.closed)
	return nil
}

func (t *RPCFinder) Stopped() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}
