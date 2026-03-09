package sysgo

import (
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/blobstore"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type L1ELNode interface {
	hydrator
	l1ELNode()
	UserRPC() string
	AuthRPC() string
}

type L1Geth struct {
	id       stack.ComponentID
	userRPC  string
	authRPC  string
	l1Geth   *geth.GethInstance
	blobPath string
}

func (*L1Geth) l1ELNode() {}

func (g *L1Geth) UserRPC() string {
	return g.userRPC
}

func (g *L1Geth) AuthRPC() string {
	return g.authRPC
}

func (n *L1Geth) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)

	frontend := shim.NewL1ELNode(shim.L1ELNodeConfig{
		ID: n.id,
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       rpcCl,
			ChainID:      n.id.ChainID(),
		},
	})
	l1Net := system.L1Network(stack.ByID[stack.L1Network](stack.NewL1NetworkID(n.id.ChainID())))
	l1Net.(stack.ExtensibleL1Network).AddL1ELNode(frontend)
}

type L1CLNode struct {
	id             stack.ComponentID
	beaconHTTPAddr string
	beacon         *fakebeacon.FakeBeacon
	fakepos        *FakePoS
}

func (n *L1CLNode) hydrate(system stack.ExtensibleSystem) {
	beaconCl := client.NewBasicHTTPClient(n.beaconHTTPAddr, system.Logger())
	frontend := shim.NewL1CLNode(shim.L1CLNodeConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           n.id,
		Client:       beaconCl,
	})
	l1Net := system.L1Network(stack.ByID[stack.L1Network](stack.NewL1NetworkID(n.id.ChainID())))
	l1Net.(stack.ExtensibleL1Network).AddL1CLNode(frontend)
}

const DevstackL1ELKindEnvVar = "DEVSTACK_L1EL_KIND"

func WithL1Nodes(l1ELID stack.ComponentID, l1CLID stack.ComponentID) stack.Option[*Orchestrator] {
	switch os.Getenv(DevstackL1ELKindEnvVar) {
	case "geth":
		return WithL1NodesSubprocess(l1ELID, l1CLID)
	default:
		return WithL1NodesInProcess(l1ELID, l1CLID)
	}
}

func WithL1NodesInProcess(l1ELID stack.ComponentID, l1CLID stack.ComponentID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		clP := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l1CLID))
		elP := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l1ELID))
		require := orch.P().Require()

		l1Net, ok := orch.GetL1Network(stack.NewL1NetworkID(l1ELID.ChainID()))
		require.True(ok, "L1 network must exist")

		blockTimeL1 := l1Net.blockTime
		l1FinalizedDistance := uint64(20)
		l1Clock := clock.SystemClock
		if orch.timeTravelClock != nil {
			l1Clock = orch.timeTravelClock
		}

		blobPath := clP.TempDir()

		clLogger := clP.Logger()
		bcn := fakebeacon.NewBeacon(clLogger, blobstore.New(), l1Net.genesis.Timestamp, blockTimeL1)
		clP.Cleanup(func() {
			_ = bcn.Close()
		})
		require.NoError(bcn.Start("127.0.0.1:0"))
		beaconApiAddr := bcn.BeaconAddr()
		require.NotEmpty(beaconApiAddr, "beacon API listener must be up")

		orch.writeDefaultJWT()

		elLogger := elP.Logger()
		l1Geth, fp, err := geth.InitL1(
			blockTimeL1,
			l1FinalizedDistance,
			l1Net.genesis,
			l1Clock,
			filepath.Join(blobPath, "l1_el"),
			bcn,
			geth.WithAuth(orch.jwtPath),
		)
		require.NoError(err)
		require.NoError(l1Geth.Node.Start())
		elP.Cleanup(func() {
			elLogger.Info("Closing L1 geth")
			_ = l1Geth.Close()
		})

		l1ELNode := &L1Geth{
			id:       l1ELID,
			userRPC:  l1Geth.Node.HTTPEndpoint(),
			authRPC:  l1Geth.Node.HTTPAuthEndpoint(),
			l1Geth:   l1Geth,
			blobPath: blobPath,
		}
		require.False(orch.registry.Has(l1ELID), "must not already exist")
		orch.registry.Register(l1ELID, l1ELNode)

		l1CLNode := &L1CLNode{
			id:             l1CLID,
			beaconHTTPAddr: beaconApiAddr,
			beacon:         bcn,
			fakepos:        &FakePoS{fakepos: fp, p: clP},
		}
		require.False(orch.registry.Has(l1CLID), "must not already exist")
		orch.registry.Register(l1CLID, l1CLNode)
	})
}

// WithExtL1Nodes initializes L1 EL and CL nodes that connect to external RPC endpoints
func WithExtL1Nodes(l1ELID stack.ComponentID, l1CLID stack.ComponentID, elRPCEndpoint string, clRPCEndpoint string) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		// Create L1 EL node with external RPC
		l1ELNode := &L1Geth{
			id:      l1ELID,
			userRPC: elRPCEndpoint,
		}
		require.False(orch.registry.Has(l1ELID), "must not already exist")
		orch.registry.Register(l1ELID, l1ELNode)

		// Create L1 CL node with external RPC
		l1CLNode := &L1CLNode{
			id:             l1CLID,
			beaconHTTPAddr: clRPCEndpoint,
		}
		require.False(orch.registry.Has(l1CLID), "must not already exist")
		orch.registry.Register(l1CLID, l1CLNode)
	})
}
