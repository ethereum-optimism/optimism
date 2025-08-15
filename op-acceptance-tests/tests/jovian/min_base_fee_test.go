package jovian

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
)

// TestConfigurableMinBaseFee verifies configurable minimum base fee using devstack presets (sysgo).
func TestConfigurableMinBaseFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	if err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian); err != nil {
		t.Skipf("precondition not met: %v", err)
	}

	t.Run("MinBaseFeeOff", func(tt devtest.T) {
		// Ensure minBaseFee factors are 0 and base-fee can decrease.
		ensureMinBaseFeeFactors(tt, sys, 0, 0)
		// Wait for L2 to reflect factors=0
		waitForMinBaseFeeFactors(tt, sys, eip1559.EncodeMinBaseFeeFactors(0, 0), 1*time.Minute)
		checkBaseFeeCanDecrease(tt, sys)
	})

	t.Run("MinBaseFeeOn", func(tt devtest.T) {
		// Configure 1 gwei minimum base fee and verify it clamps base-fee
		significand := uint8(1)
		exponent := uint8(9)
		minBase := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
		minBase.Mul(minBase, big.NewInt(int64(significand)))

		ensureMinBaseFeeFactors(tt, sys, significand, exponent)
		// Wait until extra-data reflects new factors
		waitForMinBaseFeeFactors(tt, sys, eip1559.EncodeMinBaseFeeFactors(significand, exponent), 2*time.Minute)
		// Verify subsequent blocks respect the minimum
		verifyMinBaseFeeClamp(tt, sys, minBase)
	})
}

// ensureMinBaseFeeFactors sets the minBaseFee factors via SystemConfig on L1.
func ensureMinBaseFeeFactors(t devtest.T, sys *presets.Minimal, significand, exponent uint8) {
	owner := systemConfigOwnerEOA(t, sys)
	// Build a minimal binding with SetMinBaseFee
	type sysCfgMinBaseFee struct {
		SetMinBaseFee func(sig uint8, exp uint8) bindings.TypedCall[any] `sol:"setMinBaseFee"`
	}
	syscfg := bindings.NewBindings[sysCfgMinBaseFee](
		bindings.WithTest(t),
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(sys.L2Chain.Escape().Deployment().SystemConfigProxyAddr()),
	)
	_, err := contractio.Write(syscfg.SetMinBaseFee(significand, exponent), t.Ctx(), owner.Plan())
	t.Require().NoError(err, "SetMinBaseFee transaction failed")
}

// systemConfigOwnerEOA derives the SystemConfigOwner role key and binds it to L1 EL.
func systemConfigOwnerEOA(t devtest.T, sys *presets.Minimal) *dsl.EOA {
	priv := sys.L2Chain.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(sys.L2Chain.ChainID().ToBig()))
	return dsl.NewKey(t, priv).User(sys.L1EL)
}

// waitForMinBaseFeeFactors waits until the L2 latest payload extra-data encodes the expected factors.
func waitForMinBaseFeeFactors(t devtest.T, sys *presets.Minimal, expected uint8, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	client := sys.L2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	t.Require().True(ok, "L2 client does not support extended payload API")
	for time.Now().Before(deadline) {
		payload, err := ext.PayloadByLabel(t.Ctx(), "latest")
		if err == nil && len(payload.ExecutionPayload.ExtraData) == 10 {
			got := uint8(payload.ExecutionPayload.ExtraData[9])
			if got == expected {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Require().Failf("timeout", "timeout waiting for L2 to reflect minBaseFeeFactors (want=0x%02x)", expected)
}

// checkBaseFeeCanDecrease collects a few blocks and verifies base-fee can decrease.

func checkBaseFeeCanDecrease(t devtest.T, sys *presets.Minimal) {
	// Ensure we are past genesis and collect a small sample across advancing blocks
	_ = sys.L2EL.WaitForBlock()
	el := sys.L2EL.Escape().EthClient()
	bases := make([]*big.Int, 0, 6)
	info, err := el.InfoByLabel(t.Ctx(), "latest")
	t.Require().NoError(err)
	bases = append(bases, info.BaseFee())
	for i := 0; i < 5; i++ {
		_ = sys.L2EL.WaitForBlock()
		next, err := el.InfoByLabel(t.Ctx(), "latest")
		t.Require().NoError(err)
		bases = append(bases, next.BaseFee())
	}
	decreased := false
	for i := 1; i < len(bases); i++ {
		if bases[i].Cmp(bases[i-1]) < 0 {
			decreased = true
			break
		}
	}
	t.Require().True(decreased, "expected base-fee to decrease when minBaseFee=0")
}

// verifyMinBaseFeeClamp verifies that subsequent blocks respect the minimum base fee.
func verifyMinBaseFeeClamp(t devtest.T, sys *presets.Minimal, minBase *big.Int) {
	// Give the sequencer one more block, then check a few consecutive blocks
	_ = sys.L2EL.WaitForBlock()
	el := sys.L2EL.Escape().EthClient()
	head, err := el.InfoByLabel(t.Ctx(), "latest")
	t.Require().NoError(err)
	start := head.NumberU64()
	// Skip the first block, as the minBaseFeeFactors are not yet applied.
	for i := 1; i < 5; i++ {
		target := start + uint64(i)
		waitUntilL2BlockAvailable(t, sys, target, 10*time.Second)
		info, err := el.InfoByNumber(t.Ctx(), target)
		t.Require().NoError(err)
		t.Require().True(info.BaseFee().Cmp(minBase) >= 0, "block %d base-fee %s should be >= %s", target, info.BaseFee(), minBase)
	}
}

// waitUntilL2BlockAvailable waits until the given L2 block number exists and is queryable.
func waitUntilL2BlockAvailable(t devtest.T, sys *presets.Minimal, target uint64, timeout time.Duration) {
	el := sys.L2EL.Escape().EthClient()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		head, err := el.InfoByLabel(t.Ctx(), "latest")
		t.Require().NoError(err)
		if head.NumberU64() >= target {
			if info, err := el.InfoByNumber(t.Ctx(), target); err == nil && info != nil {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Require().Failf("timeout", "timeout waiting for L2 block %d to be available", target)
}
