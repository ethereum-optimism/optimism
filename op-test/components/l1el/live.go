package l1el

import (
	test "github.com/ethereum-optimism/optimism/op-test"

	"github.com/ethereum-optimism/optimism/op-test/components/l1"
)

type LiveConfig struct {
	// TODO list of L1 EL nodes with endpoints etc.
}

type LiveBackend struct {
	T  test.Testing
	L1 l1.L1
}

var _ Backend = (*LiveBackend)(nil)

func (l *LiveBackend) RequestL1EL(name Name, opts ...Option) L1EL {
	req := RequestFromOpts(l.T, opts)
	// TODO check if online, check if properties match / or abort the test if there is no configured node

	// e.g. check block-building setting on the node we selected
	_ = req.BlockBuilding

	// TODO return bindings around Live L1 node
	return nil
}
