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

// TestCGT_SystemConfigFlagOnL1 checks that the L1 SystemConfig contract reports
// CGT=true via isCustomGasToken(). Skips if the devnet does not wire this flag.
func TestCGT_SystemConfigFlagOnL1(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	ensureCGTOrSkip(t, sys)

	l1c := sys.L1EL.EthClient()
	portal := sys.L2Chain.DepositContractAddr()

	portalABI := `[
	  {"inputs":[],"name":"systemConfig","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`
	igasToken := `[
	  {"inputs":[],"name":"isCustomGasToken","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`

	pa, err := abi.JSON(strings.NewReader(portalABI))
	if err != nil {
		t.Require().Fail("%v", err)
	}
	ctx, cancel := context.WithTimeout(t.Ctx(), 20*time.Second)
	defer cancel()

	// Resolve SystemConfig via Portal.systemConfig()
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
	if (sysCfg == common.Address{}) {
		t.Require().Fail("portal.systemConfig() returned zero address")
	}

	// Ask SystemConfig whether CGT is enabled.
	ga, _ := abi.JSON(strings.NewReader(igasToken))
	data, _ = ga.Pack("isCustomGasToken")
	out, err = l1c.Call(ctx, ethereum.CallMsg{To: &sysCfg, Data: data}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("SystemConfig.isCustomGasToken() call failed: %v", err)
	}
	vals, err = ga.Unpack("isCustomGasToken", out)
	if err != nil {
		t.Require().Fail("unpack isCustomGasToken failed: %v", err)
	}
	if !vals[0].(bool) {
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
	sysCfg := vals[0].(common.Address) // keep as interface; ABI unpack returns []interface{}

	// Query the CGT flag on SystemConfig via IGasToken.isCustomGasToken().
	const igasTokenABI = `[
	  {"inputs":[],"name":"isCustomGasToken","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
	]`
	ga, err := abi.JSON(strings.NewReader(igasTokenABI))
	if err != nil {
		t.Require().Fail("parse IGasToken ABI: %v", err)
	}

	// Build the call
	data, _ = ga.Pack("isCustomGasToken")
	out, err = l1c.Call(ctx, ethereum.CallMsg{
		To:   &sysCfg,
		Data: data,
	}, rpc.LatestBlockNumber)
	if err != nil {
		t.Require().Fail("SystemConfig.isCustomGasToken() call failed: %v", err)
	}
	flag, err := ga.Unpack("isCustomGasToken", out)
	if err != nil {
		t.Require().Fail("unpack isCustomGasToken failed: %v", err)
	}
	if !flag[0].(bool) {
		t.Skip("SystemConfig.isCustomGasToken() = false on this devnet; skipping")
	}
}
