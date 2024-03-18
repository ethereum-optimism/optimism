package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestE2EReorgDeletion(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// Conduct an L1 Deposit/Withdrawal through the standard bridge
	// which touches the CDM and root bridge contracts. Thus we'll e2e
	// test that the deletes appropriately cascades to all tables

	l1StandardBridge, err := bindings.NewL1StandardBridge(testSuite.OpCfg.L1Deployments.L1StandardBridgeProxy, testSuite.L1Client)
	require.NoError(t, err)
	l2StandardBridge, err := bindings.NewL2StandardBridge(predeploys.L2StandardBridgeAddr, testSuite.L2Client)
	require.NoError(t, err)

	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l2Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L2ChainIDBig())
	require.NoError(t, err)

	l1Opts.Value = big.NewInt(params.Ether)
	l2Opts.Value = big.NewInt(params.Ether)

	// wait for an L1 block (depends on an emitted event -- L2OO) to get indexed as a reference point prior to deletion
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		latestL1Header, err := testSuite.DB.Blocks.L1LatestBlockHeader()
		return latestL1Header != nil, err
	}))

	depositTx, err := l1StandardBridge.DepositETH(l1Opts, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	depositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L1Client, depositTx.Hash())
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= depositReceipt.BlockNumber.Uint64(), nil
	}))
	deposits, err := testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr, "", 1)
	require.NoError(t, err)
	require.Len(t, deposits.Deposits, 1)

	withdrawTx, err := l2StandardBridge.Withdraw(l2Opts, predeploys.LegacyERC20ETHAddr, l2Opts.Value, 200_000, []byte{byte(1)})
	require.NoError(t, err)
	withdrawReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, withdrawTx.Hash())
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.BridgeProcessor.LastL2Header
		return l2Header != nil && l2Header.Number.Uint64() >= withdrawReceipt.BlockNumber.Uint64(), nil
	}))
	withdrawals, err := testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr, "", 1)
	require.NoError(t, err)
	require.Len(t, withdrawals.Withdrawals, 1)

	// Stop the indexer and reorg out L1 state from the deposit transaction
	// and implicitly the derived L2 state where the withdrawal was initiated
	depositBlock, err := testSuite.DB.Blocks.L1BlockHeaderWithFilter(database.BlockHeader{Number: depositReceipt.BlockNumber})
	require.NoError(t, err)
	require.NoError(t, testSuite.Indexer.Stop(context.Background()))
	require.NoError(t, testSuite.DB.Blocks.DeleteReorgedState(deposits.Deposits[0].L1BridgeDeposit.Tx.Timestamp))

	// L1 & L2 block state deleted appropriately
	latestL1Header, err := testSuite.DB.Blocks.L2LatestBlockHeader()
	require.NoError(t, err)
	require.True(t, latestL1Header.Timestamp < depositBlock.Timestamp)
	latestL2Header, err := testSuite.DB.Blocks.L2LatestBlockHeader()
	require.NoError(t, err)
	require.True(t, latestL2Header.Timestamp < depositBlock.Timestamp)

	// Deposits/Withdrawals deletes cascade appropriately from log deletion
	deposits, err = testSuite.DB.BridgeTransfers.L1BridgeDepositsByAddress(aliceAddr, "", 1)
	require.NoError(t, err)
	require.Len(t, deposits.Deposits, 0)
	withdrawals, err = testSuite.DB.BridgeTransfers.L2BridgeWithdrawalsByAddress(aliceAddr, "", 1)
	require.NoError(t, err)
	require.Len(t, withdrawals.Withdrawals, 0)
}
