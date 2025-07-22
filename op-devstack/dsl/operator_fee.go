package dsl

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/core/types"
)

type OperatorFee struct {
	commonImpl

	l1Client     *L1ELNode
	l2Network    *L2Network
	systemConfig bindings.SystemConfig
	l1Block      bindings.L1Block

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

	vaultAfter, err := from.el.stackEL().EthClient().BalanceAt(of.ctx, predeploys.OperatorFeeVaultAddr, nil)
	of.require.NoError(err)

	vaultIncrease := new(big.Int).Sub(vaultAfter, vaultBefore)

	var expectedOperatorFee *big.Int
	if expectedScalar == 0 && expectedConstant == 0 {
		expectedOperatorFee = big.NewInt(0)
	} else {
		operatorFee := new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), big.NewInt(int64(expectedScalar)))
		operatorFee.Div(operatorFee, big.NewInt(1000000))
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
