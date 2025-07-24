package sysgo

import (
	"github.com/HashKeyChain/verse/op-devstack/devtest"
	"github.com/HashKeyChain/verse/op-e2e/e2eutils/geth"
)

type FakePoS struct {
	p       devtest.P
	fakepos *geth.FakePoS
}

func (f *FakePoS) Start() {
	f.p.Require().NoError(f.fakepos.Start(), "fakePoS failed to start")
}

func (f *FakePoS) Stop() {
	f.p.Require().NoError(f.fakepos.Stop(), "fakePoS failed to stop")
}
