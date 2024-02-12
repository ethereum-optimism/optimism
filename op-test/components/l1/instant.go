package l1

import test "github.com/ethereum-optimism/optimism/op-test"

type InstantBackend struct {
	T test.Testing
}

func (i *InstantBackend) RequestL1(name Name, option ...Option) L1 {
	//TODO implement me
	panic("implement me")
}
