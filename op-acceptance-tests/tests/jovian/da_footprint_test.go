package jovian

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type CalldataSpammer struct {
	eoa *loadtest.SyncEOA
}

func NewCalldataSpammer(eoa *loadtest.SyncEOA) *CalldataSpammer {
	return &CalldataSpammer{
		eoa: eoa,
	}
}

func (s *CalldataSpammer) Spam(t devtest.T) error {
	data := make([]byte, 50_000)
	_, err := rand.Read(data)
	t.Require().NoError(err)
	_, err = s.eoa.Include(t, txplan.WithTo(&common.Address{}), txplan.WithData(data))
	return err
}

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
	// Ensure getters exist on both L1 SystemConfig and L2 L1Block
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

// expectL1BlockDAFootprintGasScalar expects the given DA footprint gas scalar to be set in the L1Block contract.
func (env *daFootprintEnv) expectL1BlockDAFootprintGasScalar(t devtest.T, expected uint16) {
	current, err := contractio.Read(env.l1Block.DAFootprintGasScalar(), t.Ctx())
	t.Require().NoError(err, "Failed to read DA footprint gas scalar from L1Block")
	t.Require().Equal(expected, current)
}

func TestDAFootprint(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(err, "Jovian fork must be active for this test")

	env := newDAFootprintEnv(t, sys.L2Chain, sys.L1EL, sys.L2EL)
	env.checkCompatibility(t)

	systemOwner := env.getSystemConfigOwner(t)
	sys.FunderL1.FundAtLeast(systemOwner, eth.OneTenthEther)
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
		{"DefaultScalar", nil, uint16(eth.DAFootprintGasScalarDefault)},
		{"Scalar1000", &s1000, uint16(1000)},
		{"ScalarZeroUsesDefault", &s0, uint16(eth.DAFootprintGasScalarDefault)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t devtest.T) {
			if tc.setScalar != nil {
				rec := env.setDAFootprintGasScalarViaSystemConfig(t, *tc.setScalar)
				// Wait for change to propagate to L2
				env.l2EL.WaitL1OriginReached(eth.Unsafe, rec.BlockNumber.Uint64(), 20)
			} else {
				sys.L2EL.WaitForBlockNumber(2) // make sure we don't assert on genesis or first block
			}
			env.expectL1BlockDAFootprintGasScalar(t, tc.expected)

			var wg sync.WaitGroup
			defer wg.Wait()

			ctx, cancel := context.WithTimeout(t.Ctx(), time.Minute)
			defer cancel()
			t = t.WithCtx(ctx)

			wg.Add(1)
			go func() {
				defer wg.Done()
				eoa := sys.FunderL2.NewFundedEOA(eth.OneEther.Mul(100))
				includer := txinclude.NewPersistent(txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig()), struct {
					*txinclude.Resubmitter
					*txinclude.Monitor
				}{
					txinclude.NewResubmitter(ethClient, l2BlockTime),
					txinclude.NewMonitor(ethClient, l2BlockTime),
				})
				loadtest.NewBurst(l2BlockTime).Run(t, NewCalldataSpammer(loadtest.NewSyncEOA(includer, eoa.Plan())))
			}()

			rollupCfg := sys.L2Chain.Escape().RollupConfig()
			gasTarget := rollupCfg.Genesis.SystemConfig.GasLimit / rollupCfg.ChainOpConfig.EIP1559Elasticity

			var blockDAFootprint uint64
			info := sys.L2EL.WaitForUnsafe(func(info eth.BlockInfo) (bool, error) {
				blockGasUsed := info.GasUsed()
				blobGasUsed := info.BlobGasUsed()
				t.Require().NotNil(blobGasUsed, "blobGasUsed must not be nil for Jovian chains")
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
			t.Require().NoError(err)

			var totalDAFootprint uint64
			for _, tx := range txs {
				if tx.IsDepositTx() {
					continue
				}
				totalDAFootprint += tx.RollupCostData().EstimatedDASize().Uint64() * uint64(tc.expected)
			}
			t.Logf("Block %s has header/calculated DA footprint %d/%d",
				eth.ToBlockID(info), blockDAFootprint, totalDAFootprint)
			t.Require().Equal(totalDAFootprint, blockDAFootprint, "Calculated DA footprint doesn't match block header DA footprint")

			// Check base fee calculation of next block
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
			t.Require().Equal(baseFee, next.BaseFee(), "Wrong base fee")
		})
	}
}
