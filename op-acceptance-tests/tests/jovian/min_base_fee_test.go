package jovian

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"encoding/binary"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
)

type minBaseFeeEnv struct {
	l1Client     *dsl.L1ELNode
	l2Network    *dsl.L2Network
	l2EL         *dsl.L2ELNode
	systemConfig minBaseFeeSystemConfig
}

type minBaseFeeSystemConfig struct {
	SetMinBaseFee func(minBaseFee uint64) bindings.TypedCall[any] `sol:"setMinBaseFee"`
	MinBaseFee    func() bindings.TypedCall[uint64]               `sol:"minBaseFee"`
}

func newMinBaseFee(t devtest.T, l2Network *dsl.L2Network, l1EL *dsl.L1ELNode, l2EL *dsl.L2ELNode) *minBaseFeeEnv {
	systemConfig := bindings.NewBindings[minBaseFeeSystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t))

	return &minBaseFeeEnv{
		l1Client:     l1EL,
		l2Network:    l2Network,
		l2EL:         l2EL,
		systemConfig: systemConfig,
	}
}

func (mbf *minBaseFeeEnv) checkCompatibility(t devtest.T) {
	_, err := contractio.Read(mbf.systemConfig.MinBaseFee(), t.Ctx())
	if err != nil {
		t.Fail()
	}
}

func (mbf *minBaseFeeEnv) getSystemConfigOwner(t devtest.T) *dsl.EOA {
	priv := mbf.l2Network.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(mbf.l2Network.ChainID().ToBig()))
	return dsl.NewKey(t, priv).User(mbf.l1Client)
}

func (mbf *minBaseFeeEnv) setMinBaseFeeViaSytemConfigOnL1(t devtest.T, minBaseFee uint64) {
	owner := mbf.getSystemConfigOwner(t)

	_, err := contractio.Write(mbf.systemConfig.SetMinBaseFee(minBaseFee), t.Ctx(), owner.Plan())
	t.Require().NoError(err, "SetMinBaseFee transaction failed")

	t.Logf("Set min base fee on L1: minBaseFee=%d", minBaseFee)
}

func (mbf *minBaseFeeEnv) verifyMinBaseFee(t devtest.T, from *dsl.EOA, minBase *big.Int, shouldEnforce bool) {
	// Simulate user transactions
	for range 20 {
		from.Transfer(common.Address{}, eth.OneGWei)
	}

	// Wait for the next block
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	info, err := el.InfoByLabel(t.Ctx(), "latest")
	t.Require().NoError(err)

	feeCmp := info.BaseFee().Cmp(minBase)
	if !shouldEnforce {
		t.Require().True(feeCmp > 0, "expected base fee to be higher than the minBaseFee")
	} else {
		t.Require().True(feeCmp == 0, "expected base fee to be at least minBaseFee")
	}
	t.Logf("base fee %s, minBase %s", info.BaseFee(), minBase)
}

// waitForMinBaseFeeConfigChangeOnL2 waits until the L2 latest payload extra-data encodes the expected min base fee.
func (mbf *minBaseFeeEnv) waitForMinBaseFeeConfigChangeOnL2(t devtest.T, expected uint64) {
	client := mbf.l2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	t.Require().True(ok, "L2 client does not support extended payload API")

	expectedExtraData := eth.BytesMax32(eip1559.EncodeJovianExtraData(250, 6, expected))

	var actualPayload eth.BytesMax32
	t.Require().Eventually(func() bool {
		payload, err := ext.PayloadByLabel(t.Ctx(), "latest")
		if err != nil {
			return false
		}
		if len(payload.ExecutionPayload.ExtraData) != 17 {
			return false
		}

		got := binary.BigEndian.Uint64(payload.ExecutionPayload.ExtraData[9:])
		actualPayload = payload.ExecutionPayload.ExtraData
		return got == expected
	}, 2*time.Minute, 5*time.Second, "L2 min base fee did not sync within timeout")

	t.Require().Equal(expectedExtraData, actualPayload, "extradata doesnt match")
}

// TestMinBaseFee verifies configurable minimum base fee using devstack presets.
func TestMinBaseFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(err, "Jovian fork must be active for this test")

	fundAmount := eth.OneTenthEther
	alice := sys.FunderL2.NewFundedEOA(fundAmount)
	alice.WaitForBalance(fundAmount)

	minBaseFee := newMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	minBaseFee.checkCompatibility(t)

	systemOwner := minBaseFee.getSystemConfigOwner(t)
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name          string
		minBaseFee    uint64
		shouldEnforce bool
	}{
		// The min base fee is enforced since the calculated base fee is below the min base fee.
		{"MinBaseFeeEnforced", 1_000_000_000, true},
		// The min base fee is set too low so when there's activity, we enforce the
		// calculated base fee over the min base fee.
		{"MinBaseFeeNotEnforced", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.setMinBaseFeeViaSytemConfigOnL1(t, tc.minBaseFee)
			minBaseFee.waitForMinBaseFeeConfigChangeOnL2(t, tc.minBaseFee)

			minBaseFee.verifyMinBaseFee(t, alice, big.NewInt(int64(tc.minBaseFee)), tc.shouldEnforce)

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"minBaseFee", tc.minBaseFee,
				"shouldEnforce", tc.shouldEnforce)
		})
	}
}
