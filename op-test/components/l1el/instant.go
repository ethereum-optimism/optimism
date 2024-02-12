package l1el

import (
	test "github.com/ethereum-optimism/optimism/op-test"

	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type InstantBackend struct {
	T  test.Testing
	L1 l1.L1

	// TODO map of current in-memory instantiated L1 nodes
}

var _ Backend = (*InstantBackend)(nil)

func (l *InstantBackend) RequestL1EL(name Name, opts ...Option) L1EL {
	req := RequestFromOpts(l.T, opts)
	// TODO check if online, check if properties match / or instantiate a new one

	// e.g. check block-building setting on the node we selected
	_ = req.BlockBuilding

	// TODO return bindings around the instant L1 node
	return nil
}
