package safedb_parity

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestSafeHeadRecordsMatchAfterKonaCLResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystemWithExecutionClient(t, true, sysgo.MixedL2ELOpGeth)
	deriveTogether(t, sys, 3)
	recordedFloor := sys.Kona.SafeHeadDBFloor()

	sys.Kona.ManagePeer(sys.OpNode)
	sys.Kona.Stop()
	sys.KonaEL.Stop()
	sys.OpNode.Advanced(safety.LocalSafe, 3, 60)

	sys.KonaEL.Start()
	sys.KonaEL.WaitForOnline()
	t.Require().Zero(sys.KonaEL.BlockRefByLabel(eth.Unsafe).Number, "CL resync must begin with an EL reset to genesis")
	sys.Kona.Start()
	sys.KonaEL.PeerWith(sys.OpNodeEL)
	sys.Kona.InSync(sys.OpNode, safety.LocalSafe, 60)
	deriveTogether(t, sys, 1)

	sys.Kona.VerifySafeHeadDatabaseMatches(sys.OpNode, dsl.WithMinRequiredL2Block(recordedFloor))
	sys.OpNode.VerifySafeHeadDatabaseMatches(sys.Kona)
	verifySuperRootAnswersMatch(t, sys)
}
