package monitor

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrBlockNotFound = errors.New("block not found")

func (r *BlockBuffer) Pop() (eth.BlockInfo, error) {
	// if the buffer is empty, return an error
	if r.total == 0 {
		return nil, ErrBlockNotFound
	}
	// get the previous index, wrap around if necessary
	prevIndex := (r.idx + len(r.buffer) - 1) % len(r.buffer)
	block := r.buffer[prevIndex]
	// if the block is nil, the buffer is empty
	if block == nil {
		return nil, ErrBlockNotFound
	}
	// decrement and wrap the index around the buffer
	r.idx = prevIndex
	r.total--
	return block, nil
}

// JobFilter is a function that turns any executing messages from a slice of receipts
// into a slice of jobs which can be added to the Maintainer's inbox
type JobFilter func(receipts []*types.Receipt) []*Job

// FinderClient is a client that can be used to find new blocks and their receipts
// it is satisfied by the ethclient.Client type
type FinderClient interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	FetchReceiptsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Receipts, error)
}

var _ FinderClient = &sources.EthClient{}

// Finders are responsible for finding new jobs from a chain for the Maintainer to track
type Finder interface {
	Start(ctx context.Context) error
	Stop() error
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCFinder struct {
	client  FinderClient
	chainID eth.ChainID

	pollInterval       time.Duration
	expiryPollInterval time.Duration

	inbox    chan *types.Header
	toJobs   JobFilter
	callback func(*Job)
	closed   chan struct{}
	log      log.Logger

	next       uint64
	seenBlocks *BlockBuffer
}

func NewFinder(chainID eth.ChainID, client FinderClient, toCases JobFilter, callback func(*Job), log log.Logger) *RPCFinder {
	return &RPCFinder{
		chainID:            chainID,
		client:             client,
		log:                log.New("component", "rpc_finder", "chain_id", chainID),
		toJobs:             toCases,
		inbox:              make(chan *types.Header, 1000),
		closed:             make(chan struct{}),
		callback:           callback,
		pollInterval:       2 * time.Second,
		expiryPollInterval: 10 * time.Second,
		seenBlocks:         NewBlockBuffer(1000),
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

	for {
		select {
		case <-t.closed:
			t.log.Info("finder closed")
			close(t.inbox)
			return
		case <-fetchTicker.C:
			blockInfo, receipts, err := t.client.FetchReceiptsByNumber(ctx, t.next)
			if errors.Is(err, ethereum.NotFound) {
				t.log.Debug("block not found", "block", t.next)
				// once a block is not found, increase the poll interval to the configured value
				fetchTicker.Reset(t.pollInterval)
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
		}
	}
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
	jobs := t.toJobs([]*types.Receipt(receipts))
	firstSeen := time.Now()
	for _, job := range jobs {
		job.firstSeen = firstSeen
		job.UpdateStatus(jobStatusUnknown)
		t.callback(job)
	}
	if len(jobs) > 0 {
		t.log.Info("added jobs to callback", "count", len(jobs))
	}
	t.log.Debug("visited block", "block", blockInfo.NumberU64())
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
		previous, err := t.seenBlocks.Pop()
		if err != nil {
			t.log.Error("error popping block", "error", err)
			return err
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
