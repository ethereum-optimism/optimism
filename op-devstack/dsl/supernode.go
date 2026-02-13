package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func (s *Supernode) ID() stack.SupernodeID {
	return s.inner.ID()
}

func (s *Supernode) String() string {
	return s.inner.ID().String()
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
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
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

// PauseInterop pauses the interop activity at the given timestamp.
// When the interop activity attempts to process this timestamp, it returns early.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) PauseInterop(ts uint64) {
	s.require.NotNil(s.testControl, "PauseInterop requires test control; use NewSupernodeWithTestControl")
	s.testControl.PauseInteropActivity(ts)
}

// ResumeInterop clears any pause on the interop activity, allowing normal processing.
// This function is for integration test control only.
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) ResumeInterop() {
	s.require.NotNil(s.testControl, "ResumeInterop requires test control; use NewSupernodeWithTestControl")
	s.testControl.ResumeInteropActivity()
}

// EnsureInteropPaused pauses the interop activity and verifies it has stopped.
// It takes the local safe timestamps from two CL nodes, uses the maximum, then:
// 1. Pauses interop at localSafeTimestamp + pauseOffset
// 2. Awaits validation of localSafeTimestamp + pauseOffset - 1
// 3. Finds the first timestamp that is NOT verified (the actual pause point)
// Returns the first unverified timestamp (adjusted if pause came in late).
// Requires the Supernode to be created with NewSupernodeWithTestControl.
func (s *Supernode) EnsureInteropPaused(clA, clB *L2CLNode, pauseOffset uint64) uint64 {
	s.require.NotNil(s.testControl, "EnsureInteropPaused requires test control; use NewSupernodeWithTestControl")

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
	s.testControl.PauseInteropActivity(pauseTimestamp)

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
