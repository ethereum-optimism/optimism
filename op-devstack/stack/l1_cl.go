package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// L1CLNode is a L1 ethereum consensus-layer node, aka Beacon node.
// This node may not be a full beacon node, and instead run a mock L1 consensus node.
type L1CLNode interface {
	Common
	ID() ComponentID

	BeaconClient() apis.BeaconClient
}
