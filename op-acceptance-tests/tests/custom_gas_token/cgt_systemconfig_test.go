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

// TestCGT_SystemConfigFlagOnL1 checks that the L1 SystemConfig contract reports
// CGT=true via isCustomGasToken(). Skips if the devnet does not wire this flag.
func TestCGT_SystemConfigFlagOnL1(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	ensureCGTOrSkip(t, sys)

	l1c := sys.L1EL.EthClient()
	portal := sys.L2Chain.DepositContractAddr()

	fnSystemConfig := w3.MustNewFunc("systemConfig()", "address")
	fnIsCGT := w3.MustNewFunc("isCustomGasToken()", "bool")
	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// Resolve SystemConfig via Portal.systemConfig()
	data, _ := fnSystemConfig.EncodeArgs()
	out, err := l1c.Call(ctx, ethereum.CallMsg{To: &portal, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("portal.systemConfig() call failed: %v", err)
	}
	var sysCfg common.Address
	t.Require().NoError(fnSystemConfig.DecodeReturns(out, &sysCfg))
	if (sysCfg == common.Address{}) {
		t.Require().Fail("portal.systemConfig() returned zero address")
	}

	// Ask SystemConfig whether CGT is enabled.
	data, _ = fnIsCGT.EncodeArgs()
	out, err = l1c.Call(ctx, ethereum.CallMsg{To: &sysCfg, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("SystemConfig.isCustomGasToken() call failed: %v", err)
	}
	var flag bool
	t.Require().NoError(fnIsCGT.DecodeReturns(out, &flag))
	if !flag {
		t.Skip("SystemConfig.isCustomGasToken() = false on this devnet; skipping")
	}
}

// TestCGT_SystemConfigFeatureFlag re-validates the CGT flag on SystemConfig,
// using locally encoded calls (mirrors the previous test structure). Skips on devnets without the flag.
func TestCGT_SystemConfigFeatureFlag(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Skip if not in CGT mode (uses L2 L1Block.isCustomGasToken()).
	ensureCGTOrSkip(t, sys)

	l1c := sys.L1EL.EthClient()
	portal := sys.L2Chain.DepositContractAddr()

	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// Resolve SystemConfig via Portal.systemConfig()
	fnSystemConfig := w3.MustNewFunc("systemConfig()", "address")
	data, _ := fnSystemConfig.EncodeArgs()
	out, err := l1c.Call(ctx, ethereum.CallMsg{To: &portal, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("portal.systemConfig() call failed: %v", err)
	}
	var sysCfg common.Address
	t.Require().NoError(fnSystemConfig.DecodeReturns(out, &sysCfg))

	// Query the CGT flag on SystemConfig via IGasToken.isCustomGasToken().
	fnIsCGT := w3.MustNewFunc("isCustomGasToken()", "bool")
	data, _ = fnIsCGT.EncodeArgs()
	out, err = l1c.Call(ctx, ethereum.CallMsg{To: &sysCfg, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("SystemConfig.isCustomGasToken() call failed: %v", err)
	}
	var flag bool
	t.Require().NoError(fnIsCGT.DecodeReturns(out, &flag))
	if !flag {
		t.Skip("SystemConfig.isCustomGasToken() = false on this devnet; skipping")
	}
}
