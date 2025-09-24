package custom_gas_token

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
)

// TestCGT_L1PortalIntrospection checks that the L1 OptimismPortal exposes
// a valid SystemConfig address via its systemConfig() view.
func TestCGT_L1PortalIntrospection(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Skip if this devnet is not CGT-enabled (uses your existing gate).
	ensureCGTOrSkip(t, sys)

	l1c := sys.L1EL.EthClient()
	portal := sys.L2Chain.DepositContractAddr()

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// Portal exposes systemConfig() -> address
	fnSystemConfig := w3.MustNewFunc("systemConfig()", "address")
	data, _ := fnSystemConfig.EncodeArgs()
	out, err := l1c.Call(ctx, ethereum.CallMsg{To: &portal, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("portal.systemConfig() call failed: %v", err)
	}
	var sysCfg common.Address
	t.Require().NoError(fnSystemConfig.DecodeReturns(out, &sysCfg))
	if sysCfg == (common.Address{}) {
		t.Require().Fail("portal.systemConfig() returned zero address")
	}
}
