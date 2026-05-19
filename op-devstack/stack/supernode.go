package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

type Supernode interface {
	Common
	QueryAPI() apis.SupernodeQueryAPI
}

// SupernodeTestControl is the integration-test surface on a running
// supernode. See op-supernode/supernode/activity/interop for the methods
// available on the InteropActivity pointer.
type SupernodeTestControl interface {
	// InteropActivity returns the current interop activity, or nil if the
	// supernode is stopped or interop is not configured. Do not cache the
	// pointer across StartWithFreshDataDir.
	InteropActivity() *interop.Interop

	// Stop stops the supernode without touching its data dir, leaving the
	// externally-visible RPC address in place so peer components can be
	// wiped between Stop and StartWithFreshDataDir.
	Stop()

	// StartWithFreshDataDir wipes the supernode's data dir and starts it
	// against the same VNs and RPC address. Pairs with Stop.
	StartWithFreshDataDir() error
}
