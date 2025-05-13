package dsl

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
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
	syncStatus, err := cl.Escape().RollupAPI().SyncStatus(cl.ctx)
	cl.require.NoError(err, "Expected to get sync status")

	return syncStatus.SafeL2
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

func (cl *L2CLNode) HeadBlockRef(label string) eth.BlockRef {
	syncStatus := cl.SyncStatus()
	targetWrap := reflect.ValueOf(syncStatus).Elem().FieldByName(label)
	cl.require.True(targetWrap.IsValid(), "invalid label")
	target := targetWrap.Interface()
	L2Block, ok := target.(eth.L2BlockRef)
	if ok {
		return L2Block.BlockRef()
	}
	L1Block, ok := target.(eth.L1BlockRef)
	cl.require.True(ok, "invalid type")
	return L1Block
}

func (cl *L2CLNode) ChainID() eth.ChainID {
	return cl.chainID
}

func (cl *L2CLNode) Advance(label string, delta uint64, attempts int) CheckFunc {
	return func() error {
		initial := cl.HeadBlockRef(label)
		target := initial.Number + delta
		cl.log.Info("expecting chain to advance", "id", cl.inner.ID(), "chain", cl.chainID, "label", label, "delta", delta)
		return cl.Reach(label, target, attempts)()
	}
}

func (cl *L2CLNode) Reach(label string, target uint64, attempts int) CheckFunc {
	return func() error {
		cl.log.Info("expecting chain to reach", "id", cl.inner.ID(), "chain", cl.chainID, "label", label, "target", target)
		return retry.Do0(cl.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := cl.HeadBlockRef(label)
				if head.Number >= target {
					cl.log.Info("chain advanced", "id", cl.inner.ID(), "chain", cl.chainID, "label", label, "target", target)
					return nil
				}
				cl.log.Info("Chain sync status", "id", cl.inner.ID(), "chain", cl.chainID, "label", label, "target", target, "current", head.Number)
				return fmt.Errorf("expected head to advance: %s", label)
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
