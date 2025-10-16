package dsl

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/core/types"
)

type OperatorFee struct {
	commonImpl

	l1Client       *L1ELNode
	l2Network      *L2Network
	systemConfig   bindings.SystemConfig
	l1Block        bindings.L1Block
	gasPriceOracle bindings.GasPriceOracle

	originalScalar   uint32
	originalConstant uint64
}

type OperatorFeeValidationResult struct {
	TransactionReceipt   *types.Receipt
	ExpectedOperatorFee  *big.Int
	ActualTotalFee       *big.Int
	VaultBalanceIncrease *big.Int
}

func NewOperatorFee(t devtest.T, l2Network *L2Network, l1EL *L1ELNode) *OperatorFee {
	systemConfig := bindings.NewBindings[bindings.SystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Network.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t))

	l1Block := bindings.NewBindings[bindings.L1Block](
		bindings.WithClient(l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()),
		bindings.WithTo(predeploys.L1BlockAddr),
		bindings.WithTest(t))

	gasPriceOracle := bindings.NewBindings[bindings.GasPriceOracle](
		bindings.WithClient(l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()),
		bindings.WithTo(predeploys.GasPriceOracleAddr),
		bindings.WithTest(t))

	originalScalar, err := contractio.Read(systemConfig.OperatorFeeScalar(), t.Ctx())
	t.Require().NoError(err)
	originalConstant, err := contractio.Read(systemConfig.OperatorFeeConstant(), t.Ctx())
	t.Require().NoError(err)

	return &OperatorFee{
		commonImpl:       commonFromT(t),
		l1Client:         l1EL,
		l2Network:        l2Network,
		systemConfig:     systemConfig,
		l1Block:          l1Block,
		gasPriceOracle:   gasPriceOracle,
		originalScalar:   originalScalar,
		originalConstant: originalConstant,
	}
}

func (of *OperatorFee) CheckCompatibility() bool {
	_, err := contractio.Read(of.systemConfig.OperatorFeeScalar(), of.ctx)
	if err != nil {
		of.t.Skipf("Operator fee methods not available in devstack: %v", err)
		return false
	}
	return true
}

func (of *OperatorFee) GetSystemOwner() *EOA {
	systemOwnerKey := devkeys.SystemConfigOwner.Key(of.l2Network.ChainID().ToBig())
	return NewKey(of.t, of.l2Network.Escape().Keys().Secret(systemOwnerKey)).User(of.l1Client)
}

func (of *OperatorFee) SetOperatorFee(scalar uint32, constant uint64) {
	systemOwner := of.GetSystemOwner()

	_, err := contractio.Write(
		of.systemConfig.SetOperatorFeeScalars(scalar, constant),
		of.ctx,
		systemOwner.Plan())
	of.require.NoError(err)

	of.t.Logf("Set operator fee on L1: scalar=%d, constant=%d", scalar, constant)
}

func (of *OperatorFee) WaitForL2SyncWithCurrentL1State() {
	// Read current L1 values
	l1Scalar, err := contractio.Read(of.systemConfig.OperatorFeeScalar(), of.ctx)
	of.require.NoError(err)
	l1Constant, err := contractio.Read(of.systemConfig.OperatorFeeConstant(), of.ctx)
	of.require.NoError(err)

	// Wait for L2 to sync with current L1 values
	of.WaitForL2Sync(l1Scalar, l1Constant)
}

func (of *OperatorFee) WaitForL2Sync(expectedScalar uint32, expectedConstant uint64) {
	of.require.Eventually(func() bool {
		scalar, err := contractio.Read(of.l1Block.OperatorFeeScalar(), of.ctx)
		if err != nil {
			return false
		}
		constant, err := contractio.Read(of.l1Block.OperatorFeeConstant(), of.ctx)
		if err != nil {
			return false
		}

		return scalar == expectedScalar && constant == expectedConstant
	}, 2*time.Minute, 5*time.Second, "L2 operator fee parameters did not sync within 2 minutes")
}

func (of *OperatorFee) VerifyL2Config(expectedScalar uint32, expectedConstant uint64) {
	scalar, err := contractio.Read(of.l1Block.OperatorFeeScalar(), of.ctx)
	of.require.NoError(err)
	of.require.Equal(expectedScalar, scalar)

	constant, err := contractio.Read(of.l1Block.OperatorFeeConstant(), of.ctx)
	of.require.NoError(err)
	of.require.Equal(expectedConstant, constant)
}

func (of *OperatorFee) ValidateTransactionFees(from *EOA, to *EOA, amount *big.Int, expectedScalar uint32, expectedConstant uint64) OperatorFeeValidationResult {
	vaultBefore, err := from.el.stackEL().EthClient().BalanceAt(of.ctx, predeploys.OperatorFeeVaultAddr, nil)
	of.require.NoError(err)

	tx := from.Transfer(to.Address(), eth.WeiBig(amount))
	receipt, err := tx.Included.Eval(of.ctx)
	of.require.NoError(err)
	of.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	blockHash := receipt.BlockHash
	blockRef, err := from.el.stackEL().EthClient().BlockRefByHash(of.ctx, blockHash)
	of.require.NoError(err)
	isJovian := of.l2Network.IsForkActive(rollup.Jovian, blockRef.Time)

	vaultAfter, err := from.el.stackEL().EthClient().BalanceAt(of.ctx, predeploys.OperatorFeeVaultAddr, nil)
	of.require.NoError(err)

	vaultIncrease := new(big.Int).Sub(vaultAfter, vaultBefore)

	var expectedOperatorFee *big.Int
	if expectedScalar == 0 && expectedConstant == 0 {
		expectedOperatorFee = big.NewInt(0)
	} else {
		isJovianinGPO, err := contractio.Read(of.gasPriceOracle.IsJovian(), of.ctx)
		of.require.NoError(err)

		operatorFee := new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), big.NewInt(int64(expectedScalar)))
		if isJovian {
			of.require.Equal(isJovianinGPO, true)
			// Jovian formula: (gasUsed * operatorFeeScalar * 100) + operatorFeeConstant
			operatorFee.Mul(operatorFee, big.NewInt(100))
		} else {
			of.require.Equal(isJovianinGPO, false)
			// Isthmus formula: (gasUsed * operatorFeeScalar / 1e6) + operatorFeeConstant
			operatorFee.Div(operatorFee, big.NewInt(1000000))
		}
		operatorFee.Add(operatorFee, big.NewInt(int64(expectedConstant)))
		expectedOperatorFee = operatorFee
	}

	// Use Cmp for big.Int comparison to avoid representation issues
	of.require.Equal(0, expectedOperatorFee.Cmp(vaultIncrease),
		"operator fee vault balance mismatch: expected %s, got %s",
		expectedOperatorFee.String(), vaultIncrease.String())

	actualTotalFee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
	if receipt.L1Fee != nil {
		actualTotalFee.Add(actualTotalFee, receipt.L1Fee)
	}

	if expectedScalar != 0 || expectedConstant != 0 {
		of.require.NotNil(receipt.OperatorFeeScalar)
		of.require.NotNil(receipt.OperatorFeeConstant)

		of.require.Equal(expectedScalar, uint32(*receipt.OperatorFeeScalar))
		of.require.Equal(expectedConstant, *receipt.OperatorFeeConstant)
	}

	return OperatorFeeValidationResult{
		TransactionReceipt:   receipt,
		ExpectedOperatorFee:  expectedOperatorFee,
		ActualTotalFee:       actualTotalFee,
		VaultBalanceIncrease: vaultIncrease,
	}
}

func (of *OperatorFee) RestoreOriginalConfig() {
	of.SetOperatorFee(of.originalScalar, of.originalConstant)
}

func RunOperatorFeeTest(t devtest.T, l2Chain *L2Network, l1EL *L1ELNode, funderL1, funderL2 *Funder) {
	fundAmount := eth.OneTenthEther
	alice := funderL2.NewFundedEOA(fundAmount)
	alice.WaitForBalance(fundAmount)
	bob := funderL2.NewFundedEOA(eth.ZeroWei)

	operatorFee := NewOperatorFee(t, l2Chain, l1EL)
	operatorFee.CheckCompatibility()
	systemOwner := operatorFee.GetSystemOwner()
	funderL1.FundAtLeast(systemOwner, fundAmount)

	// First, ensure L2 is synced with current L1 state before starting tests
	t.Log("Ensuring L2 is synced with current L1 state...")
	operatorFee.WaitForL2SyncWithCurrentL1State()

	testCases := []struct {
		name     string
		scalar   uint32
		constant uint64
	}{
		{"ZeroFees", 0, 0},
		{"NonZeroFees", 300, 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t devtest.T) {
			operatorFee.SetOperatorFee(tc.scalar, tc.constant)
			operatorFee.WaitForL2Sync(tc.scalar, tc.constant)
			operatorFee.VerifyL2Config(tc.scalar, tc.constant)

			result := operatorFee.ValidateTransactionFees(alice, bob, big.NewInt(1000), tc.scalar, tc.constant)

			t.Log("Test completed successfully:",
				"testCase", tc.name,
				"gasUsed", result.TransactionReceipt.GasUsed,
				"actualTotalFee", result.ActualTotalFee.String(),
				"expectedOperatorFee", result.ExpectedOperatorFee.String(),
				"vaultBalanceIncrease", result.VaultBalanceIncrease.String())
		})
	}

	operatorFee.RestoreOriginalConfig()
}
