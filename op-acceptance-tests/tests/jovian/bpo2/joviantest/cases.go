package joviantest

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/jovian"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type SetupFn func(t devtest.T) *presets.Minimal

type daFootprintSystemConfig struct {
	SetDAFootprintGasScalar func(scalar uint16) bindings.TypedCall[any] `sol:"setDAFootprintGasScalar"`
	DAFootprintGasScalar    func() bindings.TypedCall[uint16]           `sol:"daFootprintGasScalar"`
}

type daFootprintL1Block struct {
	DAFootprintGasScalar func() bindings.TypedCall[uint16] `sol:"daFootprintGasScalar"`
}

type daFootprintEnv struct {
	l1Client     *dsl.L1ELNode
	l2Network    *dsl.L2Network
	l2EL         *dsl.L2ELNode
	systemConfig daFootprintSystemConfig
	l1Block      daFootprintL1Block
}

func newDAFootprintEnv(t devtest.T, l2Network *dsl.L2Network, l1EL *dsl.L1ELNode, l2EL *dsl.L2ELNode) *daFootprintEnv {
	systemConfig := bindings.NewBindings[daFootprintSystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t),
	)

	l1Block := bindings.NewBindings[daFootprintL1Block](
		bindings.WithClient(l2EL.Escape().EthClient()),
		bindings.WithTo(common.HexToAddress("0x4200000000000000000000000000000000000015")),
		bindings.WithTest(t),
	)

	return &daFootprintEnv{
		l1Client:     l1EL,
		l2Network:    l2Network,
		l2EL:         l2EL,
		systemConfig: systemConfig,
		l1Block:      l1Block,
	}
}

func (env *daFootprintEnv) checkCompatibility(t devtest.T) {
	// Ensure getters exist on both L1 SystemConfig and L2 L1Block.
	_, err := contractio.Read(env.systemConfig.DAFootprintGasScalar(), t.Ctx())
	t.Require().NoError(err)
	_, err = contractio.Read(env.l1Block.DAFootprintGasScalar(), t.Ctx())
	t.Require().NoError(err)
}

func (env *daFootprintEnv) getSystemConfigOwner(t devtest.T) *dsl.EOA {
	priv := env.l2Network.Escape().Keys().Secret(devkeys.SystemConfigOwner.Key(env.l2Network.ChainID().ToBig()))
	return dsl.NewKey(t, priv).User(env.l1Client)
}

func (env *daFootprintEnv) setDAFootprintGasScalarViaSystemConfig(t devtest.T, scalar uint16) *types.Receipt {
	owner := env.getSystemConfigOwner(t)
	rec, err := contractio.Write(env.systemConfig.SetDAFootprintGasScalar(scalar), t.Ctx(), owner.Plan())
	t.Require().NoError(err, "SetDAFootprintGasScalar transaction failed")
	t.Logf("Set DA footprint gas scalar on L1: scalar=%d", scalar)
	return rec
}

func (env *daFootprintEnv) getDAFootprintGasScalarOfSystemConfig(t devtest.T) uint16 {
	scalar, err := contractio.Read(env.systemConfig.DAFootprintGasScalar(), t.Ctx())
	t.Require().NoError(err)
	return scalar
}

// expectL1BlockDAFootprintGasScalar expects the given DA footprint gas scalar to be set in the L1Block contract.
func (env *daFootprintEnv) expectL1BlockDAFootprintGasScalar(t devtest.T, expected uint16) {
	current, err := contractio.Read(env.l1Block.DAFootprintGasScalar(), t.Ctx())
	t.Require().NoError(err, "Failed to read DA footprint gas scalar from L1Block")
	t.Require().Equal(expected, current)
}

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
		bindings.WithTest(t),
	)

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
	// Wait for the next block.
	_ = mbf.l2EL.WaitForBlock()
	el := mbf.l2EL.Escape().EthClient()
	info, err := el.InfoByLabel(t.Ctx(), "latest")
	t.Require().NoError(err)

	// Verify base fee is clamped.
	t.Require().True(info.BaseFee().Cmp(minBase) >= 0, "expected base fee to be >= minBaseFee")
	t.Logf("base fee %s, minBase %s", info.BaseFee(), minBase)
}

// waitForMinBaseFeeConfigChangeOnL2 waits until the L2 latest payload extra-data encodes the expected min base fee.
func (mbf *minBaseFeeEnv) waitForMinBaseFeeConfigChangeOnL2(t devtest.T, expected uint64) {
	client := mbf.l2EL.Escape().L2EthClient()
	expectedExtraData := eth.BytesMax32(eip1559.EncodeJovianExtraData(250, 6, expected))

	// Check extradata in block header (for all clients).
	var actualBlockExtraData []byte
	t.Require().Eventually(func() bool {
		info, err := client.InfoByLabel(t.Ctx(), "latest")
		if err != nil {
			return false
		}

		// Get header RLP and decode to access Extra field.
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

func RunDAFootprint(gt *testing.T, setup SetupFn) {
	t := devtest.ParallelT(gt)
	sys := setup(t)
	require := t.Require()

	require.True(sys.L2Chain.IsForkActive(opforks.Jovian), "Jovian fork must be active for this test")

	env := newDAFootprintEnv(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	env.checkCompatibility(t)

	systemOwner := env.getSystemConfigOwner(t)
	sys.FunderL1.FundAtLeast(systemOwner, eth.HalfEther)
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	sys.L2EL.WaitForOnline()
	ethClient := sys.L2EL.Escape().EthClient()

	s1000 := uint16(1000)
	s0 := uint16(0)
	cases := []struct {
		name      string
		setScalar *uint16
		expected  uint16
	}{
		{"DefaultScalar", nil, uint16(derive.DAFootprintGasScalarDefault)},
		{"Scalar1000", &s1000, uint16(1000)},
		{"ScalarZeroUsesDefault", &s0, uint16(derive.DAFootprintGasScalarDefault)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t devtest.T) {
			require := t.Require()
			if tc.setScalar != nil {
				rec := env.setDAFootprintGasScalarViaSystemConfig(t, *tc.setScalar)
				// Wait for change to propagate to L2.
				// Retrying up to 100 times is overkill, but lower values may not work on
				// persistent networks. See the following issue for more details.
				// https://github.com/ethereum-optimism/optimism/issues/18061
				env.l2EL.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(rec.BlockNumber), 100)
			} else {
				scalar := env.getDAFootprintGasScalarOfSystemConfig(t)
				if scalar != 0 {
					t.Skipf("Skipping default scalar test because SystemConfig DA footprint gas scalar is set to %d != 0", scalar)
				}
				sys.L2EL.WaitForBlockNumber(1) // make sure we don't assert on genesis.
			}
			env.expectL1BlockDAFootprintGasScalar(t, tc.expected)

			jovian.SpamCalldata(t, l2BlockTime, sys.L2EL, sys.Wallet, sys.FaucetL2)

			rollupCfg := sys.L2Chain.Escape().RollupConfig()
			gasTarget := rollupCfg.Genesis.SystemConfig.GasLimit / rollupCfg.ChainOpConfig.EIP1559Elasticity

			var blockDAFootprint uint64
			info := sys.L2EL.WaitForUnsafe(func(info eth.BlockInfo) (bool, error) {
				blockGasUsed := info.GasUsed()
				blobGasUsed := info.BlobGasUsed()
				require.NotNil(blobGasUsed, "blobGasUsed must not be nil for Jovian chains")
				blockDAFootprint = *blobGasUsed
				if blockDAFootprint <= blockGasUsed {
					t.Logf("Block %s has DA footprint (%d) <= gasUsed (%d), trying next...",
						eth.ToBlockID(info), blockDAFootprint, blockGasUsed)
					return false, nil
				}
				if blockDAFootprint <= gasTarget {
					t.Logf("Block %s has DA footprint (%d) <= gasTarget (%d), trying next...",
						eth.ToBlockID(info), blockDAFootprint, gasTarget)
					return false, nil
				}
				return true, nil
			})

			_, txs, err := ethClient.InfoAndTxsByHash(t.Ctx(), info.Hash())
			require.NoError(err)
			_, receipts, err := sys.L2EL.Escape().L2EthClient().FetchReceipts(t.Ctx(), info.Hash())
			require.NoError(err)

			var totalDAFootprint uint64
			for i, tx := range txs {
				if tx.IsDepositTx() {
					continue
				}
				recScalar := receipts[i].DAFootprintGasScalar
				require.NotNil(recScalar, "nil receipt DA footprint gas scalar")
				require.EqualValues(tc.expected, *recScalar, "DA footprint gas scalar mismatch in receipt")

				txDAFootprint := bigs.Uint64Strict(tx.RollupCostData().EstimatedDASize()) * uint64(tc.expected)
				require.Equal(txDAFootprint, receipts[i].BlobGasUsed, "tx DA footprint mismatch with receipt")
				totalDAFootprint += txDAFootprint
			}
			t.Logf("Block %s has header/calculated DA footprint %d/%d",
				eth.ToBlockID(info), blockDAFootprint, totalDAFootprint)
			require.Equal(totalDAFootprint, blockDAFootprint, "Calculated DA footprint doesn't match block header DA footprint")

			// Check base fee calculation of next block.
			// Calculate expected base fee as:
			// parentBaseFee + max(1, parentBaseFee * gasUsedDelta / parentGasTarget / baseFeeChangeDenominator)
			var (
				baseFee = new(big.Int)
				denom   = new(big.Int)
			)
			baseFee.SetUint64(blockDAFootprint - gasTarget) // gasUsedDelta
			baseFee.Mul(baseFee, info.BaseFee())
			baseFee.Div(baseFee, denom.SetUint64(gasTarget))
			baseFee.Div(baseFee, denom.SetUint64(*rollupCfg.ChainOpConfig.EIP1559DenominatorCanyon))
			if baseFee.Cmp(common.Big1) < 0 {
				baseFee.Add(info.BaseFee(), common.Big1)
			} else {
				baseFee.Add(info.BaseFee(), baseFee)
			}
			t.Logf("Expected base fee: %s", baseFee)

			next := sys.L2EL.WaitForBlockNumber(info.NumberU64() + 1)
			require.Equal(baseFee, next.BaseFee(), "Wrong base fee")
		})
	}
}

func RunMinBaseFee(gt *testing.T, setup SetupFn) {
	t := devtest.ParallelT(gt)
	sys := setup(t)
	require := t.Require()

	require.True(sys.L2Chain.IsForkActive(opforks.Jovian), "Jovian fork must be active for this test")

	minBaseFee := newMinBaseFee(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	minBaseFee.checkCompatibility(t)

	systemOwner := minBaseFee.getSystemConfigOwner(t)
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)

	testCases := []struct {
		name       string
		minBaseFee uint64
	}{
		// High minimum base fee.
		{"MinBaseFeeHigh", 2_000_000_000},
		// Medium minimum base fee.
		{"MinBaseFeeMedium", 1_000_000_000},
		// Zero minimum base fee (not enforced).
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

func RunOperatorFee(gt *testing.T, setup SetupFn) {
	t := devtest.ParallelT(gt)
	sys := setup(t)
	t.Require().True(sys.L2Chain.IsForkActive(opforks.Jovian), "Jovian fork must be active for this test")
	dsl.RunOperatorFeeTest(t, sys.L2Chain, sys.L1EL, sys.FunderL1, sys.FunderL2)
}
