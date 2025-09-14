package jovian

import (
	"context"
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
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/log"
)

type minBaseFee struct {
	// Ctx is the context for test execution.
	ctx context.Context
	// log is the component-specific logger instance.
	log log.Logger
	// T is a minimal test interface for panic-checks / assertions.
	t devtest.T
	// Require is a helper around the above T, ready to assert against.
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
	mbf.require.NoError(err, "SetMinBaseFee transaction failed")

	mbf.t.Logf("Set min base fee on L1: minBaseFee=%d", minBaseFee)
}

func (mbf *minBaseFee) verifyMinBaseFee(from *dsl.EOA, to *dsl.EOA, minBase *big.Int, shouldEnforce bool) {
	// Simulate user transactions
	for range 20 {
		from.Transfer(to.Address(), eth.OneGWei)
	}

	info := mbf.getBlock()
	prevBlockNum := info.NumberU64()
	for range 5 {
		n := mbf.getBlock()
		mbf.require.True(n.NumberU64() > prevBlockNum, "block number should increase")

		prevBlockNum = n.NumberU64()
		feeCmp := n.BaseFee().Cmp(minBase)
		if !shouldEnforce {
			mbf.require.True(feeCmp > 0, "expected base fee to be higher than the minBaseFee")
		} else {
			mbf.require.True(feeCmp == 0, "expected base fee to be at least minBaseFee")
		}
		mbf.t.Logf("base fee %s, minBase %s %d", n.BaseFee(), minBase)
	}
}

// waitForMinBaseFee waits until the L2 latest payload extra-data encodes the expected min base fee.
func (mbf *minBaseFee) waitForMinBaseFee(expected uint64) {
	client := mbf.l2EL.Escape().L2EthClient()
	ext, ok := client.(apis.L2EthExtendedClient)
	mbf.require.True(ok, "L2 client does not support extended payload API")

	expectedExtraData := eth.BytesMax32(eip1559.EncodeJovianExtraData(250, 6, expected))

	var actualPayload eth.BytesMax32
	mbf.require.Eventually(func() bool {
		payload, err := ext.PayloadByLabel(mbf.ctx, "latest")
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

	mbf.require.Equal(expectedExtraData, actualPayload, "extradata doesnt match")
}

func (mbf *minBaseFee) getBlock() eth.BlockInfo {
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	info, err := el.InfoByLabel(mbf.ctx, "latest")
	mbf.require.NoError(err)
	return info
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
	bob := sys.Wallet.NewEOA(sys.L2EL)

	minBaseFee := newMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	minBaseFee.checkCompatibility()

	systemOwner := minBaseFee.getSystemOwner()
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name          string
		minBaseFee    uint64
		shouldEnforce bool
	}{
		// The min base fee is set too low so when there's activity, we enforce the
		// calculated base fee over the min base fee.
		{"MinBaseFeeNotEnforced", 0, false},
		// The min base fee is enforced since the calculated base fee is below the min base fee.
		{"MinBaseFeeEnforced", 1_000_000_000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.setMinBaseFee(tc.minBaseFee)
			minBaseFee.waitForMinBaseFee(tc.minBaseFee)

			minBaseFee.verifyMinBaseFee(alice, bob, big.NewInt(int64(tc.minBaseFee)), tc.shouldEnforce)

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"minBaseFee", tc.minBaseFee,
				"shouldEnforce", tc.shouldEnforce)
		})
	}
}
