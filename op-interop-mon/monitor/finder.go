package monitor

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// JobFilter is a function that turns any executing messages from a slice of receipts
// into a slice of jobs which can be added to the Maintainer's inbox
type JobFilter func(receipts []*types.Receipt) []*Job

// FinderClient is a client that can be used to find new blocks and their receipts
// it is satisfied by the ethclient.Client type
type FinderClient interface {
	BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

var _ FinderClient = &ethclient.Client{}

// Finders are responsible for finding new jobs from a chain for the Maintainer to track
type Finder interface {
	Start(ctx context.Context) error
	Stop() error
}

// RPCFinder connects to an Ethereum chain and extracts receipts in order to create jobs
type RPCFinder struct {
	client  FinderClient
	chainID eth.ChainID

	sub         ethereum.Subscription
	subErr      <-chan error
	inbox       chan *types.Header
	lastHandled eth.BlockID
	toJobs      JobFilter
	callback    func(*Job)
	closed      chan struct{}
	log         log.Logger
}

func NewFinder(chainID eth.ChainID, client FinderClient, toCases JobFilter, callback func(*Job), log log.Logger) *RPCFinder {
	return &RPCFinder{
		chainID:  chainID,
		client:   client,
		log:      log.New("component", "rpc_finder", "chain_id", chainID),
		toJobs:   toCases,
		inbox:    make(chan *types.Header, 1000),
		closed:   make(chan struct{}),
		callback: callback,
	}
}

// GetBlockReceipts retrieves all receipts for a given block number
func (t *RPCFinder) GetBlockReceipts(ctx context.Context, blockNumber *big.Int) (types.Receipts, error) {
	receipts, err := t.client.BlockReceipts(ctx,
		rpc.BlockNumberOrHashWithNumber(
			rpc.BlockNumber(blockNumber.Uint64())))
	if err != nil {
		return nil, err
	}
	return receipts, nil
}

// SubscribeToNewBlocks subscribes to new blocks and processes their receipts
func (t *RPCFinder) SubscribeToNewBlocks(ctx context.Context) error {
	sub, err := t.client.SubscribeNewHead(ctx, t.inbox)
	if err != nil {
		t.log.Error("failed to subscribe to new blocks", "error", err)
		return err
	}
	if sub != nil {
		t.sub = sub
		t.subErr = sub.Err()
	} else {
		t.log.Warn("nil subscription returned from SubscribeNewHead")
	}
	return nil
}

func (t *RPCFinder) Start(ctx context.Context) error {
	if err := t.SubscribeToNewBlocks(ctx); err != nil {
		return err
	}
	go t.Run(ctx)
	return nil
}

func (t *RPCFinder) Run(ctx context.Context) {
	for {
		select {
		// if the finder is closed, close the inbox and outbox and end the loop
		case <-t.closed:
			t.log.Info("finder closed")
			close(t.inbox)
			return
		// if the subscription errors, close the finder and initiate Stop
		case err := <-t.subErr:
			t.log.Error("subscription error, closing finder", "error", err)
			t.Stop()
		// if the inbox has a new header, process the block and send the jobs to the outbox
		case header := <-t.inbox:
			t.log.Info("received new header", "number", header.Number, "hash", header.Hash())
			jobs, err := t.ProcessBlock(ctx, header)
			if err != nil {
				t.log.Error("error processing block", "error", err)
				continue
			}
			// give all jobs the same firstSeen time
			seen := time.Now()
			for _, job := range jobs {
				job.firstSeen = seen
				job.status = []jobStatus{jobStatusUnknown}
				t.callback(job)
			}
			if len(jobs) > 0 {
				t.log.Info("sent new jobs to callback", "count", len(jobs))
			} else {
				t.log.Trace("no new jobs found")
			}
		}
	}
}

// ProcessBlock retrieves a block of receipts, converts them to jobs, and returns the jobs to be tracked
func (t *RPCFinder) ProcessBlock(ctx context.Context, header *types.Header) (cases []*Job, err error) {
	// check and warn if the parent hash is not the last seen block
	// TODO: initiate a backfilling routine if there is a gap
	if t.lastHandled != (eth.BlockID{}) {
		if header.Hash().Cmp(t.lastHandled.Hash) == 0 {
			t.log.Trace("already processed block", "hash", header.Hash())
			return nil, nil
		}
		if t.lastHandled.Number+1 != header.Number.Uint64() {
			t.log.Info("job finder experience block discontinuity", "expectedHeight", t.lastHandled.Number+1, "actualHeight", header.Number)
		} else if header.ParentHash.Cmp(t.lastHandled.Hash) != 0 {
			t.log.Info("job finder experience parent hash discontinuity", "expectedHash", t.lastHandled.Hash, "actualHash", header.ParentHash)
			return nil, nil
		}
	}
	receipts, err := t.GetBlockReceipts(ctx, header.Number)
	if err != nil {
		return nil, err
	}
	ret := t.toJobs(receipts)
	t.lastHandled = eth.BlockID{Number: header.Number.Uint64(), Hash: header.Hash()}
	t.log.Trace("last handled block", "number", t.lastHandled.Number, "hash", t.lastHandled.Hash)
	return ret, nil
}

// TODO: add wait group to make Stop return sync
func (t *RPCFinder) Stop() error {
	if t.sub != nil {
		t.sub.Unsubscribe()
	}
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
