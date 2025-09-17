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
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
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

func (mbf *minBaseFeeEnv) verifyMinBaseFee(t devtest.T, minBase *big.Int) {
	// Wait for the next block
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	info, err := el.InfoByLabel(t.Ctx(), "latest")
	t.Require().NoError(err)

	// Verify base fee is clamped
	t.Require().True(info.BaseFee().Cmp(minBase) >= 0, "expected base fee to be >= minBaseFee")
	t.Logf("base fee %s, minBase %s", info.BaseFee(), minBase)
}

// waitForMinBaseFeeConfigChangeOnL2 waits until the L2 latest payload extra-data encodes the expected min base fee.
func (mbf *minBaseFeeEnv) waitForMinBaseFeeConfigChangeOnL2(t devtest.T, expected uint64) {
	client := mbf.l2EL.Escape().L2EthClient()
	expectedExtraData := eth.BytesMax32(eip1559.EncodeMinBaseFeeExtraData(250, 6, expected))

	// Check extradata in block header (for all clients)
	var actualBlockExtraData []byte
	t.Require().Eventually(func() bool {
		info, err := client.InfoByLabel(t.Ctx(), "latest")
		if err != nil {
			return false
		}

		// Get header RLP and decode to access Extra field
		headerRLP, err := info.HeaderRLP()
		if err != nil {
			return false
		}

		var header types.Header
		if err := rlp.DecodeBytes(headerRLP, &header); err != nil {
			return false
		}

		if len(header.Extra) != 17 {
			return false
		}

		got := binary.BigEndian.Uint64(header.Extra[9:])
		actualBlockExtraData = header.Extra
		return got == expected
	}, 2*time.Minute, 5*time.Second, "L2 min base fee in block header did not sync within timeout")

	t.Require().Equal(expectedExtraData, eth.BytesMax32(actualBlockExtraData), "block header extradata doesnt match")
}

// TestMinBaseFee verifies configurable minimum base fee using devstack presets.
func TestMinBaseFee(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(err, "Jovian fork must be active for this test")

	minBaseFee := newMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	minBaseFee.checkCompatibility(t)

	systemOwner := minBaseFee.getSystemConfigOwner(t)
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name       string
		minBaseFee uint64
	}{
		// High minimum base fee
		{"MinBaseFeeHigh", 2_000_000_000},
		// Medium minimum base fee
		{"MinBaseFeeMedium", 1_000_000_000},
		// Zero minimum base fee (not enforced)
		{"MinBaseFeeZero", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			minBaseFee.setMinBaseFeeViaSytemConfigOnL1(t, tc.minBaseFee)
			minBaseFee.waitForMinBaseFeeConfigChangeOnL2(t, tc.minBaseFee)

			minBaseFee.verifyMinBaseFee(t, big.NewInt(int64(tc.minBaseFee)))

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"minBaseFee", tc.minBaseFee)
		})
	}
}
