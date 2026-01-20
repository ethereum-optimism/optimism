package interop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

// Compile-time interface conformance assertions.
var (
	_ activity.RunnableActivity     = (*Interop)(nil)
	_ activity.VerificationActivity = (*Interop)(nil)
)

// InteropActivationTimestampFlag is the CLI flag for the interop activation timestamp.
var InteropActivationTimestampFlag = &cli.Uint64Flag{
	Name:  "interop.activation-timestamp",
	Usage: "The timestamp at which interop should start",
	Value: 0,
}

func init() {
	flags.RegisterActivityFlags(InteropActivationTimestampFlag)
}

// Interop is a VerificationActivity that can also run background work as a RunnableActivity.
type Interop struct {
	log                 gethlog.Logger
	chains              map[eth.ChainID]cc.ChainContainer
	activationTimestamp uint64

	verifiedDB *VerifiedDB

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	l1Client   *sources.L1Client
	currentL1s map[eth.ChainID]eth.BlockID
}

func (i *Interop) Name() string {
	return "interop"
}

// New constructs a new Interop activity.
func New(
	log gethlog.Logger,
	activationTimestamp uint64,
	chains map[eth.ChainID]cc.ChainContainer,
	dataDir string,
	l1Client *sources.L1Client,
) *Interop {
	verifiedDB, err := OpenVerifiedDB(dataDir)
	if err != nil {
		log.Error("failed to open verified DB", "err", err)
		return nil
	}
	return &Interop{
		log:                 log,
		chains:              chains,
		l1Client:            l1Client,
		verifiedDB:          verifiedDB,
		currentL1s:          make(map[eth.ChainID]eth.BlockID),
		activationTimestamp: activationTimestamp,
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
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		case <-ticker.C:
			// Check the L1s of each chain prior to performing interop
			currentL1s, err := i.collectCurrentL1s()
			if err != nil {
				i.log.Error("failed to collect current L1s", "err", err)
				time.Sleep(2 * time.Second)
				continue
			}
			// Perform the interop evaluation
			err = i.progressInterop()
			if err != nil {
				i.log.Error("failed to process interop", "err", err)
				time.Sleep(2 * time.Second)
				continue
			}
			// Once the interop is complete, update the current L1s
			// This ensures the current L1s were available for the whole interop process.
			err = i.updateCurrentL1s(currentL1s)
			if err != nil {
				i.log.Error("failed to update current L1s", "err", err)
				time.Sleep(2 * time.Second)
				continue
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
	if i.verifiedDB != nil {
		return i.verifiedDB.Close()
	}
	return nil
}

// collectCurrentL1s collects the current L1 heads of all chains.
// currentL1 is *not* the L1 head related to the verified timestamp, but rather the
// furthest L1 head that has been processed by the chain.
func (i *Interop) collectCurrentL1s() (map[eth.ChainID]eth.BlockID, error) {
	currentL1s := map[eth.ChainID]eth.BlockID{}
	for _, chain := range i.chains {
		status, err := chain.SyncStatus(i.ctx)
		if err != nil {
			return nil, fmt.Errorf("chain %s not ready: %w", chain.ID(), err)
		}
		block := status.CurrentL1
		// Check if the block is empty (derivation pipeline hasn't started yet)
		if block == (eth.BlockRef{}) {
			return nil, fmt.Errorf("chain %s not ready: CurrentL1 not yet populated", chain.ID())
		}
		currentL1s[chain.ID()] = block.ID()
	}
	return currentL1s, nil
}

var ErrInconsistentL1Heads = errors.New("inconsistent L1 heads")

// updateCurrentL1s updates the current processed L1s if they are consistent
func (i *Interop) updateCurrentL1s(currentL1s map[eth.ChainID]eth.BlockID) error {
	for _, l1Head := range currentL1s {
		i.log.Info("updating current L1s", "l1Head", l1Head)
		// if the current L1 head is empty, no inconsistency to consider
		if l1Head == (eth.BlockID{}) {
			continue
		}
		header, err := i.l1Client.L1BlockRefByNumber(i.ctx, l1Head.Number)
		if err != nil {
			return err
		}
		if header.ID() != l1Head {
			return ErrInconsistentL1Heads
		}
	}
	i.currentL1s = currentL1s
	i.log.Info("updated current L1s", "currentL1s", currentL1s)
	return nil
}

func (i *Interop) progressInterop() error {
	start := time.Now()
	defer func() {
		i.log.Info("progressInterop: time taken", "time", time.Since(start))
	}()

	// 0: identify the next timestamp to process.
	// The next timestamp to process is the previous timestamp + 1.
	// if the database is not initialized, we use the activation timestamp instead.
	lastTimestamp, initialized := i.verifiedDB.LastTimestamp()
	i.log.Info("last timestamp", "lastTimestamp", lastTimestamp, "initialized", initialized)
	i.log.Info("activation timestamp", "activationTimestamp", i.activationTimestamp)
	var ts uint64
	if !initialized {
		ts = i.activationTimestamp
	} else {
		ts = lastTimestamp + 1
	}

	// 1: check if all chains are ready to process the next timestamp.
	// if all chains are ready, we can proceed to download the logs
	blocksAtTimestamp, err := i.checkChainsReady(ts)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// if the chains are not ready, we can return early and wait for the next timestamp
			i.log.Info("chains not ready, returning early", "timestamp", ts)
			return nil
		}
		// other errors should be treated as fatal and returned to the caller
		return err
	}

	// 2: load the logs up through the next timestamp
	// the previous timestamp is assumed to already be downloaded and verified
	if err := i.loadLogs(ts); err != nil {
		i.log.Error("failed to load logs", "err", err)
		return err
	}

	// 3: validate interop messages
	// logs up through the next timestamp are to be downloaded and verified against other available data
	result, err := i.verifyInteropMessages(ts, blocksAtTimestamp)
	if err != nil {
		i.log.Error("failed to validate interop", "err", err)
		return err
	}

	// 3: check if the results are valid.
	// Any invalid results will be added to the Denylist of the chain containers.
	if !result.IsValid() {
		i.log.Error("interop validation failed", "results", result)
		for chainID, invalidHead := range result.InvalidHeads {
			i.invalidateBlock(chainID, invalidHead)
		}
		return nil
	}

	// 4: commit the verified results.
	err = i.commitVerifiedResult(ts, result.ToVerifiedResult())
	if err != nil {
		i.log.Error("failed to commit interop", "err", err)
		return nil
	}
	i.log.Info("committed verified result", "timestamp", ts)
	return nil
}

// checkChainsReady checks if all chains are ready to process the next timestamp.
// Queries all chains in parallel for better performance.
func (i *Interop) checkChainsReady(ts uint64) (map[eth.ChainID]eth.BlockID, error) {
	type result struct {
		chainID eth.ChainID
		blockID eth.BlockID
		err     error
	}

	results := make(chan result, len(i.chains))

	// Query all chains in parallel
	for _, chain := range i.chains {
		go func(c cc.ChainContainer) {
			block, err := c.BlockAtTimestamp(i.ctx, ts, eth.Safe)
			if err != nil {
				results <- result{chainID: c.ID(), err: fmt.Errorf("chain %s not ready for timestamp %d: %w", c.ID(), ts, err)}
				return
			}
			results <- result{chainID: c.ID(), blockID: block.ID()}
		}(chain)
	}

	// Collect results
	blocksAtTimestamp := make(map[eth.ChainID]eth.BlockID)
	for range i.chains {
		r := <-results
		if r.err != nil {
			return nil, r.err
		}
		blocksAtTimestamp[r.chainID] = r.blockID
	}

	return blocksAtTimestamp, nil
}

func (i *Interop) loadLogs(ts uint64) error {
	return nil
}

func (i *Interop) verifyInteropMessages(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
	result := Result{Timestamp: ts, L2Heads: make(map[eth.ChainID]eth.BlockID)}
	for _, chain := range i.chains {
		blockID := blocksAtTimestamp[chain.ID()]
		result.L2Heads[chain.ID()] = blockID
	}
	return result, nil
}

func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID) error {
	return nil
}

func (i *Interop) commitVerifiedResult(timestamp uint64, verifiedResult VerifiedResult) error {
	return i.verifiedDB.Commit(verifiedResult)
}

// CurrentL1 returns the L1 block which has been fully considered for interop,
// whether or not it advanced the verified timestamp.
func (i *Interop) CurrentL1() eth.BlockID {
	minCurrentL1 := eth.BlockID{}
	for _, currentL1 := range i.currentL1s {
		if minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1
		}
		if currentL1.Number < minCurrentL1.Number {
			minCurrentL1 = currentL1
		}
	}
	return minCurrentL1
}

// VerifiedAtTimestamp returns whether the data is verified at the given timestamp.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	return i.verifiedDB.Has(ts)
}
