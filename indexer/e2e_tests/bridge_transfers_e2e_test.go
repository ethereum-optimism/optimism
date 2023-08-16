package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	e2etest_utils "github.com/ethereum-optimism/optimism/indexer/e2e_tests/utils"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/stretchr/testify/require"
)

func TestE2EBridgeTransfersStandardBridgeETHDeposit(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l1StandardBridge, err := bindings.NewL1StandardBridge(testSuite.OpCfg.L1Deployments.L1StandardBridgeProxy, testSuite.L1Client)
	require.NoError(t, err)

	// 1 ETH transfer
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = big.NewInt(params.Ether)

	// (1) Test Deposit Initiation
	depositTx, err := l1StandardBridge.DepositETH(l1Opts, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	depositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)

	depositInfo, err := e2etest_utils.ParseDepositInfo(depositReceipt)
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	aliceDeposits, err := testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Len(t, aliceDeposits, 1)
	require.Equal(t, depositTx.Hash(), aliceDeposits[0].L1TransactionHash)
	require.Equal(t, types.NewTx(depositInfo.DepositTx).Hash(), aliceDeposits[0].L2TransactionHash)

	deposit := aliceDeposits[0].L1BridgeDeposit
	require.Equal(t, depositInfo.DepositTx.SourceHash, deposit.TransactionSourceHash)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.LocalTokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.RemoteTokenAddress)
	require.Equal(t, big.NewInt(params.Ether), deposit.Tx.Amount.Int)
	require.Equal(t, aliceAddr, deposit.Tx.FromAddress)
	require.Equal(t, aliceAddr, deposit.Tx.ToAddress)
	require.Equal(t, byte(1), deposit.Tx.Data[0])

	// StandardBridge flows through the messenger. We remove the first two significant
	// bytes of the nonce dedicated to the version. nonce == 0 (first message)
	require.NotNil(t, deposit.CrossDomainMessageHash)

	// (2) Test Deposit Finalization via CrossDomainMessenger relayed message
	depositReceipt, err = wait.ForReceiptOK(context.Background(), testSuite.L2Client, types.NewTx(depositInfo.DepositTx).Hash())
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return l2Header != nil && l2Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	crossDomainBridgeMessage, err := testSuite.DB.BridgeMessages.L1BridgeMessage(*deposit.CrossDomainMessageHash)
	require.NoError(t, err)
	require.NotNil(t, crossDomainBridgeMessage)
	require.NotNil(t, crossDomainBridgeMessage.RelayedMessageEventGUID)
}

func TestE2EBridgeTransfersOptimismPortalETHReceive(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)

	// 1 ETH transfer
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = big.NewInt(params.Ether)

	// (1) Test Deposit Initiation
	portalDepositTx, err := optimismPortal.Receive(l1Opts)
	require.NoError(t, err)
	portalDepositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L1Client, portalDepositTx.Hash())
	require.NoError(t, err)

	depositInfo, err := e2etest_utils.ParseDepositInfo(portalDepositReceipt)
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= portalDepositReceipt.BlockNumber.Uint64(), nil
	}))

	aliceDeposits, err := testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Equal(t, portalDepositTx.Hash(), aliceDeposits[0].L1TransactionHash)
	require.Equal(t, types.NewTx(depositInfo.DepositTx).Hash(), aliceDeposits[0].L2TransactionHash)

	deposit := aliceDeposits[0].L1BridgeDeposit
	require.Equal(t, depositInfo.DepositTx.SourceHash, deposit.TransactionSourceHash)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.LocalTokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.RemoteTokenAddress)
	require.Equal(t, big.NewInt(params.Ether), deposit.Tx.Amount.Int)
	require.Equal(t, aliceAddr, deposit.Tx.FromAddress)
	require.Equal(t, aliceAddr, deposit.Tx.ToAddress)
	require.Len(t, deposit.Tx.Data, 0)

	// deposit was not sent through the cross domain messenger
	require.Nil(t, deposit.CrossDomainMessageHash)

	// (2) Test Deposit Finalization
	depositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, types.NewTx(depositInfo.DepositTx).Hash())
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return l2Header != nil && l2Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	// Still nil as the withdrawal did not occur through the standard bridge
	aliceDeposits, err = testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Nil(t, aliceDeposits[0].L1BridgeDeposit.CrossDomainMessageHash)
}

func TestE2EBridgeTransfersStandardBridgeETHWithdrawal(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)
	l2StandardBridge, err := bindings.NewL2StandardBridge(predeploys.L2StandardBridgeAddr, testSuite.L2Client)
	require.NoError(t, err)

	// 1 ETH transfer
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice
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

	// (1) Test Withdrawal Initiation
	withdrawTx, err := l2StandardBridge.Withdraw(l2Opts, predeploys.LegacyERC20ETHAddr, l2Opts.Value, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	withdrawReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, withdrawTx.Hash())
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return l2Header != nil && l2Header.Number.Uint64() >= withdrawReceipt.BlockNumber.Uint64(), nil
	}))

	aliceWithdrawals, err := testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Len(t, aliceWithdrawals, 1)
	require.Equal(t, withdrawTx.Hash(), aliceWithdrawals[0].L2TransactionHash)

	msgPassed, err := withdrawals.ParseMessagePassed(withdrawReceipt)
	require.NoError(t, err)
	withdrawalHash, err := withdrawals.WithdrawalHash(msgPassed)
	require.NoError(t, err)

	withdrawal := aliceWithdrawals[0].L2BridgeWithdrawal
	require.Equal(t, withdrawalHash, withdrawal.TransactionWithdrawalHash)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.LocalTokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.RemoteTokenAddress)
	require.Equal(t, big.NewInt(params.Ether), withdrawal.Tx.Amount.Int)
	require.Equal(t, aliceAddr, withdrawal.Tx.FromAddress)
	require.Equal(t, aliceAddr, withdrawal.Tx.ToAddress)
	require.Equal(t, byte(1), withdrawal.Tx.Data[0])

	// StandardBridge flows through the messenger. We remove the first two
	// bytes of the nonce dedicated to the version. nonce == 0 (first message)
	require.NotNil(t, withdrawal.CrossDomainMessageHash)

	crossDomainBridgeMessage, err := testSuite.DB.BridgeMessages.L2BridgeMessage(*withdrawal.CrossDomainMessageHash)
	require.NoError(t, err)
	require.Nil(t, crossDomainBridgeMessage.RelayedMessageEventGUID)

	// (2) Test Withdrawal Proven/Finalized. Test the sql join queries to populate the right transaction
	require.Empty(t, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Empty(t, aliceWithdrawals[0].FinalizedL1TransactionHash)

	// wait for processor catchup
	proveReceipt, finalizeReceipt := op_e2e.ProveAndFinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.L1Client, testSuite.OpSys.Nodes["sequencer"], testSuite.OpCfg.Secrets.Alice, withdrawReceipt)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= finalizeReceipt.BlockNumber.Uint64(), nil
	}))

	aliceWithdrawals, err = testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Equal(t, proveReceipt.TxHash, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Equal(t, finalizeReceipt.TxHash, aliceWithdrawals[0].FinalizedL1TransactionHash)

	crossDomainBridgeMessage, err = testSuite.DB.BridgeMessages.L2BridgeMessage(*withdrawal.CrossDomainMessageHash)
	require.NoError(t, err)
	require.NotNil(t, crossDomainBridgeMessage)
	require.NotNil(t, crossDomainBridgeMessage.RelayedMessageEventGUID)
}

func TestE2EBridgeTransfersL2ToL1MessagePasserReceive(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)
	l2ToL1MessagePasser, err := bindings.NewOptimismPortal(predeploys.L2ToL1MessagePasserAddr, testSuite.L2Client)
	require.NoError(t, err)

	// 1 ETH transfer
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice
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

	// (1) Test Withdrawal Initiation
	l2ToL1MessagePasserWithdrawTx, err := l2ToL1MessagePasser.Receive(l2Opts)
	require.NoError(t, err)
	l2ToL1WithdrawReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, l2ToL1MessagePasserWithdrawTx.Hash())
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return l2Header != nil && l2Header.Number.Uint64() >= l2ToL1WithdrawReceipt.BlockNumber.Uint64(), nil
	}))

	aliceWithdrawals, err := testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Equal(t, l2ToL1MessagePasserWithdrawTx.Hash(), aliceWithdrawals[0].L2TransactionHash)

	msgPassed, err := withdrawals.ParseMessagePassed(l2ToL1WithdrawReceipt)
	require.NoError(t, err)
	withdrawalHash, err := withdrawals.WithdrawalHash(msgPassed)
	require.NoError(t, err)

	withdrawal := aliceWithdrawals[0].L2BridgeWithdrawal
	require.Equal(t, withdrawalHash, withdrawal.TransactionWithdrawalHash)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.LocalTokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.RemoteTokenAddress)
	require.Equal(t, big.NewInt(params.Ether), withdrawal.Tx.Amount.Int)
	require.Equal(t, aliceAddr, withdrawal.Tx.FromAddress)
	require.Equal(t, aliceAddr, withdrawal.Tx.ToAddress)
	require.Len(t, withdrawal.Tx.Data, 0)

	// withdrawal was not sent through the cross domain messenger
	require.Nil(t, withdrawal.CrossDomainMessageHash)

	// (2) Test Withdrawal Proven/Finalized. Test the sql join queries to populate the right transaction
	require.Empty(t, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Empty(t, aliceWithdrawals[0].FinalizedL1TransactionHash)

	// wait for processor catchup
	proveReceipt, finalizeReceipt := op_e2e.ProveAndFinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.L1Client, testSuite.OpSys.Nodes["sequencer"], testSuite.OpCfg.Secrets.Alice, l2ToL1WithdrawReceipt)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= finalizeReceipt.BlockNumber.Uint64(), nil
	}))

	aliceWithdrawals, err = testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Equal(t, proveReceipt.TxHash, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Equal(t, finalizeReceipt.TxHash, aliceWithdrawals[0].FinalizedL1TransactionHash)

	// Still nil as the withdrawal did not occur through the standard bridge
	require.Nil(t, aliceWithdrawals[0].L2BridgeWithdrawal.CrossDomainMessageHash)
}
