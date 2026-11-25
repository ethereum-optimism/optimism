package sysgo

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// devstackSupernodeKindEnv selects the interop verifier the supernode presets start.
//
// DEVSTACK_L2CL_KIND picks a chain's consensus node, and a supernode preset hosts every chain's
// node in op-supernode, so the verifier is a choice of its own: with lokahi, each chain runs its
// own kona-node, and the verifier runs as a separate lokahi process the nodes follow.
const devstackSupernodeKindEnv = "DEVSTACK_SUPERNODE_KIND"

// SupernodeKind names the interop verifier of the supernode presets.
type SupernodeKind string

const (
	// SupernodeOpSupernode is op-supernode, run in process, hosting every chain's op-node. It is
	// the default.
	SupernodeOpSupernode SupernodeKind = "op-supernode"
	// SupernodeLokahi is lokahi, run as a process of its own, with a kona-node per chain that
	// follows it.
	SupernodeLokahi SupernodeKind = "lokahi"
)

// ResolveSupernodeKind returns the verifier requested via DEVSTACK_SUPERNODE_KIND, op-supernode
// when the variable is unset, and fails the test on any other value before a node starts.
func ResolveSupernodeKind(t devtest.T) SupernodeKind {
	kind, err := supernodeKindFromEnv(os.Getenv(devstackSupernodeKindEnv))
	t.Require().NoError(err)
	return kind
}

func supernodeKindFromEnv(raw string) (SupernodeKind, error) {
	switch kind := SupernodeKind(raw); kind {
	case "", SupernodeOpSupernode:
		return SupernodeOpSupernode, nil
	case SupernodeLokahi:
		return kind, nil
	default:
		return "", fmt.Errorf("unsupported %s=%q, want %q or %q", devstackSupernodeKindEnv, raw,
			SupernodeOpSupernode, SupernodeLokahi)
	}
}
