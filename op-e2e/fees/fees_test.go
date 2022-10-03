package fees

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestFees checks that L1/L2 fees are handled.
func TestFees(t *testing.T) {
	if !op_e2e.VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := op_e2e.DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.Wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: op_e2e.TransactorHDPath,
		},
	})
	require.Nil(t, err)
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	// Find gaspriceoracle contract
	gpoContract, err := bindings.NewGasPriceOracle(common.HexToAddress(predeploys.GasPriceOracle), l2Seq)
	require.Nil(t, err)

	// GPO signer
	l2opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L2ChainID)
	require.Nil(t, err)

	// Update overhead
	tx, err := gpoContract.SetOverhead(l2opts, big.NewInt(2100))
	require.Nil(t, err, "sending overhead update tx")

	receipt, err := op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "waiting for overhead update tx")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	// Update decimals
	tx, err = gpoContract.SetDecimals(l2opts, big.NewInt(6))
	require.Nil(t, err, "sending gpo update tx")

	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "waiting for gpo decimals update tx")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	// Update scalar
	tx, err = gpoContract.SetScalar(l2opts, big.NewInt(1_000_000))
	require.Nil(t, err, "sending gpo update tx")

	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "waiting for gpo scalar update tx")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	overhead, err := gpoContract.Overhead(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo overhead")
	decimals, err := gpoContract.Decimals(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo decimals")
	scalar, err := gpoContract.Scalar(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo scalar")

	require.Equal(t, overhead.Uint64(), uint64(2100), "wrong gpo overhead")
	require.Equal(t, decimals.Uint64(), uint64(6), "wrong gpo decimals")
	require.Equal(t, scalar.Uint64(), uint64(1_000_000), "wrong gpo scalar")

	// BaseFee Recipient
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	baseFeeRecipientStartBalance, err := l2Seq.BalanceAt(ctx, cfg.BaseFeeRecipient, nil)
	require.Nil(t, err)

	// L1Fee Recipient
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	l1FeeRecipientStartBalance, err := l2Seq.BalanceAt(ctx, cfg.L1FeeRecipient, nil)
	require.Nil(t, err)

	// Simple transfer from signer to random account
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	toAddr := common.Address{0xff, 0xff}
	transferAmount := big.NewInt(1_000_000_000)
	gasTip := big.NewInt(10)
	tx = types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainID,
		Nonce:     3, // Already have deposit
		To:        &toAddr,
		Value:     transferAmount,
		GasTipCap: gasTip,
		GasFeeCap: big.NewInt(200),
		Gas:       21000,
	})
	err = l2Seq.SendTransaction(context.Background(), tx)
	require.Nil(t, err, "Sending L2 tx to sequencer")

	_, err = op_e2e.WaitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on verifier")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "TX should have succeeded")

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err := l2Seq.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	coinbaseStartBalance, err := l2Seq.BalanceAt(ctx, header.Coinbase, op_e2e.SafeAddBig(header.Number, big.NewInt(-1)))
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	coinbaseEndBalance, err := l2Seq.BalanceAt(ctx, header.Coinbase, header.Number)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err := l2Seq.BalanceAt(ctx, fromAddr, header.Number)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	baseFeeRecipientEndBalance, err := l2Seq.BalanceAt(ctx, cfg.BaseFeeRecipient, header.Number)
	require.Nil(t, err)

	l1Header, err := sys.Clients["l1"].HeaderByNumber(ctx, nil)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	l1FeeRecipientEndBalance, err := l2Seq.BalanceAt(ctx, cfg.L1FeeRecipient, nil)
	require.Nil(t, err)

	// Diff fee recipient + coinbase balances
	baseFeeRecipientDiff := new(big.Int).Sub(baseFeeRecipientEndBalance, baseFeeRecipientStartBalance)
	l1FeeRecipientDiff := new(big.Int).Sub(l1FeeRecipientEndBalance, l1FeeRecipientStartBalance)
	coinbaseDiff := new(big.Int).Sub(coinbaseEndBalance, coinbaseStartBalance)

	// Tally L2 Fee
	l2Fee := gasTip.Mul(gasTip, new(big.Int).SetUint64(receipt.GasUsed))
	require.Equal(t, l2Fee, coinbaseDiff, "l2 fee mismatch")

	// Tally BaseFee
	baseFee := new(big.Int).Mul(header.BaseFee, new(big.Int).SetUint64(receipt.GasUsed))
	require.Equal(t, baseFee, baseFeeRecipientDiff, "base fee fee mismatch")

	// Tally L1 Fee
	bytes, err := tx.MarshalBinary()
	require.Nil(t, err)
	l1GasUsed := op_e2e.CalcL1GasUsed(bytes, overhead)
	divisor := new(big.Int).Exp(big.NewInt(10), decimals, nil)
	l1Fee := new(big.Int).Mul(l1GasUsed, l1Header.BaseFee)
	l1Fee = l1Fee.Mul(l1Fee, scalar)
	l1Fee = l1Fee.Div(l1Fee, divisor)
	require.Equal(t, l1Fee, l1FeeRecipientDiff, "l1 fee mismatch")

	// Tally L1 fee against GasPriceOracle
	gpoL1Fee, err := gpoContract.GetL1Fee(&bind.CallOpts{}, bytes)
	require.Nil(t, err)
	require.Equal(t, l1Fee, gpoL1Fee, "l1 fee mismatch")

	// Calculate total fee
	baseFeeRecipientDiff.Add(baseFeeRecipientDiff, coinbaseDiff)
	totalFee := new(big.Int).Add(baseFeeRecipientDiff, l1FeeRecipientDiff)
	balanceDiff := new(big.Int).Sub(startBalance, endBalance)
	balanceDiff.Sub(balanceDiff, transferAmount)
	require.Equal(t, balanceDiff, totalFee, "balances should add up")
}
