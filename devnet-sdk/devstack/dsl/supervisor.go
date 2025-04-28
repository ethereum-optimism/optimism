package dsl

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/status"
)

type Supervisor struct {
	commonImpl
	inner stack.Supervisor
}

func NewSupervisor(inner stack.Supervisor) *Supervisor {
	return &Supervisor{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *Supervisor) String() string {
	return s.inner.ID().String()
}

func (s *Supervisor) Escape() stack.Supervisor {
	return s.inner
}

type VerifySyncStatusConfig struct {
	AllUnsafeHeadsAdvance uint64
}

// WithAllLocalUnsafeHeadsAdvancedBy verifies that the local unsafe head of every chain advances by at least the
// specified number of blocks compared to the value when VerifySyncStatus is called.
func WithAllLocalUnsafeHeadsAdvancedBy(blocks uint64) func(cfg *VerifySyncStatusConfig) {
	return func(cfg *VerifySyncStatusConfig) {
		cfg.AllUnsafeHeadsAdvance = blocks
	}
}

// VerifySyncStatus performs assertions based on the supervisor's SyncStatus endpoint.
func (s *Supervisor) VerifySyncStatus(opts ...func(config *VerifySyncStatusConfig)) {
	cfg := applyOpts(VerifySyncStatusConfig{}, opts...)
	initial := s.fetchSyncStatus()
	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		status := s.fetchSyncStatus()
		s.require.Equalf(len(initial.Chains), len(status.Chains), "Expected %d chains in status but got %d", len(initial.Chains), len(status.Chains))
		for chID, chStatus := range status.Chains {
			chInitial := initial.Chains[chID]
			required := chInitial.LocalUnsafe.Number + cfg.AllUnsafeHeadsAdvance
			if chStatus.LocalUnsafe.Number < required {
				s.log.Info("Required sync status not reached. Chain local unsafe has not advanced enough",
					"chain", chID, "initialUnsafe", chInitial.LocalUnsafe, "currentUnsafe", chStatus.LocalUnsafe, "minRequired", required)
				return false, nil
			}
		}
		return true, nil
	})
	s.require.NoError(err, "Expected sync status not found")
}

func (s *Supervisor) fetchSyncStatus() eth.SupervisorSyncStatus {
	s.log.Debug("Fetching supervisor sync status")
	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()
	syncStatus, err := retry.Do[eth.SupervisorSyncStatus](ctx, 2, retry.Fixed(500*time.Millisecond), func() (eth.SupervisorSyncStatus, error) {
		syncStatus, err := s.inner.QueryAPI().SyncStatus(s.ctx)
		if errors.Is(err, status.ErrStatusTrackerNotReady) {
			s.log.Debug("Sync status not ready from supervisor")
		}
		return syncStatus, err
	})
	s.require.NoError(err, "Failed to fetch sync status")
	s.log.Info("Fetched supervisor sync status",
		"minSyncedL1", syncStatus.MinSyncedL1,
		"safeTimestamp", syncStatus.SafeTimestamp,
		"finalizedTimestamp", syncStatus.FinalizedTimestamp)
	return syncStatus
}

func (s *Supervisor) SafeBlockID(chainID eth.ChainID) eth.BlockID {
	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()
	syncStatus, err := retry.Do[eth.SupervisorSyncStatus](ctx, 2, retry.Fixed(500*time.Millisecond), func() (eth.SupervisorSyncStatus, error) {
		syncStatus, err := s.inner.QueryAPI().SyncStatus(s.ctx)
		if errors.Is(err, status.ErrStatusTrackerNotReady) {
			s.log.Debug("Sync status not ready from supervisor")
		}
		return syncStatus, err
	})
	s.require.NoError(err, "Failed to fetch sync status")

	return syncStatus.Chains[chainID].Safe
}
