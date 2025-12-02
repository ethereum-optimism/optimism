package sysgo

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type Supervisor interface {
	hydrate(system stack.ExtensibleSystem)
	stack.Lifecycle
	UserRPC() string
}

func WithSupervisor(supervisorID stack.SupervisorID, clusterID stack.ClusterID, l1ELID stack.L1ELNodeID) stack.Option[*Orchestrator] {
	switch os.Getenv("DEVSTACK_SUPERVISOR_KIND") {
	case "kona":
		return WithKonaSupervisor(supervisorID, clusterID, l1ELID)
	default:
		return WithOPSupervisor(supervisorID, clusterID, l1ELID)
	}
}

func WithManagedBySupervisor(l2CLID stack.L2CLNodeID, supervisorID stack.SupervisorID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		require.True(ok, "looking for L2 CL node to connect to supervisor")
		interopEndpoint, secret := l2CL.InteropRPC()

		s, ok := orch.supervisors.Get(supervisorID)
		require.True(ok, "looking for supervisor")

		ctx := orch.P().Ctx()
		supClient, err := dial.DialSupervisorClientWithTimeout(ctx, orch.P().Logger(), s.UserRPC(), client.WithLazyDial())
		orch.P().Require().NoError(err)

		err = retry.Do0(ctx, 10, retry.Exponential(), func() error {
			return supClient.AddL2RPC(ctx, interopEndpoint, secret)
		})
		require.NoError(err, "must connect CL node %s to supervisor %s", l2CLID, supervisorID)
	})
}
