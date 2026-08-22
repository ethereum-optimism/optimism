package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

// SharedSupernode is the multi-chain consensus layer of the shared-supernode and interop presets,
// as those presets use it.
//
// It exists so DEVSTACK_SUPERNODE_KIND can select an implementation. Until now
// MultiChainRuntime.Supernode was the concrete in-process *SuperNode, which the presets passed as
// both the frontend's backing and the test control, so there was nothing for a kind to switch:
// selecting lokahi could only be refused, never honoured. This is the same shape the per-chain
// l2_cl_kind switch has for op-node versus kona-node, one level up.
//
// The method set is not designed here — it is exactly what the presets already ask of a supernode,
// and each half of it comes from somewhere:
//
//   - stack.SupernodeTestControl is what dsl.NewSupernodeWithTestControl takes, and every method
//     on it is RPC-shaped rather than a handle on an in-process object, which is what makes an
//     out-of-process supernode able to satisfy it at all.
//   - stack.ControlledLifecycle is what newSupernodeFrontend takes, so a test can stop and start
//     the supernode through the frontend as well as through the DSL.
//   - QueryRPC is the endpoint behind the frontend's apis.SupernodeQueryAPI.
//
// Anything a preset needs from one implementation and not the other stays off this interface and
// is reached by asserting for it, so that adding a capability to one supernode does not silently
// become a requirement on the other.
type SharedSupernode interface {
	stack.ControlledLifecycle
	stack.SupernodeTestControl

	// QueryRPC is the endpoint serving the supernode query API — supernode_syncStatus and
	// superroot_atTimestamp.
	//
	// Named for the contract rather than for the socket. The Go op-supernode serves it on the
	// same address as its per-chain /<chainID> routes, and lokahi serves it on its process-wide
	// socket while each chain answers on its own, so "the supernode's URL" is not one thing
	// across the two. What a preset actually wants here is always the query API.
	QueryRPC() string
}
