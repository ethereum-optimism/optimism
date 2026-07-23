package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1ELNode interface {
	l1ELNode()
	UserRPC() string
	AuthRPC() string
}

type L1Geth struct {
	name     string
	chainID  eth.ChainID
	userRPC  string
	authRPC  string
	l1Geth   *geth.GethInstance
	blobPath string
	command  []string
}

func (*L1Geth) l1ELNode() {}

func (g *L1Geth) UserRPC() string {
	return g.userRPC
}

func (g *L1Geth) AuthRPC() string {
	return g.authRPC
}

// Command returns the subprocess invocation used by staged runtimes. It is
// empty for the legacy in-process L1 implementation.
func (g *L1Geth) Command() []string {
	return append([]string(nil), g.command...)
}

type L1CLNode struct {
	name           string
	chainID        eth.ChainID
	beaconHTTPAddr string
	beacon         *fakebeacon.FakeBeacon
	fakepos        *FakePoS
}

func (n *L1CLNode) BeaconHTTPAddr() string {
	return n.beaconHTTPAddr
}

func (n *L1CLNode) FakePoS() stack.Lifecycle {
	return n.fakepos
}
