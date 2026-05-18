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
	// pointer across RestartWithFreshDataDir.
	InteropActivity() *interop.Interop

	// RestartWithFreshDataDir stops the supernode, deletes its on-disk
	// data directory, and starts a fresh supernode against the same chain
	// containers, virtual nodes, and externally-visible RPC address.
	RestartWithFreshDataDir() error

	// StopForExternalWipe stops the supernode without touching its data
	// directory, so tests can wipe sibling components (e.g. EL data) before
	// bringing the supernode back up via StartWithFreshDataDir.
	StopForExternalWipe() error

	// StartWithFreshDataDir wipes the supernode's data directory and starts
	// a fresh supernode against the same chain containers, virtual nodes,
	// and externally-visible RPC address. Pairs with StopForExternalWipe.
	StartWithFreshDataDir() error
}
