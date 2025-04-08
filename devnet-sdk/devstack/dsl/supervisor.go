package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Supervisor struct {
	common

	supervisor stack.Supervisor
}

func newSupervisor(c common, supervisor stack.Supervisor) *Supervisor {
	return &Supervisor{
		common:     c,
		supervisor: supervisor,
	}
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
	status, err := s.supervisor.QueryAPI().SyncStatus(s.ctx)
	s.require.NoError(err, "Failed to fetch sync status")
	s.log.Info("Fetched supervisor sync status",
		"minSyncedL1", status.MinSyncedL1,
		"safeTimestamp", status.SafeTimestamp,
		"finalizedTimestamp", status.FinalizedTimestamp)
	return status
}
