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

type EcotoneFees struct {
	commonImpl
	l2Network *L2Network
}

type EcotoneFeesValidationResult struct {
	TransactionReceipt *types.Receipt
	L1Fee              *big.Int
	L2Fee              *big.Int
	BaseFee            *big.Int
	PriorityFee        *big.Int
	TotalFee           *big.Int
	VaultBalances      VaultBalances
	WalletBalanceDiff  *big.Int
	TransferAmount     *big.Int
}

type VaultBalances struct {
	BaseFeeVault   *big.Int
	L1FeeVault     *big.Int
	SequencerVault *big.Int
	OperatorVault  *big.Int
}

func NewEcotoneFees(t devtest.T, l2Network *L2Network) *EcotoneFees {
	return &EcotoneFees{
		commonImpl: commonFromT(t),
		l2Network:  l2Network,
	}
}

func (ef *EcotoneFees) ValidateTransaction(from *EOA, to *EOA, amount *big.Int) EcotoneFeesValidationResult {
	client := ef.l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()

	startBalance := from.GetBalance()
	vaultsBefore := ef.getVaultBalances(client)

	tx := from.Transfer(to.Address(), eth.WeiBig(amount))
	receipt, err := tx.Included.Eval(ef.ctx)
	ef.require.NoError(err)
	ef.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Get block info for base fee information
	blockInfo, err := client.InfoByHash(ef.ctx, receipt.BlockHash)
	ef.require.NoError(err)

	endBalance := from.GetBalance()
	vaultsAfter := ef.getVaultBalances(client)
	vaultIncreases := ef.calculateVaultIncreases(vaultsBefore, vaultsAfter)

	// In Ecotone, L1 fee includes both base fee and blob base fee components
	l1Fee := vaultIncreases.L1FeeVault // Use actual vault increase as the source of truth

	// Calculate receipt-based fees for validation
	receiptBaseFee := new(big.Int).Mul(blockInfo.BaseFee(), big.NewInt(int64(receipt.GasUsed)))
	receiptL2Fee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))

	// Calculate L2 fees from vault increases
	baseFee := vaultIncreases.BaseFeeVault       // Use actual vault increase as the source of truth
	priorityFee := vaultIncreases.SequencerVault // Use actual vault increase as the source of truth
	l2Fee := new(big.Int).Add(baseFee, priorityFee)

	// Total fee is the sum of all vault increases (excluding OperatorVault which should be zero in Ecotone)
	totalFee := new(big.Int).Add(vaultIncreases.BaseFeeVault, vaultIncreases.L1FeeVault)
	totalFee.Add(totalFee, vaultIncreases.SequencerVault)

	walletBalanceDiff := new(big.Int).Sub(startBalance.ToBig(), endBalance.ToBig())
	walletBalanceDiff.Sub(walletBalanceDiff, amount)

	// Validate total balance first to ensure all fees are accounted for
	ef.validateTotalBalance(walletBalanceDiff, totalFee, vaultIncreases)

	// Then validate individual fee components
	ef.validateFeeDistribution(l1Fee, baseFee, priorityFee, vaultIncreases)
	ef.validateEcotoneFeatures(receipt, l1Fee)
	ef.validateReceiptFees(receipt, l1Fee, baseFee, l2Fee, receiptBaseFee, receiptL2Fee)

	return EcotoneFeesValidationResult{
		TransactionReceipt: receipt,
		L1Fee:              l1Fee,
		L2Fee:              l2Fee,
		BaseFee:            baseFee,
		PriorityFee:        priorityFee,
		TotalFee:           totalFee,
		VaultBalances:      vaultIncreases,
		WalletBalanceDiff:  walletBalanceDiff,
		TransferAmount:     amount,
	}
}

func (ef *EcotoneFees) getVaultBalances(client apis.EthClient) VaultBalances {
	baseFee := ef.getBalance(client, predeploys.BaseFeeVaultAddr)
	l1Fee := ef.getBalance(client, predeploys.L1FeeVaultAddr)
	sequencer := ef.getBalance(client, predeploys.SequencerFeeVaultAddr)
	operator := ef.getBalance(client, predeploys.OperatorFeeVaultAddr)

	return VaultBalances{
		BaseFeeVault:   baseFee,
		L1FeeVault:     l1Fee,
		SequencerVault: sequencer,
		OperatorVault:  operator,
	}
}

func (ef *EcotoneFees) getBalance(client apis.EthClient, addr common.Address) *big.Int {
	balance, err := client.BalanceAt(ef.ctx, addr, nil)
	ef.require.NoError(err)
	return balance
}

func (ef *EcotoneFees) calculateVaultIncreases(before, after VaultBalances) VaultBalances {
	return VaultBalances{
		BaseFeeVault:   new(big.Int).Sub(after.BaseFeeVault, before.BaseFeeVault),
		L1FeeVault:     new(big.Int).Sub(after.L1FeeVault, before.L1FeeVault),
		SequencerVault: new(big.Int).Sub(after.SequencerVault, before.SequencerVault),
		OperatorVault:  new(big.Int).Sub(after.OperatorVault, before.OperatorVault),
	}
}

func (ef *EcotoneFees) validateFeeDistribution(l1Fee, baseFee, priorityFee *big.Int, vaults VaultBalances) {
	ef.require.True(l1Fee.Sign() >= 0, "L1 fee must be non-negative")
	ef.require.True(baseFee.Sign() > 0, "Base fee must be positive")
	ef.require.True(priorityFee.Sign() >= 0, "Priority fee must be non-negative")

	ef.require.Equal(l1Fee, vaults.L1FeeVault, "L1 fee must match L1FeeVault increase")
	ef.require.Equal(baseFee, vaults.BaseFeeVault, "Base fee must match BaseFeeVault increase")
	ef.require.Equal(priorityFee, vaults.SequencerVault, "Priority fee must match SequencerFeeVault increase")

	// In Ecotone, operator fees should not exist (introduced in Isthmus)
	ef.require.Equal(vaults.OperatorVault.Cmp(big.NewInt(0)), 0,
		"Operator vault increase must be zero in Ecotone (operator fees introduced in Isthmus)")
}

func (ef *EcotoneFees) validateTotalBalance(walletDiff *big.Int, totalFee *big.Int, vaults VaultBalances) {
	// In Ecotone, only BaseFeeVault, L1FeeVault, and SequencerVault should have increases
	totalVaultIncrease := new(big.Int).Add(vaults.BaseFeeVault, vaults.L1FeeVault)
	totalVaultIncrease.Add(totalVaultIncrease, vaults.SequencerVault)

	ef.require.Equal(walletDiff, totalFee, "Wallet balance difference must equal total fees")
	ef.require.Equal(totalVaultIncrease, totalFee, "Total vault increases must equal total fees")
}

func (ef *EcotoneFees) validateEcotoneFeatures(receipt *types.Receipt, l1Fee *big.Int) {
	ef.require.NotNil(receipt.L1Fee, "L1 fee should be present in Ecotone")
	ef.require.True(l1Fee.Cmp(big.NewInt(0)) > 0, "L1 fee should be greater than 0 in Ecotone")
	ef.require.Greater(receipt.GasUsed, uint64(20000), "Gas used should be reasonable for transfer")
	ef.require.Less(receipt.GasUsed, uint64(50000), "Gas used should not be excessive")
	ef.require.Greater(receipt.EffectiveGasPrice.Uint64(), uint64(0), "Effective gas price should be > 0")
}

func (ef *EcotoneFees) validateReceiptFees(receipt *types.Receipt, l1Fee, vaultBaseFee, vaultL2Fee, receiptBaseFee, receiptL2Fee *big.Int) {
	// Check that receipt's L1Fee matches the vault increase
	if receipt.L1Fee != nil {
		ef.require.Equal(receipt.L1Fee, l1Fee, "Receipt L1Fee must match L1FeeVault increase")
	}

	// Sanity check: Receipt-calculated fees should match vault-based fees
	ef.require.Equal(receiptBaseFee, vaultBaseFee,
		"Receipt-calculated base fee (block.BaseFee * gasUsed) must match BaseFeeVault increase")
	ef.require.Equal(receiptL2Fee, vaultL2Fee,
		"Receipt-calculated L2 fee (effectiveGasPrice * gasUsed) must match L2 vault increases (BaseFee + SequencerFee)")

	// Validate receipt-based calculations are positive
	ef.require.True(receiptBaseFee.Sign() > 0, "Receipt-based base fee must be positive")
	ef.require.True(receiptL2Fee.Sign() > 0, "Receipt-based L2 fee must be positive")

	// The effective gas price should be consistent with the calculated L2 fee
	ef.require.Equal(receiptL2Fee.Cmp(receiptBaseFee) >= 0, true,
		"Receipt L2 fee (effectiveGasPrice * gasUsed) should be >= base fee")
}

func (ef *EcotoneFees) LogResults(result EcotoneFeesValidationResult) {
	ef.log.Info("Comprehensive Ecotone fees validation completed",
		"gasUsed", result.TransactionReceipt.GasUsed,
		"effectiveGasPrice", result.TransactionReceipt.EffectiveGasPrice,
		"l1Fee", result.L1Fee,
		"l2Fee", result.L2Fee,
		"baseFee", result.BaseFee,
		"priorityFee", result.PriorityFee,
		"totalFee", result.TotalFee,
		"baseFeeVault", result.VaultBalances.BaseFeeVault,
		"l1FeeVault", result.VaultBalances.L1FeeVault,
		"sequencerVault", result.VaultBalances.SequencerVault,
		"operatorVault", result.VaultBalances.OperatorVault,
		"walletBalanceDiff", result.WalletBalanceDiff,
		"transferAmount", result.TransferAmount)
}
