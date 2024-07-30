package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type CommonSystem interface {
	NodeClient(role string) *ethclient.Client
	RollupClient(role string) *sources.RollupClient
	Config() SystemConfig
	TestAccount(int) *ecdsa.PrivateKey
}

// TestWithdrawals checks that a deposit and then withdrawal execution succeeds. It verifies the
// balance changes on L1 and L2 and has to include gas fees in the balance checks.
// It does not check that the withdrawal can be executed prior to the end of the finality period.
func RunWithdrawalsTest(t *testing.T, sys CommonSystem) {
	t.Logf("WithdrawalsTest: running with FP == %t", e2eutils.UseFaultProofs())
	cfg := sys.Config()

	l1Client := sys.NodeClient(RoleL1)
	l2Seq := sys.NodeClient(RoleSeq)
	l2Verif := sys.NodeClient(RoleVerif)

	// Transactor Account
	ethPrivKey := sys.TestAccount(0)
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	// Create L1 signer
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Start L2 balance
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeDeposit, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	// Send deposit tx
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	t.Logf("WithdrawalsTest: depositing %v with L2 start balance %v...", mintAmount, startBalanceBeforeDeposit)
	SendDepositTx(t, cfg, l1Client, l2Verif, opts, func(l2Opts *DepositTxOpts) {
		l2Opts.Value = common.Big0
	})
	t.Log("WithdrawalsTest: waiting for balance change...")

	// Confirm L2 balance
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endBalanceAfterDeposit, err := wait.ForBalanceChange(ctx, l2Verif, fromAddr, startBalanceBeforeDeposit)
	require.NoError(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalanceAfterDeposit, startBalanceBeforeDeposit)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change after mint")

	// Start L2 balance for withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeWithdrawal, err := l2Seq.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	withdrawAmount := big.NewInt(500_000_000_000)
	t.Logf("WithdrawalsTest: sending L2 withdrawal for %v...", withdrawAmount)
	tx, receipt := SendWithdrawal(t, cfg, l2Seq, ethPrivKey, func(opts *WithdrawalTxOpts) {
		opts.Value = withdrawAmount
		opts.VerifyOnClients(l2Verif)
	})

	// Verify L2 balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	header, err := l2Verif.HeaderByNumber(ctx, receipt.BlockNumber)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.Log("WithdrawalsTest: waiting for L2 balance change...")
	endBalanceAfterWithdrawal, err := wait.ForBalanceChange(ctx, l2Seq, fromAddr, startBalanceBeforeWithdrawal)
	require.NoError(t, err)

	// Take fee into account
	diff = new(big.Int).Sub(startBalanceBeforeWithdrawal, endBalanceAfterWithdrawal)
	fees := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	fees = fees.Add(fees, receipt.L1Fee)
	diff = diff.Sub(diff, fees)
	require.Equal(t, withdrawAmount, diff)

	// Take start balance on L1
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeFinalize, err := l1Client.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	t.Log("WithdrawalsTest: ProveAndFinalizeWithdrawal...")
	proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt := ProveAndFinalizeWithdrawal(t, cfg, sys, RoleVerif, ethPrivKey, receipt)

	// Verify balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.Log("WithdrawalsTest: waiting for L1 balance change...")
	endBalanceAfterFinalize, err := wait.ForBalanceChange(ctx, l1Client, fromAddr, startBalanceBeforeFinalize)
	require.NoError(t, err)
	t.Logf("WithdrawalsTest: L1 balance changed from %v to %v", startBalanceBeforeFinalize, endBalanceAfterFinalize)

	// Ensure that withdrawal - gas fees are added to the L1 balance
	// Fun fact, the fee is greater than the withdrawal amount
	// NOTE: The gas fees include *both* the ProveWithdrawalTransaction and FinalizeWithdrawalTransaction transactions.
	diff = new(big.Int).Sub(endBalanceAfterFinalize, startBalanceBeforeFinalize)
	proveFee := new(big.Int).Mul(new(big.Int).SetUint64(proveReceipt.GasUsed), proveReceipt.EffectiveGasPrice)
	finalizeFee := new(big.Int).Mul(new(big.Int).SetUint64(finalizeReceipt.GasUsed), finalizeReceipt.EffectiveGasPrice)
	fees = new(big.Int).Add(proveFee, finalizeFee)
	if e2eutils.UseFaultProofs() {
		resolveClaimFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveClaimReceipt.GasUsed), resolveClaimReceipt.EffectiveGasPrice)
		resolveFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveReceipt.GasUsed), resolveReceipt.EffectiveGasPrice)
		fees = new(big.Int).Add(fees, resolveClaimFee)
		fees = new(big.Int).Add(fees, resolveFee)
	}
	withdrawAmount = withdrawAmount.Sub(withdrawAmount, fees)
	require.Equal(t, withdrawAmount, diff)
}
