package l1

import test "github.com/ethereum-optimism/optimism/op-test"

type ManagedBackend struct {
	T test.Testing
}

func (m *ManagedBackend) RequestL1(name Name, option ...Option) L1 {
	//TODO implement me
	panic("implement me")
}

var _ Backend = (*ManagedBackend)(nil)
