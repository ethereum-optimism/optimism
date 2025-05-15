package dsl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// L2CLNode wraps a stack.L2CLNode interface for DSL operations
type L2CLNode struct {
	commonImpl
	inner   stack.L2CLNode
	control stack.ControlPlane
	chainID eth.ChainID
}

// NewL2CLNode creates a new L2CLNode DSL wrapper
func NewL2CLNode(inner stack.L2CLNode, control stack.ControlPlane, chainID eth.ChainID) *L2CLNode {
	return &L2CLNode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		control:    control,
		chainID:    chainID,
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
	return cl.chainID
}

// Advanced returns a lambda that checks the L2CL chain head with given safety level advanced more than delta block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) Advanced(lvl types.SafetyLevel, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(lvl)
		target := initial.Number + delta
		cl.log.Info("expecting chain to advance", "id", cl.inner.ID(), "chain", cl.chainID, "label", lvl, "delta", delta)
		return cl.Reached(lvl, target, attempts)()
	}
}

// Reached returns a lambda that checks the L2CL chain head with given safety level reaches the target block number
// Composable with other lambdas to wait in parallel
func (cl *L2CLNode) Reached(lvl types.SafetyLevel, target uint64, attempts int) CheckFunc {
	return func() error {
		cl.log.Info("expecting chain to reach", "id", cl.inner.ID(), "chain", cl.chainID, "label", lvl, "target", target)
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := cl.HeadBlockRef(lvl)
				if head.Number >= target {
					cl.log.Info("chain advanced", "id", cl.inner.ID(), "chain", cl.chainID, "label", lvl, "target", target)
					return nil
				}
				cl.log.Info("Chain sync status", "id", cl.inner.ID(), "chain", cl.chainID, "label", lvl, "target", target, "current", head.Number)
				return fmt.Errorf("expected head to advance: %s", lvl)
			})
	}
}

func (cl *L2CLNode) PeerInfo() *apis.PeerInfo {
	peerInfo, err := cl.inner.P2PAPI().Self(cl.ctx)
	cl.require.NoError(err, "failed to get peer info")
	return peerInfo
}

func (cl *L2CLNode) Peers() *apis.PeerDump {
	peerDump, err := cl.inner.P2PAPI().Peers(cl.ctx, true)
	cl.require.NoError(err, "failed to get peers")
	return peerDump
}

func (cl *L2CLNode) DisconnectPeer(peer *L2CLNode) {
	peerInfo := peer.PeerInfo()
	err := cl.inner.P2PAPI().DisconnectPeer(cl.ctx, peerInfo.PeerID)
	cl.require.NoError(err, "failed to disconnect peer")
}
