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
	testControl stack.InteropTestControl
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
func NewSupernodeWithTestControl(inner stack.Supernode, testControl stack.InteropTestControl) *Supernode {
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

// RestartInterop stops the running interop activity, optionally wipes its
// on-disk logs DBs, and launches a fresh instance against the still-running
// supernode. The HTTP server, chain containers, virtual nodes, and all other
// activities keep running across the restart. Setting wipeLogsDBs=true forces
// the fresh activity to reconstruct its database via log backfill from the
// virtual nodes, making this the primary primitive for exercising backfill
// in tests.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) RestartInterop(wipeLogsDBs bool) {
	s.require.NotNil(s.testControl, "RestartInterop requires test control; use NewSupernodeWithTestControl")
	s.log.Info("restarting interop activity", "wipeLogsDBs", wipeLogsDBs)
	err := s.testControl.RestartInteropActivity(wipeLogsDBs)
	s.require.NoError(err, "failed to restart interop activity")
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
//     (the main loop's handoff happens strictly after the backfilled range)
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
	firstVerifiable := ia.FirstVerifiableTimestamp()
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

		s.require.Lessf(first.Timestamp, firstVerifiable,
			"chain %s: first seal ts %d must be < first-verifiable ts %d (backfill must hand off strictly before main loop)",
			chainID, first.Timestamp, firstVerifiable)

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
			"activation", activation, "first_verifiable", firstVerifiable,
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
