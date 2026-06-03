package dsl

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

// L2CLNode wraps a stack.L2CLNode interface for DSL operations
type L2CLNode struct {
	commonImpl
	inner        stack.L2CLNode
	managedPeers map[string]*L2CLNode
}

// NewL2CLNode creates a new L2CLNode DSL wrapper
func NewL2CLNode(inner stack.L2CLNode) *L2CLNode {
	return &L2CLNode{
		commonImpl:   commonFromT(inner.T()),
		inner:        inner,
		managedPeers: make(map[string]*L2CLNode),
	}
}

func (cl *L2CLNode) Name() string {
	return cl.inner.Name()
}

func (cl *L2CLNode) String() string {
	return cl.inner.Name()
}

// Escape returns the underlying stack.L2CLNode
func (cl *L2CLNode) Escape() stack.L2CLNode {
	return cl.inner
}

func (cl *L2CLNode) SafeL2BlockRef() eth.L2BlockRef {
	return cl.HeadBlockRef(safety.CrossSafe)
}

func (cl *L2CLNode) Start() {
	lifecycle, ok := cl.inner.(stack.Lifecycle)
	cl.require.Truef(ok, "L2CL node %s is not lifecycle-controllable", cl.inner.Name())
	lifecycle.Start()
	cl.restoreManagedPeers()
}

func (cl *L2CLNode) Stop() {
	lifecycle, ok := cl.inner.(stack.Lifecycle)
	cl.require.Truef(ok, "L2CL node %s is not lifecycle-controllable", cl.inner.Name())
	lifecycle.Stop()
}

func (cl *L2CLNode) ManagePeer(peer *L2CLNode) {
	cl.managedPeers[peer.Name()] = peer
	peer.managedPeers[cl.Name()] = cl
}

func (cl *L2CLNode) restoreManagedPeers() {
	for _, peer := range cl.managedPeers {
		cl.connectPeerRaw(peer)
		peer.connectPeerRaw(cl)
	}
}

func (cl *L2CLNode) StartSequencer() {
	// The op-node Sequencer.Start RPC requires the caller to pass the hash of op-node's
	// current unsafe head. Reading the head and issuing the start call are two separate
	// RPCs, so any unsafe payload that op-node processes in between (e.g. blocks gossiped
	// by another sequencer over p2p) invalidates the hash we just read. Retry on that
	// specific mismatch error with a freshly-read head.
	const maxAttempts = 10
	var err error
loop:
	for range maxAttempts {
		unsafe := cl.HeadBlockRef(safety.LocalUnsafe)
		cl.log.Info("Continue sequencing with consensus node (op-node)", "chain", cl.ChainID(), "unsafe", unsafe)
		err = cl.inner.RollupAPI().StartSequencer(cl.ctx, unsafe.Hash)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "block hash does not match") {
			break
		}
		cl.log.Info("Unsafe head advanced between read and StartSequencer; retrying", "chain", cl.ChainID(), "err", err)
		select {
		case <-cl.ctx.Done():
			err = cl.ctx.Err()
			break loop
		case <-time.After(250 * time.Millisecond):
		}
	}
	cl.require.NoError(err, fmt.Sprintf("Expected to be able to start sequencer on chain %d", cl.ChainID()))

	// wait for the sequencer to become active
	var active bool
	err = wait.For(cl.ctx, 1*time.Second, func() (bool, error) {
		active, err = cl.inner.RollupAPI().SequencerActive(cl.ctx)
		return active, err
	})
	cl.require.NoError(err, fmt.Sprintf("Expected to be able to call SequencerActive API on chain %d, and wait for an active state for sequencer, but got error", cl.ChainID()))

	cl.log.Info("Rollup node sequencer status", "chain", cl.ChainID(), "active", active)
}

func (cl *L2CLNode) StopSequencer() common.Hash {
	unsafeHead, err := cl.inner.RollupAPI().StopSequencer(cl.ctx)
	cl.require.NoError(err, "Expected to be able to call StopSequencer API, but got error")

	// wait for the sequencer to become inactive
	var active bool
	err = wait.For(cl.ctx, 1*time.Second, func() (bool, error) {
		active, err = cl.inner.RollupAPI().SequencerActive(cl.ctx)
		return !active, err
	})
	cl.require.NoError(err, fmt.Sprintf("Expected to be able to call SequencerActive API on chain %d, and wait for inactive state for sequencer, but got error", cl.ChainID()))

	cl.log.Info("Rollup node sequencer status", "chain", cl.ChainID(), "active", active, "unsafeHead", unsafeHead)
	return unsafeHead
}

func (cl *L2CLNode) SetSequencerRecoverMode(b bool) error {
	return cl.inner.RollupAPI().SetRecoverMode(cl.ctx, b)
}

// syncStatus fetches the L2CL sync status and returns the RPC error, if any.
// Internal callers in retry/eventually loops use this so a transient RPC timeout
// counts as a retry rather than an instant FailNow.
func (cl *L2CLNode) syncStatus() (*eth.SyncStatus, error) {
	ctx, cancel := context.WithTimeout(cl.ctx, DefaultTimeout)
	defer cancel()
	return cl.inner.RollupAPI().SyncStatus(ctx)
}

func (cl *L2CLNode) SyncStatus() *eth.SyncStatus {
	syncStatus, err := cl.syncStatus()
	cl.require.NoError(err)
	return syncStatus
}

// headBlockRef is the error-returning variant of HeadBlockRef, for use inside
// retry/eventually loops.
func (cl *L2CLNode) headBlockRef(lvl safety.Level) (eth.L2BlockRef, error) {
	syncStatus, err := cl.syncStatus()
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	switch lvl {
	case safety.Finalized:
		return syncStatus.FinalizedL2, nil
	case safety.CrossSafe:
		return syncStatus.SafeL2, nil
	case safety.LocalSafe:
		return syncStatus.LocalSafeL2, nil
	case safety.CrossUnsafe:
		return syncStatus.CrossUnsafeL2, nil
	case safety.LocalUnsafe:
		return syncStatus.UnsafeL2, nil
	default:
		return eth.L2BlockRef{}, fmt.Errorf("invalid safety level: %v", lvl)
	}
}

// HeadBlockRef fetches L2CL sync status and returns block ref with given safety level
func (cl *L2CLNode) HeadBlockRef(lvl safety.Level) eth.L2BlockRef {
	blockRef, err := cl.headBlockRef(lvl)
	cl.require.NoError(err)
	return blockRef
}

func (cl *L2CLNode) ChainID() eth.ChainID {
	return cl.inner.ChainID()
}

func (cl *L2CLNode) AwaitMinL1Processed(minL1 uint64) {
	ctx, cancel := context.WithTimeout(cl.ctx, DefaultTimeout)
	defer cancel()
	// Wait for CurrentL1 to be at least one block _past_ minL1 since CurrentL1 may not yet be fully processed.
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		ss, err := cl.syncStatus()
		if err != nil {
			cl.log.Warn("SyncStatus RPC failed while awaiting L1 processed; will retry", "err", err)
			return false, nil
		}
		return ss.CurrentL1.Number > minL1, nil
	})
	cl.require.NoErrorf(err, "CurrentL1 did not reach %v", minL1+1)
}

// AdvancedFn returns a lambda that checks the L2CL chain head with given safety level advanced more than delta block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) AdvancedFn(lvl safety.Level, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		target := initial.Number + delta
		cl.log.Info("Expecting chain to advance", "name", cl.inner.Name(), "chain", cl.ChainID(), "label", lvl, "delta", delta)
		return cl.ReachedFn(lvl, target, attempts)()
	}
}

func (cl *L2CLNode) NotAdvancedFn(lvl safety.Level, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		logger := cl.log.With("name", cl.inner.Name(), "chain", cl.ChainID(), "label", lvl, "target", initial.Number)
		logger.Info("Expecting chain not to advance")
		var lastErr error
		successes := 0
		for range attempts {
			if err := clock.SystemClock.SleepCtx(cl.ctx, 2*time.Second); err != nil { // nosemgrep: flake-sleep-in-test -- asserting absence of progress; no chain event to wait on
				return err
			}
			head, err := cl.headBlockRef(lvl)
			if err != nil {
				lastErr = err
				logger.Warn("SyncStatus RPC failed; will retry", "err", err)
				continue
			}
			successes++
			logger.Info("Chain sync status", "current", head.Number)
			if head.Hash == initial.Hash {
				continue
			}
			return fmt.Errorf("expected head not to advance: %s", lvl)
		}
		if successes == 0 {
			return fmt.Errorf("could not read %s head across %d attempts: %w", lvl, attempts, lastErr)
		}
		logger.Info("Chain not advanced")
		return nil
	}
}

// awaitSafeHeadsStalled waits until every node's safe head has stopped advancing
// for at least 10 seconds.
func (cl *L2CLNode) WaitForStall(lvl safety.Level) {
	var last eth.BlockID
	var stableSince time.Time
	cl.require.Eventuallyf(func() bool {
		head, err := cl.headBlockRef(lvl)
		if err != nil {
			cl.log.Warn("SyncStatus RPC failed while waiting for stall; will retry", "err", err)
			return false
		}
		cur := head.ID()
		if cur == last {
			if stableSince.IsZero() {
				stableSince = time.Now()
			}
			return time.Since(stableSince) >= 10*time.Second
		}
		last = cur
		stableSince = time.Time{}
		return false
	}, 2*time.Minute, 2*time.Second, "expected %v head to stall", lvl)
}

// ReachedFn returns a lambda that checks the L2CL chain head with given safety level reaches the target block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) ReachedFn(lvl safety.Level, target uint64, attempts int) CheckFunc {
	return func() error {
		logger := cl.log.With("name", cl.inner.Name(), "chain", cl.ChainID(), "label", lvl, "target", target)
		logger.Info("Expecting chain to reach")
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head, err := cl.headBlockRef(lvl)
				if err != nil {
					logger.Warn("SyncStatus RPC failed; will retry", "err", err)
					return err
				}
				if head.Number >= target {
					logger.Info("Chain advanced", "target", target)
					return nil
				}
				logger.Info("Chain sync status", "current", head.Number)
				return fmt.Errorf("expected head to advance: %s", lvl)
			})
	}
}

// ReachedRefFn is same as Reached, but has an additional check to ensure that the block referenced is not reorged
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) ReachedRefFn(lvl safety.Level, target eth.BlockID, attempts int) CheckFunc {
	return func() error {
		err := cl.ReachedFn(lvl, target.Number, attempts)()
		if err != nil {
			return err
		}

		ethclient := cl.inner.ELClient()
		result, err := ethclient.BlockRefByNumber(cl.ctx, target.Number)
		if err != nil {
			return err
		}
		if result.Hash != target.Hash {
			return fmt.Errorf("expected block ref to be the same as target %s, got but %s", target.Hash, result.Hash)
		}
		return nil
	}
}

// ReachedTimeWithoutRegressionFn waits for the head's block timestamp to reach
// targetTime, failing on any regression. Requires a non-zero baseline behind
// targetTime so a regression to genesis isn't vacuous (0 >= 0). Polls every
// 100ms; deadline is targetTime + 5min.
//
// Uses an explicit ticker/deadline loop and returns errors through the
// CheckFunc contract. require.Eventuallyf would call FailNow inside the
// dsl.CheckAll worker goroutine, which is not safe.
func (cl *L2CLNode) ReachedTimeWithoutRegressionFn(lvl safety.Level, targetTime uint64) CheckFunc {
	return func() error {
		initial, err := cl.headBlockRef(lvl)
		if err != nil {
			return fmt.Errorf("read initial %s head: %w", lvl, err)
		}
		if initial.Number == 0 {
			return fmt.Errorf("initial %s head is at genesis; cannot detect regression below zero — wait for the head to advance past genesis before snapshotting", lvl)
		}
		if initial.Time >= targetTime {
			return fmt.Errorf("initial %s head time %d already at or past target %d; nothing to observe across the boundary", lvl, initial.Time, targetTime)
		}
		const buffer = 5 * time.Minute
		deadline := time.Unix(int64(targetTime), 0).Add(buffer)
		logger := cl.log.With("name", cl.inner.Name(), "chain", cl.ChainID(), "label", lvl, "initial", initial.Number, "initial_time", initial.Time, "target_time", targetTime, "deadline", deadline)
		logger.Info("Watching head for regression until target time reached")

		ctx, cancel := context.WithDeadline(cl.ctx, deadline)
		defer cancel()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			head, err := cl.headBlockRef(lvl)
			if err != nil {
				logger.Warn("SyncStatus RPC failed; will retry", "err", err)
			} else {
				if head.Number < initial.Number {
					return fmt.Errorf("%s head regressed: was %d (%s, t=%d), observed %d (%s, t=%d)",
						lvl, initial.Number, initial.Hash, initial.Time, head.Number, head.Hash, head.Time)
				}
				if head.Time >= targetTime {
					return nil
				}
				logger.Info("Chain sync status", "current", head.Number, "current_time", head.Time)
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("%s head did not reach target time %d (initial=%d): %w", lvl, targetTime, initial.Number, ctx.Err())
			case <-ticker.C:
			}
		}
	}
}

// RewindedFn returns a lambda that checks the L2CL chain head with given safety level rewinded more than the delta block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) RewindedFn(lvl safety.Level, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		cl.require.GreaterOrEqual(initial.Number, delta, "cannot rewind before genesis")
		target := initial.Number - delta
		logger := cl.log.With("name", cl.inner.Name(), "chain", cl.ChainID(), "label", lvl)
		logger.Info("Expecting chain to rewind", "target", target, "delta", delta)
		// check rewind more aggressively, in shorter interval
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 250 * time.Millisecond},
			func() error {
				head, err := cl.headBlockRef(lvl)
				if err != nil {
					logger.Warn("SyncStatus RPC failed; will retry", "err", err)
					return err
				}
				if head.Number <= target {
					logger.Info("Chain rewinded", "target", target)
					return nil
				}
				logger.Info("Chain sync status", "target", target, "current", head.Number)
				return fmt.Errorf("expected head to rewind: %s", lvl)
			})
	}
}

func (cl *L2CLNode) Advanced(lvl safety.Level, delta uint64, attempts int) {
	cl.require.NoError(cl.AdvancedFn(lvl, delta, attempts)())
}

func (cl *L2CLNode) AdvancedUnsafe(delta uint64, attempts int) {
	cl.Advanced(safety.LocalUnsafe, delta, attempts)
}

func (cl *L2CLNode) NotAdvanced(lvl safety.Level, attempts int) {
	cl.require.NoError(cl.NotAdvancedFn(lvl, attempts)())
}

func (cl *L2CLNode) NotAdvancedUnsafe(attempts int) {
	cl.NotAdvanced(safety.LocalUnsafe, attempts)
}

func (cl *L2CLNode) Reached(lvl safety.Level, target uint64, attempts int) {
	cl.require.NoError(cl.ReachedFn(lvl, target, attempts)())
}

func (cl *L2CLNode) ReachedUnsafe(target uint64, attempts int) {
	cl.Reached(safety.LocalUnsafe, target, attempts)
}

func (cl *L2CLNode) ReachedRef(lvl safety.Level, target eth.BlockID, attempts int) {
	cl.require.NoError(cl.ReachedRefFn(lvl, target, attempts)())
}

func (cl *L2CLNode) Rewinded(lvl safety.Level, delta uint64, attempts int) {
	cl.require.NoError(cl.RewindedFn(lvl, delta, attempts)())
}

// ChainSyncStatus satisfies that the L2CLNode can provide sync status per chain
func (cl *L2CLNode) ChainSyncStatus(chainID eth.ChainID, lvl safety.Level) eth.BlockID {
	cl.require.Equal(chainID, cl.inner.ChainID(), "chain ID mismatch")
	return cl.HeadBlockRef(lvl).ID()
}

func (cl *L2CLNode) ChainBlockID(chainID eth.ChainID, number uint64) (eth.BlockID, error) {
	cl.require.Equal(chainID, cl.inner.ChainID(), "chain ID mismatch")
	ref, err := cl.inner.ELClient().BlockRefByNumber(cl.ctx, number)
	if err != nil {
		return eth.BlockID{}, err
	}
	return ref.ID(), nil
}

func (cl *L2CLNode) safeHeadAtL1Block(l1BlockNum uint64) *eth.SafeHeadResponse {
	resp, err := cl.inner.RollupAPI().SafeHeadAtL1Block(cl.ctx, l1BlockNum)
	if errors.Is(err, safedb.ErrNotFound) {
		return nil
	}
	cl.require.NoErrorf(err, "failed to get safe head at l1 block %v", l1BlockNum)
	return resp
}

// LaggedFn returns a lambda that checks the L2CL chain head with given safety level is lagged with the reference chain sync status provider
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) LaggedFn(refNode SyncStatusProvider, lvl safety.Level, attempts int, allowMatch bool) CheckFunc {
	return LaggedFn(cl, refNode, cl.log, cl.ctx, lvl, cl.ChainID(), attempts, allowMatch)
}

// MatchedFn returns a lambda that checks the L2CLNode head with given safety level is matched with the refNode chain sync status provider
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) MatchedFn(refNode SyncStatusProvider, lvl safety.Level, attempts int) CheckFunc {
	return MatchedFn(cl, refNode, cl.log, cl.ctx, lvl, cl.ChainID(), attempts)
}

// MatchedWithProgressFn returns a lambda that waits for cl's matchLvl head to
// match refNode's matchLvl head while requiring cl to keep making progress on
// progressLvl. See MatchedWithProgressFn in check.go for the precise semantics.
// Composable with other lambdas to wait in parallel.
func (cl *L2CLNode) MatchedWithProgressFn(refNode SyncStatusProvider, matchLvl, progressLvl safety.Level, maxWait, stallTimeout time.Duration) CheckFunc {
	return MatchedWithProgressFn(cl, refNode, cl.log, cl.ctx, matchLvl, progressLvl, cl.ChainID(), maxWait, stallTimeout)
}

func (cl *L2CLNode) InSyncFn(other SyncStatusProvider, lvl safety.Level, attempts int) CheckFunc {
	return InSyncFn(cl, other, cl.log, cl.ctx, lvl, cl.ChainID(), attempts)
}

func (cl *L2CLNode) Lagged(refNode SyncStatusProvider, lvl safety.Level, attempts int, allowMatch bool) {
	cl.require.NoError(cl.LaggedFn(refNode, lvl, attempts, allowMatch)())
}

func (cl *L2CLNode) Matched(refNode SyncStatusProvider, lvl safety.Level, attempts int) {
	cl.require.NoError(cl.MatchedFn(refNode, lvl, attempts)())
}

func (cl *L2CLNode) InSync(other SyncStatusProvider, lvl safety.Level, attempts int) {
	cl.require.NoError(cl.InSyncFn(other, lvl, attempts)())
}

func (cl *L2CLNode) MatchedUnsafe(refNode SyncStatusProvider, attempts int) {
	cl.Matched(refNode, safety.LocalUnsafe, attempts)
}

func (cl *L2CLNode) PeerInfo() *apis.PeerInfo {
	peerInfo, err := retry.Do(cl.ctx, 3, retry.Exponential(), func() (*apis.PeerInfo, error) {
		return cl.inner.P2PAPI().Self(cl.ctx)
	})
	cl.require.NoError(err, "failed to get peer info")
	return peerInfo
}

func (cl *L2CLNode) Peers() *apis.PeerDump {
	peerDump, err := retry.Do(cl.ctx, 3, retry.Exponential(), func() (*apis.PeerDump, error) {
		return cl.inner.P2PAPI().Peers(cl.ctx, true)
	})
	cl.require.NoError(err, "failed to get peers")
	return peerDump
}

func (cl *L2CLNode) DisconnectPeer(peer *L2CLNode) {
	delete(cl.managedPeers, peer.Name())
	delete(peer.managedPeers, cl.Name())
	cl.disconnectPeerRaw(peer)
}

func (cl *L2CLNode) disconnectPeerRaw(peer *L2CLNode) {
	peerInfo := peer.PeerInfo()
	err := retry.Do0(cl.ctx, 3, retry.Exponential(), func() error {
		return cl.inner.P2PAPI().DisconnectPeer(cl.ctx, peerInfo.PeerID)
	})
	cl.require.NoError(err, "failed to disconnect peer")
}

func (cl *L2CLNode) ConnectPeer(peer *L2CLNode) {
	cl.managedPeers[peer.Name()] = peer
	peer.managedPeers[cl.Name()] = cl
	cl.connectPeerRaw(peer)
}

func (cl *L2CLNode) connectPeerRaw(peer *L2CLNode) {
	peerInfo := peer.PeerInfo()
	cl.require.NotZero(len(peerInfo.Addresses), "failed to get peer address")
	// graceful backoff for p2p connection, to avoid dial backoff or connection refused error
	strategy := &retry.ExponentialStrategy{Min: 10 * time.Second, Max: 30 * time.Second, MaxJitter: 250 * time.Millisecond}
	err := retry.Do0(cl.ctx, 5, strategy, func() error {
		return cl.inner.P2PAPI().ConnectPeer(cl.ctx, peerInfo.Addresses[0])
	})
	cl.require.NoError(err, "failed to connect peer")
}

func (cl *L2CLNode) IsP2PConnected(peer *L2CLNode) {
	myInfo := cl.PeerInfo()
	strategy := &retry.ExponentialStrategy{Min: 10 * time.Second, Max: 30 * time.Second, MaxJitter: 250 * time.Millisecond}
	err := retry.Do0(cl.ctx, 5, strategy, func() error {
		for _, p := range peer.Peers().Peers {
			if p.PeerID == myInfo.PeerID {
				return nil
			}
		}
		return errors.New("peer not connected yet")
	})
	cl.require.NoError(err, "peer not connected")
}

func (cl *L2CLNode) WaitForPeerDisconnected(peer *L2CLNode) {
	myInfo := cl.PeerInfo()
	strategy := &retry.ExponentialStrategy{Min: 10 * time.Second, Max: 30 * time.Second, MaxJitter: 250 * time.Millisecond}
	err := retry.Do0(cl.ctx, 5, strategy, func() error {
		for _, p := range peer.Peers().Peers {
			if p.PeerID == myInfo.PeerID {
				return errors.New("peer still connected")
			}
		}
		return nil
	})
	cl.require.NoError(err, "peer not disconnected")
}

type safeHeadDbMatchOpts struct {
	minRequiredL2Block *uint64
}

func WithMinRequiredL2Block(blockNum uint64) func(opts *safeHeadDbMatchOpts) {
	return func(opts *safeHeadDbMatchOpts) {
		opts.minRequiredL2Block = &blockNum
	}
}

func (cl *L2CLNode) VerifySafeHeadDatabaseMatches(sourceOfTruth *L2CLNode, args ...func(opts *safeHeadDbMatchOpts)) {
	opts := applyOpts(safeHeadDbMatchOpts{}, args...)
	l1Block := cl.SyncStatus().CurrentL1.Number
	cl.log.Info("Verifying safe head database matches", "maxL1Block", l1Block)
	cl.AwaitMinL1Processed(l1Block) // Ensure this block is fully processed before checking safe head db
	sourceOfTruth.AwaitMinL1Processed(l1Block)
	checkSafeHeadConsistent(cl.t, l1Block, cl, sourceOfTruth, opts.minRequiredL2Block)
}

func (cl *L2CLNode) WaitForNonZeroUnsafeTime(ctx context.Context) *eth.SyncStatus {
	require := cl.require

	var ss *eth.SyncStatus
	err := retry.Do0(ctx, 10, retry.Fixed(2*time.Second), func() error {
		ss = cl.SyncStatus()
		require.NotNil(ss, "L2CL should have sync status")
		if ss.UnsafeL2.Time == 0 {
			return fmt.Errorf("L2CL unsafe time is still zero")
		}
		return nil
	})
	require.NoError(err, "L2CL unsafe time should be set within retry limit")
	require.NotZero(ss.UnsafeL2.Time, "L2CL unsafe time should not be zero")

	return ss
}

func (cl *L2CLNode) SignalTarget(refNode *L2ELNode, targetNum uint64) {
	cl.log.Info("Signaling L2CL", "target", targetNum, "refNode", refNode)
	payload := refNode.PayloadByNumber(targetNum)
	cl.PostUnsafePayload(payload)
}

func (cl *L2CLNode) PostUnsafePayload(payload *eth.ExecutionPayloadEnvelope) {
	cl.log.Info("PostUnsafePayload", "target", payload.ExecutionPayload.BlockNumber)
	err := retry.Do0(cl.ctx, 3, retry.Fixed(2*time.Second), func() error {
		return cl.inner.RollupAPI().PostUnsafePayload(cl.ctx, payload)
	})
	cl.require.NoErrorf(err, "failed to post unsafe payload via admin API: target %d", payload.ExecutionPayload.BlockNumber)
}

func (cl *L2CLNode) Reset(lvl safety.Level, target eth.L2BlockRef) {
	cl.require.NoError(retry.Do0(cl.ctx, 5, &retry.FixedStrategy{Dur: 2 * time.Second},
		func() error {
			res := cl.HeadBlockRef(lvl)
			cl.log.Info("Chain sync Status", lvl, res)
			if res.Hash == target.Hash {
				return nil
			}
			return errors.New("waiting to reset")
		}))
}

func (cl *L2CLNode) AppendUnsafePayloadUntilTip(verEL, seqEL *L2ELNode, maxAttempts int) {
	trial := 0
	cl.require.NoError(
		retry.Do0(cl.ctx, 200, &retry.FixedStrategy{Dur: 250 * time.Millisecond}, func() error {
			verUnsafe := verEL.BlockRefByLabel(eth.Unsafe)
			seqUnsafe := seqEL.BlockRefByLabel(eth.Unsafe)
			gap := seqUnsafe.Number - verUnsafe.Number
			cl.log.Info("Filling in the gap by appending unsafe payload", "gap", gap, "ver", verUnsafe, "seq", seqUnsafe, "trial", trial)
			if gap == 0 {
				return nil
			}
			trial += 1
			cl.SignalTarget(seqEL, verUnsafe.Number+1)
			return fmt.Errorf("unsafe gap with size %d still exists", gap)
		}))
}

func (cl *L2CLNode) UnsafeHead() *BlockRefResult {
	return &BlockRefResult{T: cl.t, BlockRef: cl.HeadBlockRef(safety.LocalUnsafe)}
}

func (cl *L2CLNode) SafeHead() *BlockRefResult {
	return &BlockRefResult{T: cl.t, BlockRef: cl.HeadBlockRef(safety.CrossSafe)}
}

func (cl *L2CLNode) CurrentL1MatchedFn(refNode *L2CLNode, attempts int) CheckFunc {
	return func() error {
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 1 * time.Second},
			func() error {
				currentL1 := cl.SyncStatus().CurrentL1
				ref := refNode.SyncStatus().CurrentL1
				if currentL1 == ref {
					cl.log.Info("CurrentL1 reached", "currentL1", currentL1)
					return nil
				}
				cl.log.Info("Chain sync status", "currentL1", currentL1.Number, "ref", ref)
				return fmt.Errorf("expected currentL1 to match")
			})
	}
}

func (cl *L2CLNode) LocalGameInputs(agreedBlock, claimBlock uint64) *utils.LocalGameInputs {
	cl.Reached(safety.LocalSafe, claimBlock, 60)

	rollupAPI := cl.Escape().RollupAPI()

	agreedOutput, err := rollupAPI.OutputAtBlock(cl.ctx, agreedBlock)
	cl.require.NoError(err, "fetch output at agreed block %d", agreedBlock)

	claimedOutput, err := rollupAPI.OutputAtBlock(cl.ctx, claimBlock)
	cl.require.NoError(err, "fetch output at claim block %d", claimBlock)

	syncStatus, err := rollupAPI.SyncStatus(cl.ctx)
	cl.require.NoError(err, "fetch L2 sync status")

	return &utils.LocalGameInputs{
		L1Head:           syncStatus.CurrentL1.Hash,
		L2Head:           agreedOutput.BlockRef.Hash,
		L2OutputRoot:     common.Hash(agreedOutput.OutputRoot),
		L2Claim:          common.Hash(claimedOutput.OutputRoot),
		L2SequenceNumber: new(big.Int).SetUint64(claimBlock),
	}
}
