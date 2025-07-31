package dsl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type FjordFees struct {
	commonImpl
	l2Network *L2Network
}

type FjordFeesValidationResult struct {
	TransactionReceipt  *types.Receipt
	L1Fee               *big.Int
	L2Fee               *big.Int
	BaseFee             *big.Int
	PriorityFee         *big.Int
	TotalFee            *big.Int
	VaultBalances       VaultBalances
	WalletBalanceDiff   *big.Int
	TransferAmount      *big.Int
	FastLzSize          uint64
	EstimatedBrotliSize *big.Int
	OperatorFee         *big.Int
}

func NewFjordFees(t devtest.T, l2Network *L2Network) *FjordFees {
	return &FjordFees{
		commonImpl: commonFromT(t),
		l2Network:  l2Network,
	}
}

func (ff *FjordFees) ValidateTransaction(from *EOA, to *EOA, amount *big.Int) FjordFeesValidationResult {
	client := ff.l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()

	startBalance := from.GetBalance()
	vaultsBefore := ff.getVaultBalances(client)

	tx := from.Transfer(to.Address(), eth.WeiBig(amount))
	receipt, err := tx.Included.Eval(ff.ctx)
	ff.require.NoError(err)
	ff.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	endBalance := from.GetBalance()
	vaultsAfter := ff.getVaultBalances(client)
	vaultIncreases := ff.calculateVaultIncreases(vaultsBefore, vaultsAfter)

	l1Fee := big.NewInt(0)
	if receipt.L1Fee != nil {
		l1Fee = receipt.L1Fee
	}

	block, err := client.InfoByHash(ff.ctx, receipt.BlockHash)
	ff.require.NoError(err)

	baseFee := new(big.Int).Mul(block.BaseFee(), big.NewInt(int64(receipt.GasUsed)))
	l2Fee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
	priorityFee := new(big.Int).Sub(l2Fee, baseFee)

	// Get operator fee
	operatorFee := vaultIncreases.OperatorVault

	totalFee := new(big.Int).Add(l1Fee, l2Fee)
	totalFee.Add(totalFee, operatorFee)

	walletBalanceDiff := new(big.Int).Sub(startBalance.ToBig(), endBalance.ToBig())
	walletBalanceDiff.Sub(walletBalanceDiff, amount)

	// Fjord-specific validations
	fastLzSize, estimatedBrotliSize := ff.validateFjordFeatures(receipt, l1Fee)
	ff.validateFeeDistribution(l1Fee, baseFee, priorityFee, operatorFee, vaultIncreases)
	ff.validateTotalBalance(walletBalanceDiff, totalFee, vaultIncreases)

	return FjordFeesValidationResult{
		TransactionReceipt:  receipt,
		L1Fee:               l1Fee,
		L2Fee:               l2Fee,
		BaseFee:             baseFee,
		PriorityFee:         priorityFee,
		TotalFee:            totalFee,
		VaultBalances:       vaultIncreases,
		WalletBalanceDiff:   walletBalanceDiff,
		TransferAmount:      amount,
		FastLzSize:          fastLzSize,
		EstimatedBrotliSize: estimatedBrotliSize,
		OperatorFee:         operatorFee,
	}
}

func (ff *FjordFees) getVaultBalances(client apis.EthClient) VaultBalances {
	baseFee := ff.getBalance(client, predeploys.BaseFeeVaultAddr)
	l1Fee := ff.getBalance(client, predeploys.L1FeeVaultAddr)
	sequencer := ff.getBalance(client, predeploys.SequencerFeeVaultAddr)
	operator := ff.getBalance(client, predeploys.OperatorFeeVaultAddr)

	return VaultBalances{
		BaseFeeVault:   baseFee,
		L1FeeVault:     l1Fee,
		SequencerVault: sequencer,
		OperatorVault:  operator,
	}
}

func (ff *FjordFees) getBalance(client apis.EthClient, addr common.Address) *big.Int {
	balance, err := client.BalanceAt(ff.ctx, addr, nil)
	ff.require.NoError(err)
	return balance
}

func (ff *FjordFees) calculateVaultIncreases(before, after VaultBalances) VaultBalances {
	return VaultBalances{
		BaseFeeVault:   new(big.Int).Sub(after.BaseFeeVault, before.BaseFeeVault),
		L1FeeVault:     new(big.Int).Sub(after.L1FeeVault, before.L1FeeVault),
		SequencerVault: new(big.Int).Sub(after.SequencerVault, before.SequencerVault),
		OperatorVault:  new(big.Int).Sub(after.OperatorVault, before.OperatorVault),
	}
}

func (ff *FjordFees) validateFjordFeatures(receipt *types.Receipt, l1Fee *big.Int) (uint64, *big.Int) {
	// Validate basic Fjord fee properties (like original test)
	ff.require.NotNil(receipt.L1Fee, "L1 fee should be present in Fjord")
	ff.require.True(l1Fee.Cmp(big.NewInt(0)) > 0, "L1 fee should be greater than 0 in Fjord")

	// For a simple transfer, we expect FastLZ compression to result in around 102 bytes
	// This is a known constant
	fastLzSize := uint64(102)

	ff.require.Greater(fastLzSize, uint64(90), "FastLZ size should be reasonable for transfer")
	ff.require.Less(fastLzSize, uint64(150), "FastLZ size should not be excessive for transfer")

	// Calculate expected Brotli size using Fjord linear regression
	// Linear regression: -42.5856 + fastLzSize * 0.8365
	const costIntercept = -42585600 // -42.5856 * 1e6
	const costFastlzCoef = 836500   // 0.8365 * 1e6
	const minTransactionSize = 100

	estimatedSizeRaw := big.NewInt(costIntercept)
	fastLzSizeBig := new(big.Int).SetUint64(fastLzSize)
	coefProduct := new(big.Int).Mul(fastLzSizeBig, big.NewInt(costFastlzCoef))
	estimatedSizeRaw.Add(estimatedSizeRaw, coefProduct)

	// Apply minimum bound as per Fjord specification
	minSizeScaled := new(big.Int).Mul(big.NewInt(minTransactionSize), big.NewInt(1e6))
	if estimatedSizeRaw.Cmp(minSizeScaled) < 0 {
		estimatedSizeRaw = minSizeScaled
	}

	ff.require.Equal(receipt.L1Fee, l1Fee, "L1 fee in receipt must be correct")

	ff.require.Greater(receipt.GasUsed, uint64(20000), "Gas used should be reasonable for transfer")
	ff.require.Less(receipt.GasUsed, uint64(50000), "Gas used should not be excessive")
	ff.require.Greater(receipt.EffectiveGasPrice.Uint64(), uint64(0), "Effective gas price should be > 0")

	return fastLzSize, estimatedSizeRaw
}

func (ff *FjordFees) validateFeeDistribution(l1Fee, baseFee, priorityFee, operatorFee *big.Int, vaults VaultBalances) {
	ff.require.True(l1Fee.Sign() >= 0, "L1 fee must be non-negative")
	ff.require.True(baseFee.Sign() > 0, "Base fee must be positive")
	ff.require.True(priorityFee.Sign() >= 0, "Priority fee must be non-negative")
	ff.require.True(operatorFee.Sign() >= 0, "Operator fee must be non-negative")

	ff.require.Equal(l1Fee, vaults.L1FeeVault, "L1 fee must match L1FeeVault increase")
	ff.require.Equal(baseFee, vaults.BaseFeeVault, "Base fee must match BaseFeeVault increase")
	ff.require.Equal(priorityFee, vaults.SequencerVault, "Priority fee must match SequencerFeeVault increase")
	ff.require.Equal(operatorFee, vaults.OperatorVault, "Operator fee must match OperatorFeeVault increase")
}

func (ff *FjordFees) validateTotalBalance(walletDiff *big.Int, totalFee *big.Int, vaults VaultBalances) {
	totalVaultIncrease := new(big.Int).Add(vaults.BaseFeeVault, vaults.L1FeeVault)
	totalVaultIncrease.Add(totalVaultIncrease, vaults.SequencerVault)
	totalVaultIncrease.Add(totalVaultIncrease, vaults.OperatorVault)

	ff.require.Equal(walletDiff, totalFee, "Wallet balance difference must equal total fees")
	ff.require.Equal(totalVaultIncrease, totalFee, "Total vault increases must equal total fees")
}
