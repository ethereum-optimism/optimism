package interop

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Compile-time interface conformance assertions.
var (
	_ activity.RunnableActivity     = (*Interop)(nil)
	_ activity.VerificationActivity = (*Interop)(nil)
)

// Interop is a VerificationActivity that can also run background work.
// Skeleton implementation: wiring, lifecycle, and interface surface only.
type Interop struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer

	lastProcessedTimestamp uint64

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// New constructs a new Interop activity.
func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Interop {
	return &Interop{
		log:    log,
		chains: chains,
	}
}

// Start begins the Interop activity background loop and blocks until ctx is canceled.
func (i *Interop) Start(ctx context.Context) error {
	i.mu.Lock()
	if i.started {
		i.mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	i.ctx, i.cancel = context.WithCancel(ctx)
	i.started = true
	i.mu.Unlock()

	// Periodically query each chain container for its current safe head and log it.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		case <-ticker.C:
			err := i.processInterop()
			if err != nil {
				i.log.Error("failed to process interop", "err", err)
			}
		}
	}
}

// Stop stops the Interop activity.
func (i *Interop) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.cancel != nil {
		i.cancel()
	}
	return nil
}

func (i *Interop) processInterop() error {
	// 1: check if all chains are ready to process the next timestamp.
	// The next timestamp to process is the previous timestamp + 1.
	// if all chains are ready, we can process the next timestamp.
	nextTimestamp := i.lastProcessedTimestamp + 1
	ready, err := i.checkChainsReady(nextTimestamp)
	if err != nil {
		i.log.Error("failed to check chains ready", "err", err)
	}
	if !ready {
		i.log.Info("chains not ready, skipping timestamp", "timestamp", nextTimestamp)
		return nil
	}

	// 2: validate interop messages
	// logs up through the next timestamp are to be downloaded and verified against other available data
	results, err := i.validateInterop(nextTimestamp)
	if err != nil {
		i.log.Error("failed to validate interop", "err", err)
		return err
	}

	// 3: check if the results are valid.
	// Any invalid results will be added to the Denylist of the chain containers.
	allValid := true
	for chainID, result := range results {
		if result.err != nil {
			allValid = false
			i.log.Error("interop validation failed", "chain", chainID, "result", result)
			i.invalidateBlock(chainID, result.blockID)
		}
	}
	if !allValid {
		i.log.Info("interop validation failed, skipping timestamp", "timestamp", nextTimestamp)
		return nil
	}

	// 4: commit the valid results.
	err = i.commitValidResults(results)
	if err != nil {
		i.log.Error("failed to commit interop", "err", err)
		return nil
	}
	return nil
}

func (i *Interop) checkChainsReady(ts uint64) (bool, error) {
	return true, nil
}

func (i *Interop) validateInterop(ts uint64) (map[eth.ChainID]interopResult, error) {
	return nil, nil
}

func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID) error {
	return nil
}

func (i *Interop) commitValidResults(results map[eth.ChainID]interopResult) error {
	return nil
}

type interopResult struct {
	chainID eth.ChainID
	blockID eth.BlockID
	err     error
}

// CurrentL1 reports the most recent verified L1 block known to this verification activity.
// Skeleton returns the zero value until implemented.
func (i *Interop) CurrentL1() eth.BlockID {
	return eth.BlockID{}
}

// VerifiedAtTimestamp returns whether the data is verified at the given timestamp.
// Skeleton returns false until implemented.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	return false, nil
}
