package custom_gas_token

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// TestCGT_PortalReceiveReverts asserts that sending ETH to the L1 OptimismPortal
// (receive() -> depositTransaction) reverts under CGT, preventing ETH from getting stuck.
func TestCGT_PortalReceiveReverts(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	ensureCGTOrSkip(t, sys)

	l1c := sys.L1EL.EthClient()
	portal := sys.L2Chain.DepositContractAddr()

	// Try to send 1 wei to the Portal (receive() -> depositTransaction); should revert in CGT mode.
	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()
	_, err := l1c.EstimateGas(ctx, ethereum.CallMsg{
		To:    &portal,
		Value: common.Big1,
	})
	if err == nil {
		t.Require().Fail("expected L1 Portal to revert on direct ETH send in CGT mode, but estimator returned no error")
	}
}
