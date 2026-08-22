package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

type Supernode interface {
	Common
	ClientRPC() client.RPC
	QueryAPI() apis.SupernodeQueryAPI
	UserRPC() string
}

// SupernodeTestControl is the integration-test surface on a running supernode:
// what a test needs beyond the read-only query API in order to hold the node
// still, restart it, or inspect its verifier.
//
// Nothing on this interface is a handle on an in-process object. Test control
// over the interop verifier is expressed as apis.SupernodeInteropTestAPI, an
// RPC-shaped surface, so a supernode running in another process can back these
// presets by serving it — which the previous shape of this interface, returning
// a *interop.Interop pointer, made impossible.
type SupernodeTestControl interface {
	// InteropTestAPI returns the test-control surface for the supernode's
	// interop verification activity, or nil if the supernode is stopped or
	// interop is not configured.
	//
	// In-process implementations return a value bound to the current supernode
	// instance, so do not cache it across Stop/Start or
	// RestartWithFreshDataDir; re-fetch it per operation.
	InteropTestAPI() apis.SupernodeInteropTestAPI

	// RestartWithFreshDataDir stops the supernode, deletes its on-disk
	// data directory, and starts a fresh supernode against the same chain
	// containers, virtual nodes, and externally-visible RPC address.
	RestartWithFreshDataDir() error

	// Stop halts the supernode while preserving its data directory and RPC
	// address; Start brings it back up. Used by sync tests that need to halt
	// the verifier, mutate external state, and resume.
	Stop()
	Start()
}
