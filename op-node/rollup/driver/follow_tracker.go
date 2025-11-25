package driver

import (
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L2RPCFollowTracker struct {
	*sources.L2Client
}

func NewL2RPCFollowTracker(client *sources.L2Client) *L2RPCFollowTracker {
	return &L2RPCFollowTracker{L2Client: client}
}
