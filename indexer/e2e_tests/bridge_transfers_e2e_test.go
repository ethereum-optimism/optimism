package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	e2etest_utils "github.com/ethereum-optimism/optimism/indexer/e2e_tests/utils"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/stretchr/testify/require"
)

func TestE2EBridgeTransfersStandardBridgeETHDeposit(t *testing.T) {
	testSuite := createE2ETestSuite(t)
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	l1StandardBridge, err := bindings.NewL1StandardBridge(testSuite.OpCfg.L1Deployments.L1StandardBridgeProxy, testSuite.L1Client)
	require.NoError(t, err)

	// 1 ETH transfer
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = big.NewInt(params.Ether)

	// Pause the L2Processor so that we can test for finalization separately. A pause is
	// required since deposit inclusion is apart of the L2 block derivation process
	testSuite.Indexer.L2Processor.PauseForTest()

	// (1) Test Deposit Initiation
	depositTx, err := l1StandardBridge.DepositETH(l1Opts, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	depositReceipt, err := utils.WaitReceiptOK(testCtx, testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)

	depositInfo, err := e2etest_utils.ParseDepositInfo(depositReceipt)
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, utils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	aliceDeposits, err := testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr)
	require.NoError(t, err)
	require.Len(t, aliceDeposits, 1)
	require.Equal(t, depositTx.Hash(), aliceDeposits[0].L1TransactionHash)
	require.Empty(t, aliceDeposits[0].FinalizedL2TransactionHash)

	deposit := aliceDeposits[0].L1BridgeDeposit
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.L1TokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, deposit.TokenPair.L2TokenAddress)
	require.Equal(t, big.NewInt(params.Ether), deposit.Tx.Amount.Int)
	require.Equal(t, aliceAddr, deposit.Tx.FromAddress)
	require.Equal(t, aliceAddr, deposit.Tx.ToAddress)
	require.Equal(t, byte(1), deposit.Tx.Data[0])

	// (2) Test Deposit Finalization
	require.Nil(t, deposit.FinalizedL2EventGUID)
	testSuite.Indexer.L2Processor.ResumeForTest()

	// wait for the l2 processor to catch this deposit in the derivation process
	depositReceipt, err = utils.WaitReceiptOK(testCtx, testSuite.L2Client, types.NewTx(depositInfo.DepositTx).Hash())
	require.NoError(t, err)
	require.NoError(t, utils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
		return l2Header != nil && l2Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))

	aliceDeposits, err = testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr)
	require.NoError(t, err)
	require.NotNil(t, aliceDeposits[0].L1BridgeDeposit.FinalizedL2EventGUID)
	require.Equal(t, types.NewTx(depositInfo.DepositTx).Hash(), aliceDeposits[0].FinalizedL2TransactionHash)
}

func TestE2EBridgeTransfersStandardBridgeETHWithdrawal(t *testing.T) {
	testSuite := createE2ETestSuite(t)
	testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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
	_, err = utils.WaitReceiptOK(testCtx, testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)

	// (1) Test Withdrawal Initiation
	withdrawTx, err := l2StandardBridge.Withdraw(l2Opts, predeploys.LegacyERC20ETHAddr, l2Opts.Value, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	withdrawReceipt, err := utils.WaitReceiptOK(testCtx, testSuite.L2Client, withdrawTx.Hash())
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, utils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
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
	require.Equal(t, withdrawalHash, withdrawal.WithdrawalHash)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.L1TokenAddress)
	require.Equal(t, predeploys.LegacyERC20ETHAddr, withdrawal.TokenPair.L2TokenAddress)
	require.Equal(t, big.NewInt(params.Ether), withdrawal.Tx.Amount.Int)
	require.Equal(t, aliceAddr, withdrawal.Tx.FromAddress)
	require.Equal(t, aliceAddr, withdrawal.Tx.ToAddress)
	require.Equal(t, byte(1), withdrawal.Tx.Data[0])

	// (2) Test Withdrawal Proven/Finalized. Test the sql join queries to populate the right transaction
	require.Nil(t, withdrawal.ProvenL1EventGUID)
	require.Nil(t, withdrawal.FinalizedL1EventGUID)
	require.Empty(t, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Empty(t, aliceWithdrawals[0].FinalizedL1TransactionHash)

	// wait for processor catchup
	proveReceipt, finalizeReceipt := op_e2e.ProveAndFinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.L1Client, testSuite.OpSys.Nodes["sequencer"], testSuite.OpCfg.Secrets.Alice, withdrawReceipt)
	require.NoError(t, utils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
		return l1Header != nil && l1Header.Number.Uint64() >= finalizeReceipt.BlockNumber.Uint64(), nil
	}))

	aliceWithdrawals, err = testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr)
	require.NoError(t, err)
	require.NotNil(t, aliceWithdrawals[0].L2BridgeWithdrawal.ProvenL1EventGUID)
	require.NotNil(t, aliceWithdrawals[0].L2BridgeWithdrawal.FinalizedL1EventGUID)
	require.Equal(t, proveReceipt.TxHash, aliceWithdrawals[0].ProvenL1TransactionHash)
	require.Equal(t, finalizeReceipt.TxHash, aliceWithdrawals[0].FinalizedL1TransactionHash)
}
