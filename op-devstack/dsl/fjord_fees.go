package dsl

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
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
	CoinbaseDiff        *big.Int
}

func NewFjordFees(t devtest.T, l2Network *L2Network) *FjordFees {
	return &FjordFees{
		commonImpl: commonFromT(t),
		l2Network:  l2Network,
	}
}

// ValidateTransaction validates the transaction and returns the validation result
func (ff *FjordFees) ValidateTransaction(from *EOA, to *EOA, amount *big.Int) FjordFeesValidationResult {
	client := ff.l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()

	startBalance := from.GetBalance()
	vaultsBefore := ff.getVaultBalances(client)
	coinbaseStartBalance := ff.getCoinbaseBalance(client)

	tx := from.Transfer(to.Address(), eth.WeiBig(amount))
	receipt, err := tx.Included.Eval(ff.ctx)
	ff.require.NoError(err)
	ff.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	endBalance := from.GetBalance()
	vaultsAfter := ff.getVaultBalances(client)
	vaultIncreases := ff.calculateVaultIncreases(vaultsBefore, vaultsAfter)
	coinbaseEndBalance := ff.getCoinbaseBalance(client)
	coinbaseDiff := new(big.Int).Sub(coinbaseEndBalance, coinbaseStartBalance)

	l1Fee := big.NewInt(0)
	if receipt.L1Fee != nil {
		l1Fee = receipt.L1Fee
	}

	block, err := client.InfoByHash(ff.ctx, receipt.BlockHash)
	ff.require.NoError(err)

	baseFee := new(big.Int).Mul(block.BaseFee(), big.NewInt(int64(receipt.GasUsed)))
	totalGasFee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
	priorityFee := new(big.Int).Sub(totalGasFee, baseFee)

	l2Fee := new(big.Int).Set(priorityFee)

	operatorFee := vaultIncreases.OperatorVault

	ff.validateVaultIncreaseFees(l2Fee, baseFee, priorityFee, l1Fee, operatorFee, coinbaseDiff, vaultsAfter, vaultsBefore)

	totalFee := new(big.Int).Add(l1Fee, l2Fee)
	totalFee.Add(totalFee, baseFee)
	totalFee.Add(totalFee, operatorFee)

	walletBalanceDiff := new(big.Int).Sub(startBalance.ToBig(), endBalance.ToBig())
	walletBalanceDiff.Sub(walletBalanceDiff, amount)

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
		CoinbaseDiff:        coinbaseDiff,
	}
}

// getVaultBalances gets the balances of the vaults
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

// getBalance gets the balance of an address
func (ff *FjordFees) getBalance(client apis.EthClient, addr common.Address) *big.Int {
	balance, err := client.BalanceAt(ff.ctx, addr, nil)
	ff.require.NoError(err)
	return balance
}

// calculateVaultIncreases calculates the increases in the vaults
func (ff *FjordFees) calculateVaultIncreases(before, after VaultBalances) VaultBalances {
	return VaultBalances{
		BaseFeeVault:   new(big.Int).Sub(after.BaseFeeVault, before.BaseFeeVault),
		L1FeeVault:     new(big.Int).Sub(after.L1FeeVault, before.L1FeeVault),
		SequencerVault: new(big.Int).Sub(after.SequencerVault, before.SequencerVault),
		OperatorVault:  new(big.Int).Sub(after.OperatorVault, before.OperatorVault),
	}
}

// validateFjordFeatures validates that the features of the Fjord transaction are correct
func (ff *FjordFees) validateFjordFeatures(receipt *types.Receipt, l1Fee *big.Int) (uint64, *big.Int) {
	ff.require.NotNil(receipt.L1Fee, "L1 fee should be present in Fjord")
	ff.require.True(l1Fee.Cmp(big.NewInt(0)) > 0, "L1 fee should be greater than 0 in Fjord")

	client := ff.l2Network.inner.L2ELNode(match.FirstL2EL).EthClient()

	_, txs, err := client.InfoAndTxsByHash(ff.ctx, receipt.BlockHash)
	ff.require.NoError(err)

	var signedTx *types.Transaction
	for _, tx := range txs {
		if tx.Hash() == receipt.TxHash {
			signedTx = tx
			break
		}
	}
	ff.require.NotNil(signedTx, "should find the signed transaction")

	unsignedTx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     signedTx.Nonce(),
		To:        signedTx.To(),
		Value:     signedTx.Value(),
		Gas:       signedTx.Gas(),
		GasFeeCap: signedTx.GasFeeCap(),
		GasTipCap: signedTx.GasTipCap(),
		Data:      signedTx.Data(),
	})

	txUnsigned, err := unsignedTx.MarshalBinary()
	ff.require.NoError(err)
	txSigned, err := signedTx.MarshalBinary()
	ff.require.NoError(err)

	fastLzSizeUnsigned := uint64(types.FlzCompressLen(txUnsigned) + 68) // overhead used by the original test
	fastLzSizeSigned := uint64(types.FlzCompressLen(txSigned))

	// Validate that FastLZ compression produces reasonable results
	ff.require.Greater(fastLzSizeUnsigned, uint64(0), "FastLZ size should be positive")
	ff.require.Greater(fastLzSizeSigned, uint64(0), "FastLZ size should be positive")

	txLenGPO := len(txUnsigned) + 68
	flzUpperBound := uint64(txLenGPO + txLenGPO/255 + 16)
	ff.require.LessOrEqual(fastLzSizeUnsigned, flzUpperBound, "Compressed size should not exceed upper bound")

	signedUpperBound := uint64(len(txSigned) + len(txSigned)/255 + 16)
	ff.require.LessOrEqual(fastLzSizeSigned, signedUpperBound, "Compressed size should not exceed upper bound")

	receiptL1Fee := receipt.L1Fee
	if receiptL1Fee == nil {
		ff.t.Logf("L1 fee is nil in receipt, skipping L1 fee validation")
		return fastLzSizeSigned, nil
	}

	expectedFee, err := CalculateFjordL1Cost(ff.ctx, client, signedTx.RollupCostData(), receipt.BlockHash)
	ff.require.NoError(err, "should calculate L1 fee")

	ff.require.Equalf(expectedFee, receiptL1Fee, "Calculated L1 fee should match receipt L1 fee (expected=%s actual=%s)", expectedFee.String(), receiptL1Fee.String())

	ff.require.Equalf(expectedFee, receipt.L1Fee, "L1 fee in receipt must be correct (expected=%s actual=%s)", expectedFee.String(), receipt.L1Fee.String())

	return fastLzSizeSigned, expectedFee
}

// validateFeeDistribution validates that the fees are distributed correctly to the vaults
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

// validateTotalBalance validates that the total balance of the wallet and the vaults is correct
func (ff *FjordFees) validateTotalBalance(walletDiff *big.Int, totalFee *big.Int, vaults VaultBalances) {
	totalVaultIncrease := new(big.Int).Add(vaults.BaseFeeVault, vaults.L1FeeVault)
	totalVaultIncrease.Add(totalVaultIncrease, vaults.SequencerVault)
	totalVaultIncrease.Add(totalVaultIncrease, vaults.OperatorVault)

	ff.require.Equal(walletDiff, totalFee, "Wallet balance difference must equal total fees")
	ff.require.Equal(totalVaultIncrease, totalFee, "Total vault increases must equal total fees")
}

// getCoinbaseBalance gets the balance of the coinbase address (block miner/sequencer)
func (ff *FjordFees) getCoinbaseBalance(client apis.EthClient) *big.Int {
	block, err := client.InfoByLabel(ff.ctx, "latest")
	ff.require.NoError(err, "should get latest block")

	coinbase := block.Coinbase()
	balance, err := client.BalanceAt(ff.ctx, coinbase, nil)
	ff.require.NoError(err, "should get coinbase balance")
	return balance
}

// validateVaultIncreaseFees validates that the fees are distributed correctly to the vaults
func (ff *FjordFees) validateVaultIncreaseFees(
	l2Fee, baseFee, priorityFee, l1Fee, operatorFee, coinbaseDiff *big.Int,
	vaultsAfter, vaultsBefore VaultBalances) {

	ff.require.Equal(l2Fee, coinbaseDiff, "L2 fee must equal coinbase difference (coinbase is always sequencer fee vault)")

	vaultsIncrease := ff.calculateVaultIncreases(vaultsBefore, vaultsAfter)
	ff.require.Equal(baseFee, vaultsIncrease.BaseFeeVault, "base fee must match BaseFeeVault increase")

	ff.require.Equal(priorityFee, vaultsIncrease.SequencerVault, "priority fee must match SequencerFeeVault increase")

	ff.require.Equal(l1Fee, vaultsIncrease.L1FeeVault, "L1 fee must match L1FeeVault increase")

	ff.require.Equal(operatorFee, vaultsIncrease.OperatorVault, "operator fee must match OperatorFeeVault increase")

	ff.t.Logf("Comprehensive fee validation passed:")
	ff.t.Logf("  L2 Fee: %s (coinbase diff: %s)", l2Fee, coinbaseDiff)
	ff.t.Logf("  Base Fee: %s (vault increase: %s)", baseFee, vaultsIncrease.BaseFeeVault)
	ff.t.Logf("  Priority Fee: %s (vault increase: %s)", priorityFee, vaultsIncrease.SequencerVault)
	ff.t.Logf("  L1 Fee: %s (vault increase: %s)", l1Fee, vaultsIncrease.L1FeeVault)
	ff.t.Logf("  Operator Fee: %s (vault increase: %s)", operatorFee, vaultsIncrease.OperatorVault)
}

// FindSignedTransactionFromReceipt finds the signed transaction from a receipt and block
func FindSignedTransactionFromReceipt(ctx context.Context, client apis.EthClient, receipt *types.Receipt) (*types.Transaction, error) {
	_, txs, err := client.InfoAndTxsByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, err
	}

	for _, tx := range txs {
		if tx.Hash() == receipt.TxHash {
			return tx, nil
		}
	}
	return nil, fmt.Errorf("signed transaction not found for hash %s", receipt.TxHash)
}

// CreateUnsignedTransactionFromSigned creates an unsigned transaction from a signed one
func CreateUnsignedTransactionFromSigned(signedTx *types.Transaction) (*types.Transaction, error) {
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:     signedTx.Nonce(),
		To:        signedTx.To(),
		Value:     signedTx.Value(),
		Gas:       signedTx.Gas(),
		GasFeeCap: signedTx.GasFeeCap(),
		GasTipCap: signedTx.GasTipCap(),
		Data:      signedTx.Data(),
	}), nil
}

// ReadGasPriceOracleL1FeeAt reads the L1 fee from GasPriceOracle for an unsigned transaction
// evaluated against a specific L2 block hash.
func ReadGasPriceOracleL1FeeAt(ctx context.Context, client apis.EthClient, gpo *bindings.GasPriceOracle, txUnsigned []byte, blockHash common.Hash) (*big.Int, error) {
	overrideBlockOpt := func(ptx *txplan.PlannedTx) {
		ptx.AgainstBlock.Fn(func(ctx context.Context) (eth.BlockInfo, error) {
			return client.InfoByHash(ctx, blockHash)
		})
	}
	result, err := contractio.Read(gpo.GetL1Fee(txUnsigned), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}
	return result.ToBig(), nil
}

// ReadGasPriceOracleL1FeeUpperBoundAt reads the L1 fee upper bound for a tx length pinned to a block hash.
func ReadGasPriceOracleL1FeeUpperBoundAt(ctx context.Context, client apis.EthClient, gpo *bindings.GasPriceOracle, txLen int, blockHash common.Hash) (*big.Int, error) {
	overrideBlockOpt := func(ptx *txplan.PlannedTx) {
		ptx.AgainstBlock.Fn(func(ctx context.Context) (eth.BlockInfo, error) {
			return client.InfoByHash(ctx, blockHash)
		})
	}
	result, err := contractio.Read(gpo.GetL1FeeUpperBound(big.NewInt(int64(txLen))), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}
	return result.ToBig(), nil
}

// ValidateL1FeeMatches checks that the calculated L1 fee matches the actual receipt L1 fee
func ValidateL1FeeMatches(t devtest.T, calculatedFee, receiptFee *big.Int) {
	require := t.Require()
	require.NotNil(receiptFee, "L1 fee should be present in receipt")
	require.Equalf(calculatedFee.Uint64(), receiptFee.Uint64(), "L1 fee mismatch (expected=%d actual=%d)", calculatedFee.Uint64(), receiptFee.Uint64())
}

// CalculateFjordL1Cost calculates L1 cost using Fjord formula with block-specific L1 state
func CalculateFjordL1Cost(ctx context.Context, client apis.EthClient, rollupCostData types.RollupCostData, blockHash common.Hash) (*big.Int, error) {
	l1Block := bindings.NewL1Block(
		bindings.WithClient(client),
		bindings.WithTo(predeploys.L1BlockAddr),
	)

	overrideBlockOpt := func(ptx *txplan.PlannedTx) {
		ptx.AgainstBlock.Fn(func(ctx context.Context) (eth.BlockInfo, error) {
			return client.InfoByHash(ctx, blockHash)
		})
	}

	baseFeeScalar, err := contractio.Read(l1Block.BasefeeScalar(), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}
	l1BaseFee, err := contractio.Read(l1Block.Basefee(), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}
	blobBaseFeeScalar, err := contractio.Read(l1Block.BlobBaseFeeScalar(), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}
	blobBaseFee, err := contractio.Read(l1Block.BlobBaseFee(), ctx, overrideBlockOpt)
	if err != nil {
		return nil, err
	}

	costFunc := types.NewL1CostFuncFjord(
		l1BaseFee,
		blobBaseFee,
		new(big.Int).SetUint64(uint64(baseFeeScalar)),
		new(big.Int).SetUint64(uint64(blobBaseFeeScalar)))

	fee, _ := costFunc(rollupCostData)
	return fee, nil
}
