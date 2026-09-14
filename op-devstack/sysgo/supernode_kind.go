package sysgo

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// devstackSupernodeKindEnv selects which implementation of the OP Stack supernode the
// multi-chain / interop presets run as their shared consensus layer.
//
// This is the multi-chain counterpart to DEVSTACK_L2CL_KIND (see devstackL2CLKind): that
// variable picks the per-chain CL of a single-chain preset — op-node or the Rust
// kona-node — and the multi-chain presets have no such node to swap, because their CL is
// a supernode hosting every chain at once behind one RPC with per-chain routes. Selecting
// the supernode implementation therefore needs its own variable rather than an extra
// DEVSTACK_L2CL_KIND value: a run can legitimately want kona-node for the single-chain
// presets and lokahi for the multi-chain ones, which is exactly what the acceptance CI
// variant asks for.
const devstackSupernodeKindEnv = "DEVSTACK_SUPERNODE_KIND"

// SupernodeKind names an implementation of the OP Stack supernode.
type SupernodeKind string

const (
	// SupernodeOpSupernode is the Go op-supernode, run in-process by the devstack. This is
	// the default, so an unset environment behaves exactly as it did before this switch
	// existed.
	SupernodeOpSupernode SupernodeKind = "op-supernode"
	// SupernodeLokahi is lokahi, the Rust supernode in rust/lokahi, run as a child process.
	SupernodeLokahi SupernodeKind = "lokahi"
)

// ResolveSupernodeKind returns the supernode implementation requested via
// DEVSTACK_SUPERNODE_KIND, defaulting to the in-process Go op-supernode when the variable
// is unset. An unrecognized value fails the test rather than being ignored: silently
// falling back to op-supernode would report a green run for an implementation that never
// started.
func ResolveSupernodeKind(t devtest.T) SupernodeKind {
	kind, err := supernodeKindFromEnv(os.Getenv(devstackSupernodeKindEnv))
	t.Require().NoError(err, "invalid supernode kind environment")
	return kind
}

// supernodeKindFromEnv holds the env-independent decision behind ResolveSupernodeKind so
// it can be exercised directly, including the rejection path.
func supernodeKindFromEnv(raw string) (SupernodeKind, error) {
	switch SupernodeKind(raw) {
	case "", SupernodeOpSupernode:
		return SupernodeOpSupernode, nil
	case SupernodeLokahi:
		return SupernodeLokahi, nil
	default:
		return "", fmt.Errorf("unknown %s %q: expected %q or %q",
			devstackSupernodeKindEnv, raw, SupernodeOpSupernode, SupernodeLokahi)
	}
}
