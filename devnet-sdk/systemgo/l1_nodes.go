package systemgo

import (
	"path/filepath"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type L1ELNode struct {
	userRPC  string
	l1Geth   *geth.GethInstance
	blobPath string
}

type L1CLNode struct {
	beaconHTTPAddr string
	beacon         *fakebeacon.FakeBeacon
}

func WithL1Nodes(l1ELID system2.L1ELNodeID, l1CLID system2.L1CLNodeID) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)

		l1NetID := setup.System.L1NetworkID(l1ELID.ChainID)
		l1Net, ok := orch.l1Nets.Get(l1NetID)
		setup.Require.True(ok, "L1 network must exist")

		sysL1Net := setup.System.L1Network(l1NetID).(system2.ExtensibleL1Network)

		blockTimeL1 := l1Net.blockTime
		l1FinalizedDistance := uint64(3)
		l1Clock := clock.SystemClock
		if orch.timeTravelClock != nil {
			l1Clock = orch.timeTravelClock
		}

		blobPath := orch.t.TempDir()

		clLog := setup.Log.New("id", l1CLID)
		bcn := fakebeacon.NewBeacon(clLog, e2eutils.NewBlobStore(), l1Net.genesis.Timestamp, blockTimeL1)
		orch.t.Cleanup(func() {
			_ = bcn.Close()
		})
		setup.Require.NoError(bcn.Start("127.0.0.1:0"))
		beaconApiAddr := bcn.BeaconAddr()
		setup.Require.NotEmpty(beaconApiAddr, "beacon API listener must be up")

		l1Geth, err := geth.InitL1(
			blockTimeL1,
			l1FinalizedDistance,
			l1Net.genesis,
			l1Clock,
			filepath.Join(blobPath, "l1_el"),
			bcn)
		setup.Require.NoError(err)
		setup.Require.NoError(l1Geth.Node.Start())
		orch.t.Cleanup(func() {
			clLog.Info("Closing L1 geth")
			_ = l1Geth.Close()
		})

		l1ELNode := &L1ELNode{
			userRPC:  l1Geth.Node.HTTPEndpoint(),
			l1Geth:   l1Geth,
			blobPath: blobPath,
		}
		setup.Require.True(orch.l1ELs.SetIfMissing(l1ELID, l1ELNode), "must not already exist")

		l1CLNode := &L1CLNode{
			beaconHTTPAddr: beaconApiAddr,
			beacon:         bcn,
		}
		setup.Require.True(orch.l1CLs.SetIfMissing(l1CLID, l1CLNode), "must not already exist")

		elClient, err := client.NewRPC(setup.Ctx, setup.Log, l1ELNode.userRPC, client.WithLazyDial())
		setup.Require.NoError(err)

		sysL1Net.AddL1ELNode(
			system2.NewL1ELNode(system2.L1ELNodeConfig{
				ID: l1ELID,
				ELNodeConfig: system2.ELNodeConfig{
					CommonConfig: setup.CommonConfig(),
					Client:       elClient,
					ChainID:      l1ELID.ChainID,
				},
			}),
		)

		beaconCl := client.NewBasicHTTPClient(bcn.BeaconAddr(), clLog)
		sysL1Net.AddL1CLNode(
			system2.NewL1CLNode(system2.L1CLNodeConfig{
				CommonConfig: setup.CommonConfig(),
				ID:           l1CLID,
				Client:       beaconCl,
			}),
		)
	}
}
