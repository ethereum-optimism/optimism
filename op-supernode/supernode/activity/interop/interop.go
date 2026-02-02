package interop

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

// Compile-time interface conformance assertions.
var (
	_            activity.RunnableActivity     = (*Interop)(nil)
	_            activity.VerificationActivity = (*Interop)(nil)
	tickerPeriod                               = 500 * time.Millisecond
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
	log                 log.Logger
	chains              map[eth.ChainID]cc.ChainContainer
	activationTimestamp uint64
	dataDir             string

	verifiedDB *VerifiedDB
	logsDBs    map[eth.ChainID]LogsDB

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool

	currentL1 eth.BlockID

	verifyFn func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error)
}

func (i *Interop) Name() string {
	return "interop"
}

// New constructs a new Interop activity.
func New(
	log log.Logger,
	activationTimestamp uint64,
	chains map[eth.ChainID]cc.ChainContainer,
	dataDir string,
) *Interop {
	verifiedDB, err := OpenVerifiedDB(dataDir)
	if err != nil {
		log.Error("failed to open verified DB", "err", err)
		return nil
	}

	// Initialize logsDBs for each chain
	logsDBs := make(map[eth.ChainID]LogsDB)
	for chainID := range chains {
		logsDB, err := openLogsDB(log, chainID, dataDir)
		if err != nil {
			log.Error("failed to open logs DB for chain", "chainID", chainID, "err", err)
			// Clean up already created logsDBs
			for _, db := range logsDBs {
				_ = db.Close()
			}
			_ = verifiedDB.Close()
			return nil
		}
		logsDBs[chainID] = logsDB
	}

	i := &Interop{
		log:                 log,
		chains:              chains,
		verifiedDB:          verifiedDB,
		logsDBs:             logsDBs,
		dataDir:             dataDir,
		currentL1:           eth.BlockID{},
		activationTimestamp: activationTimestamp,
	}
	// default to using the verifyInteropMessages function
	// (can be overridden by tests)
	i.verifyFn = i.verifyInteropMessages
	return i
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
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return i.ctx.Err()
		case <-ticker.C:
			err := i.progressAndRecord()
			if err != nil {
				i.log.Error("failed to progress and record interop", "err", err)
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
	// Close all logsDBs
	for chainID, db := range i.logsDBs {
		if err := db.Close(); err != nil {
			i.log.Error("failed to close logs DB", "chainID", chainID, "err", err)
		}
	}
	if i.verifiedDB != nil {
		return i.verifiedDB.Close()
	}
	return nil
}

func (i *Interop) progressAndRecord() error {
	// Check the L1s of each chain prior to performing interop
	localCurrentL1, err := i.collectCurrentL1()
	if err != nil {
		i.log.Error("failed to collect current L1", "err", err)
		return err
	}
	// Perform the interop evaluation
	result, err := i.progressInterop()
	if err != nil {
		i.log.Error("failed to progress interop", "err", err)
		return err
	}

	// Handle the result by committing verified results or invalidating blocks
	err = i.handleResult(result)
	if err != nil {
		i.log.Error("failed to handle result", "err", err)
		return err
	}
	// if the result is invalid, exit without updating the current L1s
	if !result.IsEmpty() && !result.IsValid() {
		i.log.Warn("result is invalid, skipping current L1 update", "results", result)
		return nil
	}

	// Once interop is complete and recorded, update the current L1s
	// the current L1s being considered by the Activity right now depend on what progress was made:
	// - if interop failed to run, the current L1s are not updated
	// - if interop ran but did not advance the verified timestamp, the CurrentL1 values collected are used directly
	// - if interop ran and advanced the verified timestamp, the CurrentL1 is the L1 head at the verified timestamp
	// this is because the individual chains may advance their CurrentL1, and if progress is being made, we might not be done using the collected L1s.
	i.mu.Lock()
	if !result.IsEmpty() {
		// the new CurrentL1 is the L1 head at the verified timestamp
		i.currentL1 = result.L1Head
	} else {
		// the new CurrentL1 is the lowest CurrentL1 from the collected chains
		i.currentL1 = localCurrentL1
	}
	i.mu.Unlock()
	return nil
}

// collectCurrentL1 collects the current L1 head of all chains,
// which is the minimum L1 head of all the derivation pipelines in Chain Containers
func (i *Interop) collectCurrentL1() (eth.BlockID, error) {
	var currentL1 eth.BlockID
	first := true
	for _, chain := range i.chains {
		status, err := chain.SyncStatus(i.ctx)
		if err != nil {
			return eth.BlockID{}, fmt.Errorf("chain %s not ready: %w", chain.ID(), err)
		}
		block := status.CurrentL1
		if first || block.Number < currentL1.Number {
			currentL1 = block.ID()
			first = false
		}
	}
	return currentL1, nil
}

func (i *Interop) progressInterop() (Result, error) {
	start := time.Now()
	defer func() {
		i.log.Debug("progressInterop: time taken", "time", time.Since(start))
	}()

	// 0: identify the next timestamp to process.
	// The next timestamp to process is the previous timestamp + 1.
	// if the database is not initialized, we use the activation timestamp instead.
	lastTimestamp, initialized := i.verifiedDB.LastTimestamp()
	var ts uint64
	if !initialized {
		i.log.Info("initializing interop activity with activation timestamp", "activationTimestamp", i.activationTimestamp)
		ts = i.activationTimestamp
	} else {
		i.log.Info("attempting to progress interop to next timestamp", "lastTimestamp", lastTimestamp, "timestamp", lastTimestamp+1)
		ts = lastTimestamp + 1
	}

	// 1: check if all chains are ready to process the next timestamp.
	// if all chains are ready, we can proceed to download the logs
	blocksAtTimestamp, err := i.checkChainsReady(ts)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// if the chains are not ready, we can return early and wait for the next timestamp
			// no error is returned, as this is expected behavior
			i.log.Info("chains not ready, returning early", "timestamp", ts)
			return Result{}, nil
		}
		// other errors should be treated as fatal and returned to the caller
		return Result{}, err
	}

	// 2: load the logs up through the next timestamp
	// the previous timestamp is assumed to already be downloaded and verified
	if err := i.loadLogs(ts); err != nil {
		i.log.Error("failed to load logs", "err", err)
		return Result{}, err
	}

	// 3: validate interop messages
	// and return the result and any errors
	return i.verifyFn(ts, blocksAtTimestamp)
}

func (i *Interop) handleResult(result Result) error {
	// if the result is empty, return nil
	if result.IsEmpty() {
		return nil
	}

	// if the result is invalid, invalidate the blocks and return
	if !result.IsValid() {
		i.log.Error("interop validation failed", "results", result)
		for chainID, invalidHead := range result.InvalidHeads {
			if err := i.invalidateBlock(chainID, invalidHead); err != nil {
				i.log.Error("failed to invalidate block", "chainID", chainID, "blockID", invalidHead, "err", err)
				return err
			}
		}
		return nil
	}

	// if the result is valid, commit the verified result
	err := i.commitVerifiedResult(result.Timestamp, result.ToVerifiedResult())
	if err != nil {
		i.log.Error("failed to commit verified result", "err", err)
		return err
	}
	i.log.Info("committed verified result", "timestamp", result.Timestamp)
	return nil
}

// invalidateBlock handles an invalid block by notifying the chain to reorg.
func (i *Interop) invalidateBlock(chainID eth.ChainID, blockID eth.BlockID) error {
	// TODO(#18944): Implement block invalidation
	// This should trigger the chain container to reorg away from the invalid block
	i.log.Warn("invalidateBlock called but not implemented", "chainID", chainID, "blockID", blockID)
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

func (i *Interop) commitVerifiedResult(timestamp uint64, verifiedResult VerifiedResult) error {
	return i.verifiedDB.Commit(verifiedResult)
}

// CurrentL1 returns the L1 block which has been fully considered for interop,
// whether or not it advanced the verified timestamp.
func (i *Interop) CurrentL1() eth.BlockID {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.currentL1
}

// VerifiedAtTimestamp returns whether the data is verified at the given timestamp.
// For timestamps before the activation timestamp, this returns true since interop
// wasn't active yet and verification proceeds automatically.
// For timestamps at or after the activation timestamp, this checks the verifiedDB.
func (i *Interop) VerifiedAtTimestamp(ts uint64) (bool, error) {
	// Timestamps before the activation timestamp are considered verified
	// because interop wasn't active yet
	if ts < i.activationTimestamp {
		return true, nil
	}
	return i.verifiedDB.Has(ts)
}
