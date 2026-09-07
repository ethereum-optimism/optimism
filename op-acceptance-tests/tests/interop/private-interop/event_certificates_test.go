package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// A holder of an event attestation can settle and execute it while the private operator is offline.
func TestPrivateEventCertificateSurvivesSequencerOutage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(sysgo.WithLocalContractSources(), sysgo.WithSequencingWindow(10),
			sysgo.WithFinalizationPeriodSeconds(1), sysgo.WithProofMaturityDelaySeconds(2),
			sysgo.WithDisputeGameFinalityDelaySeconds(2)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()),
	)
	submitter := sys.FunderL1.NewFundedEOA(eth.OneEther)
	sender := sys.FunderB.NewFundedEOA(eth.OneEther)
	receiver := sys.FunderA.NewFundedEOA(eth.OneEther)
	certificates := dsl.NewEventCertificates(t, sys.L2B, sys.L2A, sys.L2BSupernodeEL, sys.L2ASupernodeEL, sys.L1EL, submitter)
	tx := txintent.NewIntent[*txintent.SendTrigger, *txintent.InteropOutput](sender.Plan())
	tx.Content.Set(&txintent.SendTrigger{Emitter: predeploys.L2toL2CrossDomainMessengerAddr,
		DestChainID: sys.L2A.ChainID(), Target: receiver.Address()})
	receipt, err := tx.PlannedTx.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	attestation := certificates.AttestMessage(sys.L2ELB, receipt)

	sys.L2BatcherB.Stop()
	sys.L2BCL.Stop()
	sys.L2ELB.Stop()
	withdrawal := certificates.Export(attestation)
	projection := dsl.NewL2Network(sys.L2B.Escape(), sys.L2BSupernodeEL, sys.L2BSupernodeCL, sys.L1EL, nil, nil)
	bridge := dsl.NewStandardBridge(t, projection, sys.L1EL)
	withdrawal.Prove(submitter)
	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()
	sys.AdvanceTime(max(bridge.WithdrawalDelay(), bridge.DisputeGameFinalityDelay()) + time.Second)
	withdrawal.Finalize(submitter)
	certificates.Relay(attestation)

	sys.L2ELB.Start()
	sys.L2BCL.Start()
	sys.L2BatcherB.Start()
	certificates.VerifyPrivateDepositReverted(sys.L2ELB, withdrawal, attestation)
}
