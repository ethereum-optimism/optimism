package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// TestSystemE2E sets up a L1 Geth node, 2 rollup nodes, and 2 L2 geth nodes.
// All nodes are run in process (but are the full nodes, not mocked or stubbed).
func TestSystemInteropE2E(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfigInterop(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
}

func sendDepositTx(
	t *testing.T, optimismPortalProxy common.Address, l1Client *ethclient.Client,
	l2Client *ethclient.Client, l1Opts *bind.TransactOpts, applyL2Opts DepositTxOptsFn,
	l1BlockTime uint64,
) *types.Receipt {
	l2Opts := defaultDepositTxOpts(l1Opts)
	applyL2Opts(l2Opts)

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(optimismPortalProxy, l1Client)
	require.Nil(t, err)

	// Finally send TX
	// Add 10% padding for the L1 gas limit because the estimation process can be affected by the 1559 style cost scale
	// for buying L2 gas in the portal contracts.
	tx, err := transactions.PadGasEstimate(l1Opts, 1.1, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return depositContract.DepositTransaction(opts, l2Opts.ToAddr, l2Opts.Value, l2Opts.GasLimit, l2Opts.IsCreation, l2Opts.Data)
	})
	require.Nil(t, err, "with deposit tx")

	// Wait for transaction on L1
	l1Receipt, err := geth.WaitForTransaction(tx.Hash(), l1Client, 10*time.Duration(l1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	// Wait for transaction to be included on L2
	reconstructedDep, err := derive.UnmarshalDepositLogEvent(l1Receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	// Use a long wait because the l2Client may not be configured to receive gossip from the sequencer
	// so has to wait for the batcher to submit and then import those blocks from L1.
	l2Receipt, err := geth.WaitForTransaction(tx.Hash(), l2Client, 60*time.Second)
	require.NoError(t, err)
	require.Equal(t, l2Opts.ExpectedStatus, l2Receipt.Status, "l2 transaction status")
	return l2Receipt
}

func testWithdraw(
	t *testing.T, ethPrivKey *ecdsa.PrivateKey, l1ChainID *big.Int,
	l2Verifier *ethclient.Client, l1Deployments *genesis.L1Deployments,
	l1Client *ethclient.Client, l2Sequencer *ethclient.Client, l1BlockTime uint64,
	l2ChainID *big.Int, l2BlockTime uint64, l2Node EthInstance,
) {
	// Create L1 signer
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, l1ChainID)
	require.Nil(t, err)

	// Start L2 balance
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)
	startBalance, err := l2Verifier.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Send deposit tx
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	sendDepositTx(t, l1Deployments.OptimismPortalProxy , l1Client, l2Verifier, opts, func(l2Opts *DepositTxOpts) {
		l2Opts.Value = common.Big0
	}, l1BlockTime)

	// Confirm L2 balance
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	endBalance, err := l2Verifier.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change after mint")

	// Start L2 balance for withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	startBalance, err = l2Sequencer.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	withdrawAmount := big.NewInt(500_000_000_000)
	tx, receipt := SendWithdrawalInterop(t, l2ChainID, l2Sequencer, ethPrivKey, func(opts *WithdrawalTxOpts) {
		opts.Value = withdrawAmount
		opts.VerifyOnClients(l2Verifier)
	}, l1BlockTime, l2BlockTime)

	// Verify L2 balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	header, err := l2Verifier.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	endBalance, err = l2Verifier.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Take fee into account
	diff = new(big.Int).Sub(startBalance, endBalance)
	fees := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	fees = fees.Add(fees, receipt.L1Fee)
	diff = diff.Sub(diff, fees)
	require.Equal(t, withdrawAmount, diff)

	// Take start balance on L1
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	startBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	proveReceipt, finalizeReceipt := ProveAndFinalizeWithdrawalInterop(t, l1BlockTime, l1Client, l2Node, ethPrivKey, receipt, l1Deployments, l1ChainID)

	// Verify balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	endBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Ensure that withdrawal - gas fees are added to the L1 balance
	// Fun fact, the fee is greater than the withdrawal amount
	// NOTE: The gas fees include *both* the ProveWithdrawalTransaction and FinalizeWithdrawalTransaction transactions.
	diff = new(big.Int).Sub(endBalance, startBalance)
	proveFee := new(big.Int).Mul(new(big.Int).SetUint64(proveReceipt.GasUsed), proveReceipt.EffectiveGasPrice)
	finalizeFee := new(big.Int).Mul(new(big.Int).SetUint64(finalizeReceipt.GasUsed), finalizeReceipt.EffectiveGasPrice)
	fees = new(big.Int).Add(proveFee, finalizeFee)
	withdrawAmount = withdrawAmount.Sub(withdrawAmount, fees)
	require.Equal(t, withdrawAmount, diff)
}

// TestWithdrawals checks that a deposit and then withdrawal execution succeeds on each L2. It verifies the
// balance changes on L1 and L2 and has to include gas fees in the balance checks.
// It does not check that the withdrawal can be executed prior to the end of the finality period.
func TestWithdrawalsOnL2s(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfigInterop(t)
	cfg.DeployConfigs[0].FinalizationPeriodSeconds = 2 // 2s finalization period
	cfg.DeployConfigs[1].FinalizationPeriodSeconds = 2 // 2s finalization period

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	testWithdraw(
		t, ethPrivKey, cfg.L1ChainIDBig(), sys.Clients["verifier"], cfg.L1Deployments[0],
		sys.Clients["l1"], sys.Clients["sequencer"], cfg.DeployConfigs[0].L1BlockTime,
		cfg.L2ChainIDBig(), cfg.DeployConfigs[0].L2BlockTime, sys.EthInstances["verifier"],
	)

	testWithdraw(
		t, ethPrivKey, cfg.L1ChainIDBig(), sys.Clients["verifier_2"], cfg.L1Deployments[1],
		sys.Clients["l1_2"], sys.Clients["sequencer_2"], cfg.DeployConfigs[1].L1BlockTime,
		cfg.L2ChainIDBig_2(), cfg.DeployConfigs[1].L2BlockTime, sys.EthInstances["verifier_2"],
	)
}
