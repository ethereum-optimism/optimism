package withdrawals

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestWithdrawals checks that a deposit and then withdrawal execution succeeds. It verifies the
// balance changes on L1 and L2 and has to include gas fees in the balance checks.
// It does not check that the withdrawal can be executed prior to the end of the finality period.
func TestWithdrawals(t *testing.T) {
	if !op_e2e.VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := op_e2e.DefaultSystemConfig(t)
	cfg.DepositCFG.FinalizationPeriod = big.NewInt(2) // 2s finalization period

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
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

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)

	// Create L1 signer
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainID)
	require.Nil(t, err)

	// Start L2 balance
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Finally send TX
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	tx, err := depositContract.DepositTransaction(opts, fromAddr, common.Big0, 1_000_000, false, nil)
	require.Nil(t, err, "with deposit tx")

	receipt, err := op_e2e.WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	// Bind L2 Withdrawer Contract
	l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Seq)
	require.Nil(t, err, "binding withdrawer on L2")

	// Wait for deposit to arrive
	reconstructedDep, err := derive.UnmarshalDepositLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	// Confirm L2 balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change after mint")

	// Start L2 balance for withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err = l2Seq.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Intiate Withdrawal
	withdrawAmount := big.NewInt(500_000_000_000)
	l2opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L2ChainID)
	require.Nil(t, err)
	l2opts.Value = withdrawAmount
	tx, err = l2withdrawer.InitiateWithdrawal(l2opts, fromAddr, big.NewInt(21000), nil)
	require.Nil(t, err, "sending initiate withdraw tx")

	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "withdrawal initiated on L2 sequencer")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	// Verify L2 balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err := l2Verif.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err = l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Take fee into account
	diff = new(big.Int).Sub(startBalance, endBalance)
	fees := op_e2e.CalcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	diff = diff.Sub(diff, fees)
	require.Equal(t, withdrawAmount, diff)

	// Take start balance on L1
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Wait for finalization and then create the Finalized Withdrawal Transaction
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Duration(cfg.L1BlockTime)*time.Second)
	defer cancel()
	blockNumber, err := withdrawals.WaitForFinalizationPeriod(ctx, l1Client, sys.DepositContractAddr, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err = l2Verif.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	require.Nil(t, err)

	rpc, err := rpc.Dial(sys.Nodes["verifier"].WSEndpoint())
	require.Nil(t, err)
	l2client := withdrawals.NewClient(rpc)

	// Now create withdrawal
	params, err := withdrawals.FinalizeWithdrawalParameters(context.Background(), l2client, tx.Hash(), header)
	require.Nil(t, err)

	portal, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)

	opts.Value = nil
	tx, err = portal.FinalizeWithdrawalTransaction(
		opts,
		bindings.TypesWithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		},
		params.BlockNumber,
		params.OutputRootProof,
		params.WithdrawalProof,
	)

	require.Nil(t, err)

	receipt, err = op_e2e.WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "finalize withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Verify balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err = l1Client.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Ensure that withdrawal - gas fees are added to the L1 balance
	// Fun fact, the fee is greater than the withdrawal amount
	diff = new(big.Int).Sub(endBalance, startBalance)
	fees = op_e2e.CalcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	withdrawAmount = withdrawAmount.Sub(withdrawAmount, fees)
	require.Equal(t, withdrawAmount, diff)
}
