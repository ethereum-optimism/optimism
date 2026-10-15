package op_reth_opgeth_peering

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// highestEthVersion returns the highest eth protocol version among the capabilities a peer
// advertised.
func highestEthVersion(t devtest.T, caps []string) uint64 {
	version, err := sysgo.HighestEthVersion(caps)
	t.Require().NoError(err, "a peer must advertise an eth version")
	return version
}

// TestOpRethAndOpGethNegotiateTheHighestCommonEthVersion peers an op-reth and an op-geth follower
// as basic peers, with either client dialing, and requires both to report the same negotiated eth
// protocol version through admin_peers: the highest version both advertise. The advertised
// versions are logged, so a run records which version the pair settled on.
func TestOpRethAndOpGethNegotiateTheHighestCommonEthVersion(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, dialer := range followerKinds {
		t.Run(fmt.Sprintf("%s_dials_%s", dialer, otherKind(dialer)), func(t devtest.T) {
			t.Parallel()
			skipUnsupported(t)
			sys := newPairSystem(t, dialer)
			opReth := sys.follower(sysgo.MixedL2ELOpReth).EL
			opGeth := sys.follower(sysgo.MixedL2ELOpGeth).EL

			rethReports := opReth.PeerInfoOf(opGeth)
			gethReports := opGeth.PeerInfoOf(opReth)
			opRethAdvertises := highestEthVersion(t, gethReports.Caps)
			opGethAdvertises := highestEthVersion(t, rethReports.Caps)
			t.Logger().Info("Negotiated eth version",
				"op_reth_advertises", opRethAdvertises, "op_geth_advertises", opGethAdvertises,
				"op_reth_reports", rethReports.EthVersion, "op_geth_reports", gethReports.EthVersion,
				"op_reth_caps", gethReports.Caps, "op_geth_caps", rethReports.Caps)

			t.Require().Equal(rethReports.EthVersion, gethReports.EthVersion, "both clients must report the same negotiated eth version")
			t.Require().Equal(min(opRethAdvertises, opGethAdvertises), rethReports.EthVersion, "the pair must negotiate the highest eth version both advertise")
		})
	}
}
