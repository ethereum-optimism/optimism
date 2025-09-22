package custom_gas_token

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
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
	const portalABI = `[
	  {"inputs":[],"name":"systemConfig","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`
	pa, err := abi.JSON(strings.NewReader(portalABI))
	if err != nil {
		t.Require().Fail("parse portal ABI: %v", err)
	}

	data, _ := pa.Pack("systemConfig")
	out, err := l1c.Call(ctx, ethereum.CallMsg{To: &portal, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("portal.systemConfig() call failed: %v", err)
	}
	vals, err := pa.Unpack("systemConfig", out)
	if err != nil {
		t.Require().Fail("unpack portal.systemConfig() failed: %v", err)
	}
	sysCfg := vals[0].(common.Address)
	if sysCfg == (common.Address{}) {
		t.Require().Fail("portal.systemConfig() returned zero address")
	}
}
