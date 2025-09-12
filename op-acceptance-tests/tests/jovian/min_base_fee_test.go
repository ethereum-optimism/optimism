package jovian

import (
	"context"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type minBaseFee struct {
	ctx     context.Context
	log     log.Logger
	t       devtest.T
	require *testreq.Assertions

	l1Client     *dsl.L1ELNode
	l2Network    *dsl.L2Network
	l2EL         *dsl.L2ELNode
	systemConfig minBaseFeeSystemConfig

	originalMinBaseFee uint64
}

type minBaseFeeSystemConfig struct {
	SetMinBaseFee func(minBaseFee uint64) bindings.TypedCall[any] `sol:"setMinBaseFee"`
	MinBaseFee    func() bindings.TypedCall[uint64]               `sol:"minBaseFee"`
}

func newMinBaseFee(t devtest.T, l2Network *dsl.L2Network, l1EL *dsl.L1ELNode, l2EL *dsl.L2ELNode) *minBaseFee {
	systemConfig := bindings.NewBindings[minBaseFeeSystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t))

	originalMinBaseFee, err := contractio.Read(systemConfig.MinBaseFee(), t.Ctx())
	t.Require().NoError(err, "reading original minBaseFee")

	return &minBaseFee{
		ctx:                t.Ctx(),
		log:                t.Logger(),
		t:                  t,
		require:            t.Require(),
		l1Client:           l1EL,
		l2Network:          l2Network,
		l2EL:               l2EL,
		systemConfig:       systemConfig,
		originalMinBaseFee: originalMinBaseFee,
	}
}

func (mbf *minBaseFee) checkCompatibility() bool {
	_, err := contractio.Read(mbf.systemConfig.MinBaseFee(), mbf.ctx)
	if err != nil {
		mbf.t.Fail()
		return false
	}
	return true
}

func (mbf *minBaseFee) getSystemOwner() *dsl.EOA {
	priv := mbf.l2Network.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(mbf.l2Network.ChainID().ToBig()))
	return dsl.NewKey(mbf.t, priv).User(mbf.l1Client)
}

func (mbf *minBaseFee) setMinBaseFee(minBaseFee uint64) {
	owner := mbf.getSystemOwner()

	_, err := contractio.Write(mbf.systemConfig.SetMinBaseFee(minBaseFee), mbf.ctx, owner.Plan())
	mbf.require.NoError(err, "setMinBaseFee transaction failed")

	mbf.t.Logf("Set min base fee on L1: minBaseFee=%d", minBaseFee)
}

func (mbf *minBaseFee) checkBaseFeeCanDecrease() {
	var prevBlockNum uint64
	// Ensure we are past genesis and collect a small sample across advancing blocks
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	bases := make([]*big.Int, 0, 6)
	info, err := el.InfoByLabel(mbf.ctx, "latest")
	prevBlockNum = info.NumberU64()
	mbf.require.NoError(err)
	bases = append(bases, info.BaseFee())
	for range 5 {
		_ = mbf.l2EL.WaitForBlock()
		next, err := el.InfoByLabel(mbf.ctx, "latest")
		mbf.require.Greater(next.NumberU64(), prevBlockNum, "expected block number to increase")
		prevBlockNum = next.NumberU64()
		mbf.require.NoError(err)
		bases = append(bases, next.BaseFee())
	}
	decreased := false
	for i := 1; i < len(bases); i++ {
		if bases[i].Cmp(bases[i-1]) < 0 {
			decreased = true
			break
		}
	}
	mbf.require.True(decreased, "expected base-fee to decrease when minBaseFee=0")
}

func (mbf *minBaseFee) verifyMinBaseFeeClamp(minBase *big.Int) {
	// Give the sequencer one more block, then check 5 consecutive blocks
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()

	// Check 5 consecutive blocks to ensure min base fee is consistently applied
	for i := 1; i <= 5; i++ {
		_ = mbf.l2EL.WaitForBlock()
		info, err := el.InfoByLabel(mbf.ctx, "latest")
		mbf.require.NoError(err)
		mbf.require.True(info.BaseFee().Cmp(minBase) >= 0, "block %d base-fee %s should be >= %s", info.NumberU64(), info.BaseFee(), minBase)
	}
}

func (mbf *minBaseFee) restoreOriginalConfig() {
	mbf.setMinBaseFee(mbf.originalMinBaseFee)
	mbf.waitForMinBaseFee(mbf.originalMinBaseFee)
}

// waitForMinBaseFee waits until the L2 latest payload extra-data encodes the expected min base fee.
func (mbf *minBaseFee) waitForMinBaseFee(expected uint64) {
	client := mbf.l2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	mbf.require.True(ok, "L2 client does not support extended payload API")

	// Construct expected Jovian extraData with the expected min base fee
	expectedExtraData := eip1559.EncodeJovianExtraData(250, 6, expected)

	mbf.require.Eventually(func() bool {
		payload, err := ext.PayloadByLabel(mbf.ctx, "latest")
		if err != nil {
			mbf.t.Logf("Failed to get latest payload: %v", err)
			return false
		}

		// Assert payload has Jovian extraData format (17 bytes)
		actualExtraData, err := payload.ExecutionPayload.ExtraData.MarshalText()
		if err != nil {
			mbf.t.Logf("Failed to get extra data: %v", err)
			return false
		}
		if len(actualExtraData) != 17 {
			mbf.t.Logf("ExtraData length is %d, expected 17 bytes for Jovian format", len(actualExtraData))
			return false
		}

		// Assert extraData matches exactly what we expect for Jovian with the min base fee
		mbf.require.Equal(expectedExtraData, actualExtraData, "ExtraData mismatch")

		// Extract and verify the encoded min base fee
		encodedMinBaseFee := binary.BigEndian.Uint64(actualExtraData[9:])
		if encodedMinBaseFee != expected {
			mbf.t.Logf("Encoded min base fee mismatch: expected %d, got %d", expected, encodedMinBaseFee)
			return false
		}
		mbf.require.Equal(encodedMinBaseFee, expected, "Encoded min base fee should be greater than expected")

		// Assert that the block's actual base fee is >= minBaseFee
		mbf.require.GreaterOrEqual(payload.ExecutionPayload.BaseFeePerGas, expected, "Block base fee should be > 0")

		return true
	}, 2*time.Minute, 5*time.Second, "L2 min base fee did not sync within timeout")
}

// TestMinBaseFee verifies configurable minimum base fee using devstack presets.
func TestMinBaseFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(err, "Jovian fork must be active for this test")

	minBaseFee := newMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)

	minBaseFee.checkCompatibility()
	systemOwner := minBaseFee.getSystemOwner()
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name        string
		minBaseFee  uint64
		shouldClamp bool
	}{
		{"MinBaseFeeOff", 0, false},
		{"MinBaseFeeOn", 1_000_000_000, true},
		{"MinBaseFeeBackToOff", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.setMinBaseFee(tc.minBaseFee)
			minBaseFee.waitForMinBaseFee(tc.minBaseFee)

			if tc.shouldClamp {
				minBaseFee.verifyMinBaseFeeClamp(big.NewInt(int64(tc.minBaseFee)))
			} else {
				minBaseFee.checkBaseFeeCanDecrease()
			}

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"minBaseFee", tc.minBaseFee,
				"shouldClamp", tc.shouldClamp)
		})
	}

	minBaseFee.restoreOriginalConfig()
}
