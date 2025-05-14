package dsl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/status"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type Supervisor struct {
	commonImpl
	inner   stack.Supervisor
	control stack.ControlPlane
}

func NewSupervisor(inner stack.Supervisor, control stack.ControlPlane) *Supervisor {
	return &Supervisor{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		control:    control,
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
	initial := s.FetchSyncStatus()
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		status := s.FetchSyncStatus()
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

func (s *Supervisor) AwaitMinL1(minL1 uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		return s.FetchSyncStatus().MinSyncedL1.Number >= minL1, nil
	})
	s.require.NoError(err, "Expected sync status not found")
}

func (s *Supervisor) FetchSyncStatus() eth.SupervisorSyncStatus {
	s.log.Debug("Fetching supervisor sync status")
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
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
	return s.L2HeadBlockID(chainID, types.CrossSafe)
}

// L2HeadBlockID fetches supervisor sync status and returns block id with given safety level
func (s *Supervisor) L2HeadBlockID(chainID eth.ChainID, lvl types.SafetyLevel) eth.BlockID {
	supervisorSyncStatus := s.FetchSyncStatus()
	supervisorChainSyncStatus, ok := supervisorSyncStatus.Chains[chainID]
	s.require.True(ok, "chain id not found in supervisor sync status")
	var blockID eth.BlockID
	switch lvl {
	case types.Finalized:
		blockID = supervisorChainSyncStatus.Finalized
	case types.CrossSafe:
		blockID = supervisorChainSyncStatus.CrossSafe
	case types.LocalSafe:
		blockID = supervisorChainSyncStatus.LocalSafe
	case types.CrossUnsafe:
		blockID = supervisorChainSyncStatus.CrossUnsafe
	case types.LocalUnsafe:
		blockID = supervisorChainSyncStatus.LocalUnsafe.ID()
	default:
		s.require.NoError(errors.New("invalid safety level"))
	}
	return blockID
}

// AdvancedL2Head checks the supervisor view of L2CL chain head with given safety level advanced more than delta block number
func (s *Supervisor) AdvancedL2Head(chainID eth.ChainID, delta uint64, lvl types.SafetyLevel, attempts int) {
	chInitial := s.L2HeadBlockID(chainID, lvl)
	target := chInitial.Number + delta
	err := retry.Do0(s.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
		func() error {
			chStatus := s.L2HeadBlockID(chainID, lvl)
			s.log.Info("Supervisor view",
				"chain", chainID, "label", lvl, "initial", chInitial.Number, "current", chStatus.Number, "target", target)
			if chStatus.Number >= target {
				s.log.Info("Supervisor view advanced", "chain", chainID, "label", lvl, "target", target)
				return nil
			}
			return fmt.Errorf("expected head to advance: %s", lvl)
		})
	s.require.NoError(err)
}

func (s *Supervisor) AdvancedUnsafeHead(chainID eth.ChainID, block uint64) {
	attempts := int(block + 3) // intentionally allow few more attempts for avoid flaking
	s.AdvancedL2Head(chainID, block, types.LocalUnsafe, attempts)
}

func (s *Supervisor) AdvancedSafeHead(chainID eth.ChainID, block uint64, attempts int) {
	s.AdvancedL2Head(chainID, block, types.CrossSafe, attempts)
}

func (s *Supervisor) Start() {
	s.control.SupervisorState(s.inner.ID(), stack.Start)
}

func (s *Supervisor) Stop() {
	s.control.SupervisorState(s.inner.ID(), stack.Stop)
}

func (s *Supervisor) AddManagedL2CL(cl *L2CLNode) {
	interopEndpoint, secret := cl.inner.InteropRPC()
	err := s.inner.AdminAPI().AddL2RPC(s.ctx, interopEndpoint, secret)
	s.require.NoError(err, "failed to connect L2CL to supervisor")
}
