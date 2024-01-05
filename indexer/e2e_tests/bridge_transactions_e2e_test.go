package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	e2etest_utils "github.com/ethereum-optimism/optimism/indexer/e2e_tests/utils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/stretchr/testify/require"
)

func TestE2EBridgeTransactionsOptimismPortalDeposits(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)

	bobAddr := testSuite.OpCfg.Secrets.Addresses().Bob
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	// attach 1 ETH to the deposit and random calldata
	calldata := []byte{byte(1), byte(2), byte(3)}
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = big.NewInt(params.Ether)

	// In the same deposit transaction, transfer, 0.5ETH to Bob. We do this to ensure we're only indexing
	// bridged funds from the source address versus any transferred value to a recipient in the same L2 transaction
	depositTx, err := optimismPortal.DepositTransaction(l1Opts, bobAddr, big.NewInt(params.Ether/2), 100_000, false, calldata)
	require.NoError(t, err)
	depositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)

	depositInfo, err := e2etest_utils.ParseDepositInfo(depositReceipt)
	require.NoError(t, err)

	depositL2TxHash := types.NewTx(depositInfo.DepositTx).Hash()

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	deposit, err := testSuite.DB.BridgeTransactions.L1TransactionDeposit(depositInfo.DepositTx.SourceHash)
	require.NoError(t, err)
	require.NotNil(t, deposit)
	require.Equal(t, depositL2TxHash, deposit.L2TransactionHash)
	require.Equal(t, uint64(100_000), deposit.GasLimit.Uint64())
	require.Equal(t, uint64(params.Ether), deposit.Tx.Amount.Uint64())
	require.Equal(t, aliceAddr, deposit.Tx.FromAddress)
	require.Equal(t, bobAddr, deposit.Tx.ToAddress)
	require.ElementsMatch(t, calldata, deposit.Tx.Data)

	event, err := testSuite.DB.ContractEvents.L1ContractEvent(deposit.InitiatedL1EventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, event.TransactionHash, depositTx.Hash())

	// NOTE: The indexer does not track deposit inclusion as it's apart of the block derivation process.
	// If this changes, we'd like to test for this here.
}

func TestE2EBridgeTransactionsL2ToL1MessagePasserWithdrawal(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)
	l2ToL1MessagePasser, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, testSuite.L2Client)
	require.NoError(t, err)

	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	// attach 1 ETH to the withdrawal and random calldata
	calldata := []byte{byte(1), byte(2), byte(3)}
	l2Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L2ChainIDBig())
	require.NoError(t, err)
	l2Opts.Value = big.NewInt(params.Ether)

	// Ensure L1 has enough funds for the withdrawal by depositing an equal amount into the OptimismPortal
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = l2Opts.Value
	depositTx, err := optimismPortal.Receive(l1Opts)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)

	withdrawTx, err := l2ToL1MessagePasser.InitiateWithdrawal(l2Opts, aliceAddr, big.NewInt(100_000), calldata)
	require.NoError(t, err)
	withdrawReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, withdrawTx.Hash())
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.BridgeProcessor.LastL2Header
		return l2Header != nil && l2Header.Number.Uint64() >= withdrawReceipt.BlockNumber.Uint64(), nil
	}))

	msgPassed, err := withdrawals.ParseMessagePassed(withdrawReceipt)
	require.NoError(t, err)
	withdrawalHash, err := withdrawals.WithdrawalHash(msgPassed)
	require.NoError(t, err)

	withdraw, err := testSuite.DB.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
	require.NoError(t, err)
	require.NotNil(t, withdraw)
	require.Equal(t, msgPassed.Nonce.Uint64(), withdraw.Nonce.Uint64())
	require.Equal(t, uint64(100_000), withdraw.GasLimit.Uint64())
	require.Equal(t, uint64(params.Ether), withdraw.Tx.Amount.Uint64())
	require.Equal(t, aliceAddr, withdraw.Tx.FromAddress)
	require.Equal(t, aliceAddr, withdraw.Tx.ToAddress)
	require.ElementsMatch(t, calldata, withdraw.Tx.Data)

	event, err := testSuite.DB.ContractEvents.L2ContractEvent(withdraw.InitiatedL2EventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, event.TransactionHash, withdrawTx.Hash())

	// Test Withdrawal Proven
	require.Nil(t, withdraw.ProvenL1EventGUID)
	require.Nil(t, withdraw.FinalizedL1EventGUID)

	withdrawParams, proveReceipt := op_e2e.ProveWithdrawal(t, *testSuite.OpCfg, testSuite.OpSys, "sequencer", testSuite.OpCfg.Secrets.Alice, withdrawReceipt)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastFinalizedL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= proveReceipt.BlockNumber.Uint64(), nil
	}))

	withdraw, err = testSuite.DB.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
	require.NoError(t, err)
	require.NotNil(t, withdraw.ProvenL1EventGUID)

	proveEvent, err := testSuite.DB.ContractEvents.L1ContractEvent(*withdraw.ProvenL1EventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, proveEvent.TransactionHash, proveReceipt.TxHash)

	// Test Withdrawal Finalized
	require.Nil(t, withdraw.FinalizedL1EventGUID)

	finalizeReceipt := op_e2e.FinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.L1Client, testSuite.OpCfg.Secrets.Alice, proveReceipt, withdrawParams)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastFinalizedL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= finalizeReceipt.BlockNumber.Uint64(), nil
	}))

	withdraw, err = testSuite.DB.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
	require.NoError(t, err)
	require.NotNil(t, withdraw.FinalizedL1EventGUID)
	require.NotNil(t, withdraw.Succeeded)
	require.True(t, *withdraw.Succeeded)

	finalizedEvent, err := testSuite.DB.ContractEvents.L1ContractEvent(*withdraw.FinalizedL1EventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, finalizedEvent.TransactionHash, finalizeReceipt.TxHash)
}

func TestE2EBridgeTransactionsL2ToL1MessagePasserFailedWithdrawal(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l2ToL1MessagePasser, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, testSuite.L2Client)
	require.NoError(t, err)

	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	// Try to withdraw 1 ETH from L2 without any corresponding deposits on L1
	l2Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L2ChainIDBig())
	require.NoError(t, err)
	l2Opts.Value = big.NewInt(params.Ether)

	withdrawTx, err := l2ToL1MessagePasser.InitiateWithdrawal(l2Opts, aliceAddr, big.NewInt(100_000), nil)
	require.NoError(t, err)
	withdrawReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, withdrawTx.Hash())
	require.NoError(t, err)

	msgPassed, err := withdrawals.ParseMessagePassed(withdrawReceipt)
	require.NoError(t, err)
	withdrawalHash, err := withdrawals.WithdrawalHash(msgPassed)
	require.NoError(t, err)

	// Prove&Finalize withdrawal
	_, finalizeReceipt := op_e2e.ProveAndFinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.OpSys, "sequencer", testSuite.OpCfg.Secrets.Alice, withdrawReceipt)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastFinalizedL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= finalizeReceipt.BlockNumber.Uint64(), nil
	}))

	// Withdrawal registered but marked as unsuccessful
	withdraw, err := testSuite.DB.BridgeTransactions.L2TransactionWithdrawal(withdrawalHash)
	require.NoError(t, err)
	require.NotNil(t, withdraw.Succeeded)
	require.False(t, *withdraw.Succeeded)
}
