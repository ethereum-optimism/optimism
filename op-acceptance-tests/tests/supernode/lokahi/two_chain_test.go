package lokahi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// TestLokahiTwoChains is the gate on the lokahi devstack component: one out-of-process
// supernode, two chains, and the three things a supernode has to get right that a
// single-chain node cannot get wrong.
//
//   - Both chains sync. Each chain is produced by its own op-node sequencer and batcher, and the
//     safe head is only ever set from L1 batch data, so a safe head advancing under lokahi is
//     lokahi deriving that chain. (lokahi shares the chain's L2 P2P mesh, which is what gets its
//     engine past EL sync and no more; see sysgo.joinL2P2P.)
//   - Each chain answers for itself. The two endpoints are different sockets on one process,
//     and each must report its own chain rather than the other's.
//   - One execution layer going down stalls only its chain. A supernode whose chains share a
//     fate would be strictly worse than N single-chain nodes.
func TestLokahiTwoChains(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := sysgo.NewTwoL2LokahiRuntime(t)

	require.Len(t, sys.Chains, 2, "two chains")
	chainA, chainB := sys.Chains[0], sys.Chains[1]
	require.NotEqual(t,
		chainA.Network.ChainID(), chainB.Network.ChainID(), "the chains must be different")

	statusA := newStatus(t, chainA.CL.UserRPC())
	statusB := newStatus(t, chainB.CL.UserRPC())

	// Each endpoint answers for its own chain. Asked of the node rather than assumed from the
	// port it was given, so a supernode that crossed its routes is caught here. The genesis
	// hash identifies the chain: it is in the answer, and it cannot coincide.
	require.Equal(t,
		chainA.Network.RollupConfig().Genesis.L2.Hash, statusA.genesisHash(), "chain A endpoint")
	require.Equal(t,
		chainB.Network.RollupConfig().Genesis.L2.Hash, statusB.genesisHash(), "chain B endpoint")

	// Both chains derive. The safe head is what proves it: gossip moves the unsafe head, but the
	// safe head is only ever set from batches read off L1.
	safeA, safeB := statusA.safe(), statusB.safe()
	t.Logger().Info("initial safe heads", "l2a", safeA, "l2b", safeB)
	t.Require().Eventually(func() bool {
		return statusA.safe() > safeA && statusB.safe() > safeB
	}, 3*time.Minute, 2*time.Second, "both chains must derive a safe head from L1")

	// One chain's execution layer goes away. lokahi holds a stopped engine open per chain, so
	// only the chain that lost its execution layer stops.
	chainA.VerifierEL.Stop()
	t.Cleanup(chainA.VerifierEL.Start)

	stalledA, movingB := statusA.safe(), statusB.safe()
	t.Require().Eventually(func() bool {
		return statusB.safe() >= movingB+2
	}, 3*time.Minute, 2*time.Second, "the other chain must keep deriving")
	// One more block is tolerated: a payload already handed to the engine can land before the
	// stop takes effect. Two chains' worth of progress is what must not happen.
	require.LessOrEqual(t, statusA.safe(), stalledA+1,
		"the chain whose execution layer is down must not keep deriving")
	require.True(t, sys.Lokahi.Running(), "the supernode must survive one execution layer going down")
}

// status reads one chain's sync status over its own RPC socket.
type status struct {
	t      devtest.T
	logger log.Logger
	rpc    client.RPC
}

func newStatus(t devtest.T, endpoint string) *status {
	logger := t.Logger().New("component", "lokahi-status", "endpoint", endpoint)
	rpc, err := client.NewRPC(t.Ctx(), logger, endpoint, client.WithLazyDial())
	t.Require().NoError(err, "dial the lokahi chain RPC %s", endpoint)
	t.Cleanup(rpc.Close)
	return &status{t: t, logger: logger, rpc: rpc}
}

func (s *status) get() eth.SyncStatus {
	var out eth.SyncStatus
	s.t.Require().NoError(
		s.rpc.CallContext(s.t.Ctx(), &out, "optimism_syncStatus"), "read the sync status")
	return out
}

// safe is the derived head: it advances only from L1 data, so it is what says the chain is
// being derived rather than merely gossiped.
func (s *status) safe() uint64 {
	out := s.get()
	s.logger.Debug("sync status", "unsafe", out.UnsafeL2.Number, "safe", out.SafeL2.Number)
	return out.SafeL2.Number
}

// genesisHash reads the L2 genesis hash out of the rollup config the endpoint serves, which
// is what says which chain answered.
func (s *status) genesisHash() common.Hash {
	var out struct {
		Genesis struct {
			L2 struct {
				Hash common.Hash `json:"hash"`
			} `json:"l2"`
		} `json:"genesis"`
	}
	s.t.Require().NoError(
		s.rpc.CallContext(s.t.Ctx(), &out, "optimism_rollupConfig"), "read the rollup config")
	return out.Genesis.L2.Hash
}
