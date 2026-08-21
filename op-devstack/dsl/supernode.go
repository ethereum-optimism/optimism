package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Supernode wraps a stack.Supernode interface for DSL operations
type Supernode struct {
	SuperRootQuerier
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
		SuperRootQuerier: SuperRootQuerier{commonImpl: commonFromT(inner.T()), api: inner.QueryAPI(), userRPC: inner.UserRPC()},
		inner:            inner,
	}
}

// NewSupernodeWithTestControl creates a new Supernode DSL wrapper with test control support.
// The testControl parameter can be nil if no test control is needed.
func NewSupernodeWithTestControl(inner stack.Supernode, testControl stack.SupernodeTestControl) *Supernode {
	return &Supernode{
		SuperRootQuerier: SuperRootQuerier{commonImpl: commonFromT(inner.T()), api: inner.QueryAPI(), userRPC: inner.UserRPC()},
		inner:            inner,
		testControl:      testControl,
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

func (s *Supernode) ClientRPC() opclient.RPC {
	return s.inner.ClientRPC()
}

func (s *Supernode) UserRPC() string {
	return s.inner.UserRPC()
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

// interopTestAPI returns the test-control surface for the currently running
// interop verifier, failing the test if test control is not wired up or the
// verifier is not present. All methods below that exercise the verifier route
// through this helper so the guard is written in exactly one place.
//
// The returned surface is deliberately not cached on the DSL wrapper: for an
// in-process supernode it is bound to the current instance, so a Stop/Start or
// RestartWithFreshDataDir must be followed by a fresh lookup.
func (s *Supernode) interopTestAPI() apis.SupernodeInteropTestAPI {
	s.require.NotNil(s.testControl, "operation requires test control; use NewSupernodeWithTestControl")
	api := s.testControl.InteropTestAPI()
	s.require.NotNil(api, "interop activity not present (supernode stopped or interop disabled)")
	return api
}

// interopStatus reads the verifier's test-visible progress, failing the test if
// it cannot be read.
func (s *Supernode) interopStatus(api apis.SupernodeInteropTestAPI) eth.SupernodeInteropStatus {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	status, err := api.InteropStatus(ctx)
	s.require.NoError(err, "failed to read interop status")
	return status
}

// PauseInterop pauses the interop verifier at the given timestamp.
// When the verifier attempts to process this timestamp, it returns early.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) PauseInterop(ts uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	s.require.NoError(s.interopTestAPI().PauseInterop(ctx, ts), "failed to pause interop at %d", ts)
}

// ResumeInterop clears any pause on the interop verifier, allowing normal processing.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) ResumeInterop() {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	s.require.NoError(s.interopTestAPI().ResumeInterop(ctx), "failed to resume interop")
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
// running interop verifier's most recent (re)start.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) BackfillAttempts() int32 {
	return s.interopStatus(s.interopTestAPI()).BackfillAttempts
}

// AwaitBackfillAttempts blocks until BackfillAttempts() >= minAttempts or the
// timeout elapses. Fails the test on timeout.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) AwaitBackfillAttempts(minAttempts int32) {
	api := s.interopTestAPI()
	ctx, cancel := context.WithTimeout(s.ctx, 3*DefaultTimeout)
	defer cancel()
	var attempts int32
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		status, err := api.InteropStatus(ctx)
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		attempts = status.BackfillAttempts
		return attempts >= minAttempts, nil
	})
	s.require.NoErrorf(err, "backfill did not reach %d attempts in time (got %d)",
		minAttempts, attempts)
}

// awaitBackfillCompleted blocks until the interop verifier finishes its
// cold-start initialization, then returns the status that reported it. Fails
// the test on timeout. Callers that need post-init fields read them off the
// returned status rather than issuing a second call, so that what they assert
// on is the same snapshot that told them init was done.
func (s *Supernode) awaitBackfillCompleted(api apis.SupernodeInteropTestAPI, what string) eth.SupernodeInteropStatus {
	ctx, cancel := context.WithTimeout(s.ctx, 3*DefaultTimeout)
	defer cancel()
	var status eth.SupernodeInteropStatus
	err := wait.For(ctx, 500*time.Millisecond, func() (bool, error) {
		got, err := api.InteropStatus(ctx)
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		status = got
		return status.BackfillCompleted, nil
	})
	s.require.NoErrorf(err, "%s did not complete in time", what)
	return status
}

// AwaitBackfillCompleted blocks until the interop verifier finishes its
// log backfill phase, or the timeout elapses. Fails the test on timeout.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) AwaitBackfillCompleted() {
	s.awaitBackfillCompleted(s.interopTestAPI(), "backfill")
}

// ActivationTimestamp returns the configured interop activation timestamp.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) ActivationTimestamp() uint64 {
	return s.interopStatus(s.interopTestAPI()).ActivationTimestamp
}

// VerificationStartTimestamp returns the L2 timestamp the current interop
// verifier began verifying at. Returns 0 before cold-start init completes.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) VerificationStartTimestamp() uint64 {
	return s.interopStatus(s.interopTestAPI()).VerificationStartTimestamp
}

// AwaitVerificationStartsAt blocks until cold-start init completes, then
// asserts VerificationStartTimestamp equals expected.
// Requires NewSupernodeWithTestControl.
func (s *Supernode) AwaitVerificationStartsAt(expected uint64) {
	status := s.awaitBackfillCompleted(s.interopTestAPI(), "cold-start initialization")
	actual := status.VerificationStartTimestamp
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
//  2. firstSealed.Timestamp <  FirstVerifiableTimestamp
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
	api := s.interopTestAPI()

	status := s.interopStatus(api)
	activation := status.ActivationTimestamp
	backfillHandoff := status.FirstVerifiableTimestamp
	depthSec := uint64(depth / time.Second)

	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()

	for _, chainID := range chains {
		sealed, err := api.InteropSealedBlocks(ctx, chainID)
		s.require.NoErrorf(err, "chain %s: sealed block range must be readable", chainID)
		s.require.Truef(sealed.HasBlocks,
			"chain %s: logs DB must contain at least one sealed block after backfill", chainID)
		first, latest := sealed.First, sealed.Latest

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

		s.require.Greaterf(latest.ID.Number, first.ID.Number,
			"chain %s: backfill should produce multiple sealed blocks (first=%d, latest=%d)",
			chainID, first.ID.Number, latest.ID.Number)

		s.log.Info("backfill coverage verified",
			"chain", chainID,
			"first_num", first.ID.Number, "first_ts", first.Timestamp,
			"latest_num", latest.ID.Number, "latest_ts", latest.Timestamp,
			"activation", activation, "backfill_handoff", backfillHandoff,
			"depth_sec", depthSec)
	}
}

// EnsureInteropPaused pauses the interop verifier and verifies it has stopped.
// It takes the local safe timestamps from two CL nodes, uses the maximum, then:
// 1. Pauses interop at localSafeTimestamp + pauseOffset
// 2. Awaits validation of localSafeTimestamp + pauseOffset - 1
// 3. Finds the first timestamp that is NOT verified (the actual pause point)
// Returns the first unverified timestamp (adjusted if pause came in late).
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) EnsureInteropPaused(clA, clB *L2CLNode, pauseOffset uint64) uint64 {
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

	// Pause the interop verifier at the pause timestamp
	s.PauseInterop(pauseTimestamp)

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
