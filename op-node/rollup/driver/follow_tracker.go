package driver

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2RPCFollowTracker struct {
	*sources.L2Client
	prevExternalSafe eth.L2BlockRef
}

func NewL2RPCFollowTracker(client *sources.L2Client) *L2RPCFollowTracker {
	return &L2RPCFollowTracker{L2Client: client}
}

func (ft *L2RPCFollowTracker) SetPrevExternalSafe(safe eth.L2BlockRef) {
	ft.prevExternalSafe = safe
}

func (ft *L2RPCFollowTracker) GetPrevExternalSafe() eth.L2BlockRef {
	return ft.prevExternalSafe
}
