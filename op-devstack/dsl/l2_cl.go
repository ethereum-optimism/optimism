package dsl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

// L2CLNode wraps a stack.L2CLNode interface for DSL operations
type L2CLNode struct {
	commonImpl
	inner   stack.L2CLNode
	control stack.ControlPlane
}

// NewL2CLNode creates a new L2CLNode DSL wrapper
func NewL2CLNode(inner stack.L2CLNode, control stack.ControlPlane) *L2CLNode {
	return &L2CLNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		control:    control,
	}
}

func (cl *L2CLNode) String() string {
	return cl.inner.ID().String()
}

// Escape returns the underlying stack.L2CLNode
func (cl *L2CLNode) Escape() stack.L2CLNode {
	return cl.inner
}

func (cl *L2CLNode) SafeL2BlockRef() eth.L2BlockRef {
	return cl.HeadBlockRef(types.CrossSafe)
}

func (cl *L2CLNode) Start() {
	cl.control.L2CLNodeState(cl.inner.ID(), stack.Start)
}

func (cl *L2CLNode) Stop() {
	cl.control.L2CLNodeState(cl.inner.ID(), stack.Stop)
}

func (cl *L2CLNode) StartSequencer() {
	unsafe := cl.HeadBlockRef(types.LocalUnsafe)
	cl.log.Info("Continue sequencing with consensus node (op-node)", "chain", cl.ChainID(), "unsafe", unsafe)

	err := cl.inner.RollupAPI().StartSequencer(cl.ctx, unsafe.Hash)
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

func (cl *L2CLNode) SyncStatus() *eth.SyncStatus {
	ctx, cancel := context.WithTimeout(cl.ctx, DefaultTimeout)
	defer cancel()
	syncStatus, err := cl.inner.RollupAPI().SyncStatus(ctx)
	cl.require.NoError(err)
	return syncStatus
}

// HeadBlockRef fetches L2CL sync status and returns block ref with given safety level
func (cl *L2CLNode) HeadBlockRef(lvl types.SafetyLevel) eth.L2BlockRef {
	syncStatus := cl.SyncStatus()
	var blockRef eth.L2BlockRef
	switch lvl {
	case types.Finalized:
		blockRef = syncStatus.FinalizedL2
	case types.CrossSafe:
		blockRef = syncStatus.SafeL2
	case types.LocalSafe:
		blockRef = syncStatus.LocalSafeL2
	case types.CrossUnsafe:
		blockRef = syncStatus.CrossUnsafeL2
	case types.LocalUnsafe:
		blockRef = syncStatus.UnsafeL2
	default:
		cl.require.NoError(errors.New("invalid safety level"))
	}
	return blockRef
}

func (cl *L2CLNode) ChainID() eth.ChainID {
	return cl.inner.ID().ChainID()
}

// AdvancedFn returns a lambda that checks the L2CL chain head with given safety level advanced more than delta block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) AdvancedFn(lvl types.SafetyLevel, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		target := initial.Number + delta
		cl.log.Info("Expecting chain to advance", "id", cl.inner.ID(), "chain", cl.ChainID(), "label", lvl, "delta", delta)
		return cl.ReachedFn(lvl, target, attempts)()
	}
}

func (cl *L2CLNode) NotAdvancedFn(lvl types.SafetyLevel, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		logger := cl.log.With("id", cl.inner.ID(), "chain", cl.ChainID(), "label", lvl, "target", initial.Number)
		logger.Info("Expecting chain not to advance")
		for range attempts {
			time.Sleep(2 * time.Second)
			head := cl.HeadBlockRef(lvl)
			logger.Info("Chain sync status", "current", head.Number)
			if head.Hash == initial.Hash {
				continue
			}
			return fmt.Errorf("expected head not to advance: %s", lvl)
		}
		logger.Info("Chain not advanced")
		return nil
	}
}

// ReachedFn returns a lambda that checks the L2CL chain head with given safety level reaches the target block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) ReachedFn(lvl types.SafetyLevel, target uint64, attempts int) CheckFunc {
	return func() error {
		logger := cl.log.With("id", cl.inner.ID(), "chain", cl.ChainID(), "label", lvl, "target", target)
		logger.Info("Expecting chain to reach")
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := cl.HeadBlockRef(lvl)
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
func (cl *L2CLNode) ReachedRefFn(lvl types.SafetyLevel, target eth.BlockID, attempts int) CheckFunc {
	return func() error {
		err := cl.ReachedFn(lvl, target.Number, attempts)()
		if err != nil {
			return err
		}
		ethclient := cl.inner.ELs()[0].EthClient()
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

// RewindedFn returns a lambda that checks the L2CL chain head with given safety level rewinded more than the delta block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) RewindedFn(lvl types.SafetyLevel, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		cl.require.GreaterOrEqual(initial.Number, delta, "cannot rewind before genesis")
		target := initial.Number - delta
		logger := cl.log.With("id", cl.inner.ID(), "chain", cl.ChainID(), "label", lvl)
		logger.Info("Expecting chain to rewind", "target", target, "delta", delta)
		// check rewind more aggressively, in shorter interval
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 250 * time.Millisecond},
			func() error {
				head := cl.HeadBlockRef(lvl)
				if head.Number <= target {
					logger.Info("Chain rewinded", "target", target)
					return nil
				}
				logger.Info("Chain sync status", "target", target, "current", head.Number)
				return fmt.Errorf("expected head to rewind: %s", lvl)
			})
	}
}

func (cl *L2CLNode) Advanced(lvl types.SafetyLevel, delta uint64, attempts int) {
	cl.require.NoError(cl.AdvancedFn(lvl, delta, attempts)())
}

func (cl *L2CLNode) NotAdvanced(lvl types.SafetyLevel, attempts int) {
	cl.require.NoError(cl.NotAdvancedFn(lvl, attempts)())
}

func (cl *L2CLNode) Reached(lvl types.SafetyLevel, target uint64, attempts int) {
	cl.require.NoError(cl.ReachedFn(lvl, target, attempts)())
}

func (cl *L2CLNode) ReachedRef(lvl types.SafetyLevel, target eth.BlockID, attempts int) {
	cl.require.NoError(cl.ReachedRefFn(lvl, target, attempts)())
}

func (cl *L2CLNode) Rewinded(lvl types.SafetyLevel, delta uint64, attempts int) {
	cl.require.NoError(cl.RewindedFn(lvl, delta, attempts)())
}

// ChainSyncStatus satisfies that the L2CLNode can provide sync status per chain
func (cl *L2CLNode) ChainSyncStatus(chainID eth.ChainID, lvl types.SafetyLevel) eth.BlockID {
	cl.require.Equal(chainID, cl.inner.ID().ChainID(), "chain ID mismatch")
	return cl.HeadBlockRef(lvl).ID()
}

// LaggedFn returns a lambda that checks the L2CL chain head with given safety level is lagged with the reference chain sync status provider
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) LaggedFn(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int, allowMatch bool) CheckFunc {
	return LaggedFn(cl, refNode, cl.log, cl.ctx, lvl, cl.ChainID(), attempts, allowMatch)
}

// MatchedFn returns a lambda that checks the L2CLNode head with given safety level is matched with the refNode chain sync status provider
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) MatchedFn(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int) CheckFunc {
	return MatchedFn(cl, refNode, cl.log, cl.ctx, lvl, cl.ChainID(), attempts)
}

func (cl *L2CLNode) Lagged(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int, allowMatch bool) {
	cl.require.NoError(cl.LaggedFn(refNode, lvl, attempts, allowMatch)())
}

func (cl *L2CLNode) Matched(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int) {
	cl.require.NoError(cl.MatchedFn(refNode, lvl, attempts)())
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
	peerInfo := peer.PeerInfo()
	err := retry.Do0(cl.ctx, 3, retry.Exponential(), func() error {
		return cl.inner.P2PAPI().DisconnectPeer(cl.ctx, peerInfo.PeerID)
	})
	cl.require.NoError(err, "failed to disconnect peer")
}

func (cl *L2CLNode) ConnectPeer(peer *L2CLNode) {
	peerInfo := peer.PeerInfo()
	cl.require.NotZero(len(peerInfo.Addresses), "failed to get peer address")
	// graceful backoff for p2p connection, to avoid dial backoff or connection refused error
	strategy := &retry.ExponentialStrategy{Min: 10 * time.Second, Max: 30 * time.Second, MaxJitter: 250 * time.Millisecond}
	err := retry.Do0(cl.ctx, 5, strategy, func() error {
		return cl.inner.P2PAPI().ConnectPeer(cl.ctx, peerInfo.Addresses[0])
	})
	cl.require.NoError(err, "failed to connect peer")
}
