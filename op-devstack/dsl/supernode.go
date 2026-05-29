package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	"github.com/ethereum/go-ethereum/common"
)

// Supernode wraps a stack.Supernode interface for DSL operations
type Supernode struct {
	commonImpl
	inner       stack.Supernode
	testControl stack.SupernodeTestControl
	managedVNs  []*L2CLNode
}

// ManageVN registers a VN-proxy L2CLNode as hosted by this supernode so that
// Start() can re-establish its DSL-managed peer connections after a Stop/Start
// cycle. Initial peering is done by the sysgo runtime; this hook only matters
// across restarts driven through Supernode.Start.
func (s *Supernode) ManageVN(vn *L2CLNode) {
	s.managedVNs = append(s.managedVNs, vn)
}

// NewSupernode creates a new Supernode DSL wrapper
func NewSupernode(inner stack.Supernode) *Supernode {
	return &Supernode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

// NewSupernodeWithTestControl creates a new Supernode DSL wrapper with test control support.
// The testControl parameter can be nil if no test control is needed.
func NewSupernodeWithTestControl(inner stack.Supernode, testControl stack.SupernodeTestControl) *Supernode {
	return &Supernode{
		commonImpl:  commonFromT(inner.T()),
		inner:       inner,
		testControl: testControl,
	}
}

func (s *Supernode) Name() string {
	return s.inner.Name()
}

func (s *Supernode) String() string {
	return s.inner.Name()
}

// Escape returns the underlying stack.Supernode
func (s *Supernode) Escape() stack.Supernode {
	return s.inner
}

// QueryAPI returns the supernode's query API
func (s *Supernode) QueryAPI() apis.SupernodeQueryAPI {
	return s.inner.QueryAPI()
}

// SuperRootAtTimestamp fetches the super-root at the given timestamp
func (s *Supernode) SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, timestamp)
	s.require.NoError(err, "failed to get super-root at timestamp %d", timestamp)
	return resp
}

// AssertSuperRootAtTimestamp asserts that the super-root at the given timestamp matches the expected root claim
func (s *Supernode) AssertSuperRootAtTimestamp(l2SequenceNumber uint64, rootClaim common.Hash) {
	resp := s.SuperRootAtTimestamp(l2SequenceNumber)
	s.require.NotNilf(resp.Data, "super root does not exist at time %d", l2SequenceNumber)
	superRoot := eth.SuperRoot(resp.Data.Super)
	s.require.Equal(superRoot[:], rootClaim[:])
}

// AwaitFullyProcessedL1 waits until the supernode has fully processed the given L1
// block number. SuperRootAtTimestamp's CurrentL1 names the block currently being
// processed (L1[<CurrentL1] is fully processed), so this returns once
// CurrentL1.Number > targetL1.
func (s *Supernode) AwaitFullyProcessedL1(targetL1 uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, uint64(time.Now().Unix()))
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		return resp.CurrentL1.Number > targetL1, nil
	})
	s.require.NoError(err, "supernode did not fully process L1 block %d in time", targetL1)
}

// AwaitValidatedTimestamp waits for the super-root at the given timestamp to be fully validated
func (s *Supernode) AwaitValidatedTimestamp(timestamp uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		return resp.Data != nil, nil
	})
	s.require.NoError(err, "super-root at timestamp %d was not validated in time", timestamp)
}

// AwaitFinalizationAdvanced reads the supernode's current finalized timestamp
// from supernode_syncStatus and waits until it strictly advances past that
// value, then returns the new finalized timestamp. The first call therefore
// guarantees finalization has progressed past genesis (since genesis is the
// initial finalized timestamp).
func (s *Supernode) AwaitFinalizationAdvanced() uint64 {
	ctx, cancel := context.WithTimeout(s.ctx, 5*DefaultTimeout)
	defer cancel()
	initial, err := s.inner.QueryAPI().SyncStatus(ctx)
	s.require.NoError(err, "failed to read initial supernode sync status")
	start := initial.FinalizedTimestamp
	var ts uint64
	err = wait.For(ctx, 1*time.Second, func() (bool, error) {
		status, err := s.inner.QueryAPI().SyncStatus(ctx)
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		if status.FinalizedTimestamp <= start {
			return false, nil
		}
		ts = status.FinalizedTimestamp
		return true, nil
	})
	s.require.NoError(err, "supernode finalized timestamp did not advance past %d in time", start)
	return ts
}

// SuperRootAt returns the validated super-root at the given timestamp,
// asserting that all expectedChainIDs are present. The timestamp must already
// be finalized; this method does not wait for finalization.
func (s *Supernode) SuperRootAt(timestamp uint64, expectedChainIDs ...eth.ChainID) eth.SuperRootAtTimestampResponse {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, timestamp)
	s.require.NoError(err, "failed to query super-root at timestamp %d", timestamp)
	s.require.NotNil(resp.Data, "supernode returned no super-root data at finalized timestamp %d", timestamp)
	for _, chainID := range expectedChainIDs {
		s.require.Contains(resp.ChainIDs, chainID, "supernode super-root at timestamp %d missing chain %s", timestamp, chainID)
	}
	return resp
}

// interopActivity returns the currently running interop activity, failing
// the test if test control is not wired up or the activity is not present.
// All methods below that exercise the interop activity route through this
// helper so that the nil-guard is written in exactly one place.
func (s *Supernode) interopActivity() *interop.Interop {
	s.require.NotNil(s.testControl, "operation requires test control; use NewSupernodeWithTestControl")
	ia := s.testControl.InteropActivity()
	s.require.NotNil(ia, "interop activity not present (supernode stopped or interop disabled)")
	return ia
}

// PauseInterop pauses the interop activity at the given timestamp.
// When the interop activity attempts to process this timestamp, it returns early.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) PauseInterop(ts uint64) {
	s.interopActivity().PauseAt(ts)
}

// ResumeInterop clears any pause on the interop activity, allowing normal processing.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) ResumeInterop() {
	s.interopActivity().Resume()
}

// RestartWithFreshDataDir stops the supernode, deletes its on-disk data
// directory in full, and starts a fresh supernode against the same chain
// containers, virtual nodes, and externally-visible RPC address.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) RestartWithFreshDataDir() {
	s.require.NotNil(s.testControl,
		"RestartWithFreshDataDir requires test control; use NewSupernodeWithTestControl")
	s.log.Info("restarting supernode with fresh data dir")
	err := s.testControl.RestartWithFreshDataDir()
	s.require.NoError(err, "failed to restart supernode with fresh data dir")
}

// Stop halts the supernode while preserving its data directory and RPC
// address. Requires NewSupernodeWithTestControl.
func (s *Supernode) Stop() {
	s.require.NotNil(s.testControl, "Stop requires test control; use NewSupernodeWithTestControl")
	s.log.Info("stopping supernode")
	s.testControl.Stop()
}

// Start brings the supernode back up after Stop. Requires NewSupernodeWithTestControl.
// After the supernode is up, any VNs registered via ManageVN have their
// DSL-managed peer connections re-established, mirroring L2CLNode.Start.
func (s *Supernode) Start() {
	s.require.NotNil(s.testControl, "Start requires test control; use NewSupernodeWithTestControl")
	s.log.Info("starting supernode")
	s.testControl.Start()
	for _, vn := range s.managedVNs {
		vn.restoreManagedPeers()
	}
}

// BackfillAttempts returns the number of log-backfill attempts since the
// running interop activity's most recent (re)start.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) BackfillAttempts() int32 {
	return s.interopActivity().BackfillAttempts()
}

// AwaitBackfillAttempts blocks until BackfillAttempts() >= minAttempts or the
// timeout elapses. Fails the test on timeout.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) AwaitBackfillAttempts(minAttempts int32) {
	ia := s.interopActivity()
	ctx, cancel := context.WithTimeout(s.ctx, 3*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		return ia.BackfillAttempts() >= minAttempts, nil
	})
	s.require.NoErrorf(err, "backfill did not reach %d attempts in time (got %d)",
		minAttempts, ia.BackfillAttempts())
}

// AwaitBackfillCompleted blocks until the interop activity finishes its
// log backfill phase, or the timeout elapses. Fails the test on timeout.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) AwaitBackfillCompleted() {
	ia := s.interopActivity()
	ctx, cancel := context.WithTimeout(s.ctx, 3*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		return ia.BackfillCompleted(), nil
	})
	s.require.NoError(err, "backfill did not complete in time")
}

// ActivationTimestamp returns the configured interop activation timestamp.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) ActivationTimestamp() uint64 {
	return s.interopActivity().ActivationTimestamp()
}

// VerificationStartTimestamp returns the L2 timestamp the current interop
// activity began verifying at. Returns 0 before cold-start init completes.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) VerificationStartTimestamp() uint64 {
	return s.interopActivity().VerificationStartTimestamp()
}

// AwaitVerificationStartsAt blocks until cold-start init completes, then
// asserts VerificationStartTimestamp equals expected.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) AwaitVerificationStartsAt(expected uint64) {
	ia := s.interopActivity()
	ctx, cancel := context.WithTimeout(s.ctx, 3*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		return ia.BackfillCompleted(), nil
	})
	s.require.NoError(err, "cold-start initialization did not complete in time")
	actual := ia.VerificationStartTimestamp()
	s.require.Equalf(expected, actual,
		"verificationStartTimestamp mismatch after cold-start init: expected %d, got %d",
		expected, actual)
	s.log.Info("verification start timestamp confirmed", "expected", expected, "actual", actual)
}

// AssertBackfillCovers verifies, for each supplied chain, that the interop
// logs DB contains blocks spanning from a first-seal at or near the expected
// T_lo all the way to a latest-seal at or near the safe tip. Specifically it
// asserts the three invariants log backfill must preserve:
//
//  1. firstSealed.Timestamp + blockTime > ActivationTimestamp
//     (the first seal is at most one block before activation; when activation
//     is not aligned to a block boundary, the block representing the chain
//     state as of activation is the correct pairing anchor and is sealed).
//  2. firstSealed.Timestamp <  FirstVerifiableTimestamp()
//     (the post-backfill handoff happens strictly after the backfilled range)
//  3. firstSealed.Timestamp <= max(ActivationTimestamp, latestSealed.Timestamp - depth)
//     + blockTime                         (backfill reached ~depth back,
//     or all the way to activation if the chain is younger than depth)
//
// This is the strongest test-side evidence that backfill actually populated
// the DB, rather than the supernode simply resuming off of existing disk state.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) AssertBackfillCovers(depth time.Duration, blockTime uint64, chains ...eth.ChainID) {
	s.require.Positive(len(chains), "AssertBackfillCovers requires at least one chain")
	ia := s.interopActivity()

	activation := ia.ActivationTimestamp()
	backfillHandoff := ia.FirstVerifiableTimestamp()
	depthSec := uint64(depth / time.Second)

	for _, chainID := range chains {
		first, err := ia.FirstSealedBlock(chainID)
		s.require.NoErrorf(err, "chain %s: first sealed block must be readable", chainID)
		latest, hasLatest, err := ia.LatestSealedBlock(chainID)
		s.require.NoErrorf(err, "chain %s: latest sealed block must be readable", chainID)
		s.require.Truef(hasLatest, "chain %s: logs DB must contain at least one sealed block after backfill", chainID)

		s.require.Greaterf(first.Timestamp+blockTime, activation,
			"chain %s: first seal ts %d must be within one block time (%d) of activation ts %d",
			chainID, first.Timestamp, blockTime, activation)

		s.require.Lessf(first.Timestamp, backfillHandoff,
			"chain %s: first seal ts %d must be < backfill handoff ts %d (backfill must hand off strictly after the sealed range)",
			chainID, first.Timestamp, backfillHandoff)

		expectedLowerBound := activation
		if latest.Timestamp > activation+depthSec {
			expectedLowerBound = latest.Timestamp - depthSec
		}
		s.require.LessOrEqualf(first.Timestamp, expectedLowerBound+blockTime,
			"chain %s: first seal ts %d should be within one block of expected lower bound %d (latest ts %d, depth %s)",
			chainID, first.Timestamp, expectedLowerBound, latest.Timestamp, depth)

		s.require.Greaterf(latest.Number, first.Number,
			"chain %s: backfill should produce multiple sealed blocks (first=%d, latest=%d)",
			chainID, first.Number, latest.Number)

		s.log.Info("backfill coverage verified",
			"chain", chainID,
			"first_num", first.Number, "first_ts", first.Timestamp,
			"latest_num", latest.Number, "latest_ts", latest.Timestamp,
			"activation", activation, "backfill_handoff", backfillHandoff,
			"depth_sec", depthSec)
	}
}

// EnsureInteropPaused pauses the interop activity and verifies it has stopped.
// It takes the local safe timestamps from two CL nodes, uses the maximum, then:
// 1. Pauses interop at localSafeTimestamp + pauseOffset
// 2. Awaits validation of localSafeTimestamp + pauseOffset - 1
// 3. Finds the first timestamp that is NOT verified (the actual pause point)
// Returns the first unverified timestamp (adjusted if pause came in late).
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) EnsureInteropPaused(clA, clB *L2CLNode, pauseOffset uint64) uint64 {
	ia := s.interopActivity()

	// Get the local safe of both chains from sync status
	statusA := clA.SyncStatus()
	statusB := clB.SyncStatus()

	// Use the maximum local safe timestamp between both chains
	localSafeTimestamp := max(statusA.LocalSafeL2.Time, statusB.LocalSafeL2.Time)

	s.log.Info("EnsureInteropPaused: initial sync status",
		"chainA_local_safe_num", statusA.LocalSafeL2.Number,
		"chainA_local_safe_ts", statusA.LocalSafeL2.Time,
		"chainB_local_safe_num", statusB.LocalSafeL2.Number,
		"chainB_local_safe_ts", statusB.LocalSafeL2.Time,
		"localSafeTimestamp", localSafeTimestamp,
	)

	pauseTimestamp := localSafeTimestamp + pauseOffset
	awaitTimestamp := pauseTimestamp - 1

	// Pause interop activity at the pause timestamp
	ia.PauseAt(pauseTimestamp)

	// Await interop validation of the timestamp before the pause
	s.AwaitValidatedTimestamp(awaitTimestamp)

	s.log.Info("EnsureInteropPaused: validation confirmed before pause", "timestamp", awaitTimestamp)

	// Find the first timestamp that is NOT verified.
	// If the pause came in late, some timestamps past pauseTimestamp may already be verified.
	// We scan forward to find where interop actually stopped.
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()

	for ts := pauseTimestamp; ts < pauseTimestamp+100; ts++ {
		resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, ts)
		if err != nil || resp.Data == nil {
			// Found the first unverified timestamp
			s.log.Info("EnsureInteropPaused: confirmed interop is paused",
				"intendedPauseTimestamp", pauseTimestamp,
				"actualPauseTimestamp", ts,
			)
			return ts
		}
		// This timestamp is verified, continue scanning
		s.log.Warn("EnsureInteropPaused: pause came in late, timestamp already verified",
			"timestamp", ts,
			"intendedPause", pauseTimestamp,
		)
	}

	s.t.Error("EnsureInteropPaused: failed to find unverified timestamp within 100 timestamps")
	s.t.FailNow()
	return pauseTimestamp
}
