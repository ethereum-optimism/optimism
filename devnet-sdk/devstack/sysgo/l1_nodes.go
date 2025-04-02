package sysgo

import (
	"path/filepath"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type L1ELNode struct {
	id       stack.L1ELNodeID
	userRPC  string
	l1Geth   *geth.GethInstance
	blobPath string
}

func (n *L1ELNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)

	frontend := shim.NewL1ELNode(shim.L1ELNodeConfig{
		ID: n.id,
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       rpcCl,
			ChainID:      n.id.ChainID,
		},
	})
	l1ID := system.L1NetworkID(n.id.ChainID)
	l1Net := system.L1Network(l1ID)
	l1Net.(stack.ExtensibleL1Network).AddL1ELNode(frontend)
}

type L1CLNode struct {
	id             stack.L1CLNodeID
	beaconHTTPAddr string
	beacon         *fakebeacon.FakeBeacon
}

func (n *L1CLNode) hydrate(system stack.ExtensibleSystem) {
	beaconCl := client.NewBasicHTTPClient(n.beaconHTTPAddr, system.Logger())
	frontend := shim.NewL1CLNode(shim.L1CLNodeConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           n.id,
		Client:       beaconCl,
	})
	l1ID := system.L1NetworkID(n.id.ChainID)
	l1Net := system.L1Network(l1ID)
	l1Net.(stack.ExtensibleL1Network).AddL1CLNode(frontend)
}

func WithL1Nodes(l1ELID stack.L1ELNodeID, l1CLID stack.L1CLNodeID) stack.Option {
	return func(o stack.Orchestrator) {
		orch := o.(*Orchestrator)
		require := o.P().Require()

		l1Net, ok := orch.l1Nets.Get(l1ELID.ChainID)
		require.True(ok, "L1 network must exist")

		blockTimeL1 := l1Net.blockTime
		l1FinalizedDistance := uint64(3)
		l1Clock := clock.SystemClock
		if orch.timeTravelClock != nil {
			l1Clock = orch.timeTravelClock
		}

		blobPath := orch.p.TempDir()

		logger := o.P().Logger().New("id", l1CLID)
		bcn := fakebeacon.NewBeacon(logger, e2eutils.NewBlobStore(), l1Net.genesis.Timestamp, blockTimeL1)
		orch.p.Cleanup(func() {
			_ = bcn.Close()
		})
		require.NoError(bcn.Start("127.0.0.1:0"))
		beaconApiAddr := bcn.BeaconAddr()
		require.NotEmpty(beaconApiAddr, "beacon API listener must be up")

		l1Geth, err := geth.InitL1(
			blockTimeL1,
			l1FinalizedDistance,
			l1Net.genesis,
			l1Clock,
			filepath.Join(blobPath, "l1_el"),
			bcn)
		require.NoError(err)
		require.NoError(l1Geth.Node.Start())
		orch.p.Cleanup(func() {
			logger.Info("Closing L1 geth")
			_ = l1Geth.Close()
		})

		l1ELNode := &L1ELNode{
			id:       l1ELID,
			userRPC:  l1Geth.Node.HTTPEndpoint(),
			l1Geth:   l1Geth,
			blobPath: blobPath,
		}
		require.True(orch.l1ELs.SetIfMissing(l1ELID, l1ELNode), "must not already exist")

		l1CLNode := &L1CLNode{
			id:             l1CLID,
			beaconHTTPAddr: beaconApiAddr,
			beacon:         bcn,
		}
		require.True(orch.l1CLs.SetIfMissing(l1CLID, l1CLNode), "must not already exist")
	}
}
