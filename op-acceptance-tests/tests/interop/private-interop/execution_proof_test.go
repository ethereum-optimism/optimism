package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func executionMockOption(t *testing.T) sysgo.PrivateInteropOption {
	t.Helper()
	command, err := (rustbin.Spec{SrcDir: "rust", Package: "kona-sp1-super-range-executor", Binary: "kona-sp1-private-projection-executor"}).EnsureExists(t.Context(), testlog.Logger(t, log.LevelInfo))
	require.NoError(t, err)
	return sysgo.WithPrivateInteropExecutionMock(command)
}

// Publication must run private EVM execution, not just hash operator-supplied receipts.
func TestPrivatePublicationExecutesWitnessBeforeAdmission(gt *testing.T) {
	option := executionMockOption(gt)
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithPrivateInteropChain(option))
	t.Require().Equal(projection.ExecutionMock, sys.PrivateInterop.RenderingRollupConfig.PrivateProjection.Verifier)
	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	bridge := dsl.NewStandardBridge(t, sys.L2B, sys.L1EL)
	bridge.Deposit(eth.OneTenthEther, alice)
	privateAlice := alice.AsEL(sys.L2ELB)
	privateAlice.VerifyBalanceExact(eth.OneTenthEther)
	target := sys.FunderB.NewFundedEOA(eth.OneEther)
	privateAlice.Transfer(target.Address(), eth.OneHundredthEther)
	head := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.PrivateInterop.Invariant.RequireRenderingReached(head.Number, 4*time.Minute)
	sys.PrivateInterop.Invariant.RequirePrivateSafeReached(head.Number, 4*time.Minute)
	alice.AsEL(sys.L2BSupernodeEL).VerifyBalanceExact(eth.ZeroWei)
}

func TestPrivateExecutionProofRecoversAfterInvalidation(gt *testing.T) {
	testPrivateInvalidMessageRecovery(gt, false, true, executionMockOption(gt))
}

func TestPrivateExecutionProofMessagesBothDirections(gt *testing.T) {
	testPrivateInteropMessengerBothDirections(gt, executionMockOption(gt))
}
func TestPrivateExecutionProofResumesAfterWindowExpiry(gt *testing.T) {
	testPrivateOutageDoesNotBlockPublicProgress(gt, executionMockOption(gt))
}
