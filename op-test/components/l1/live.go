package l1

import test "github.com/ethereum-optimism/optimism/op-test"

type LiveBackend struct {
	T test.Testing
}

var _ Backend = (*LiveBackend)(nil)

func (l *LiveBackend) RequestL1(name Name, opts ...Option) L1 {
	//TODO implement me
	panic("implement me")
}
