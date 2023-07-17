package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/processor"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	e2eutils "github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestE2EBridge(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l1Client := testSuite.OpSys.Clients["l1"]
	l2Client := testSuite.OpSys.Clients["sequencer"]
	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	l1StandardBridge, err := bindings.NewL1StandardBridge(predeploys.DevL1StandardBridgeAddr, l1Client)
	require.NoError(t, err)

	_, err = bindings.NewL2StandardBridge(predeploys.L2StandardBridgeAddr, l2Client)
	require.NoError(t, err)

	_, err = bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
	require.NoError(t, err)

	t.Run("indexes ETH deposits", func(t *testing.T) {
		testCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// pause L2Processor so that we can test for finalization seperately
		testSuite.Indexer.L2Processor.PauseForTest()

		l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
		require.NoError(t, err)

		// Deposit 1 ETH (add some extra data for fun)
		l1Opts.Value = big.NewInt(params.Ether)
		tx, err := l1StandardBridge.DepositETH(l1Opts, 200_000, []byte{byte(1)})
		require.NoError(t, err)

		// (1) Test Deposit Initiation

		// wait for deposit to be included & processor catchup
		depositReceipt, err := e2eutils.WaitReceiptOK(testCtx, l1Client, tx.Hash())
		require.NoError(t, err)
		l1Height, err := l1Client.BlockNumber(testCtx)
		require.NoError(t, err)
		require.NoError(t, e2eutils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
			l1Header := testSuite.Indexer.L1Processor.LatestProcessedHeader()
			return l1Header != nil && l1Header.Number.Uint64() >= l1Height, nil
		}))

		aliceDeposits, err := testSuite.DB.Bridge.DepositsByAddress(aliceAddr)
		require.NoError(t, err)
		require.Len(t, aliceDeposits, 1)
		require.Equal(t, tx.Hash(), aliceDeposits[0].L1TransactionHash)
		require.Empty(t, aliceDeposits[0].FinalizedL2TransactionHash)

		deposit := aliceDeposits[0].Deposit
		require.Nil(t, deposit.FinalizedL2EventGUID)
		require.Equal(t, processor.EthAddress, deposit.TokenPair.L1TokenAddress)
		require.Equal(t, processor.EthAddress, deposit.TokenPair.L2TokenAddress)
		require.Equal(t, big.NewInt(params.Ether), deposit.Tx.Amount.Int)
		require.Equal(t, aliceAddr, deposit.Tx.FromAddress)
		require.Equal(t, aliceAddr, deposit.Tx.ToAddress)
		require.Equal(t, byte(1), deposit.Tx.Data[0])

		// (2) Test Deposit Finalization

		testSuite.Indexer.L2Processor.ResumeForTest()

		// finalization hash can be deterministically derived from TransactionDeposited log
		var txHash common.Hash
		for _, log := range depositReceipt.Logs {
			if len(log.Topics) == 0 || log.Topics[0] != derive.DepositEventABIHash {
				continue
			}

			depLog, err := derive.UnmarshalDepositLogEvent(log)
			require.NoError(t, err)
			tx := types.NewTx(depLog)
			txHash = tx.Hash()
		}

		// wait for the l2 processor to catch this deposit in the derivation process
		_, err = e2eutils.WaitReceiptOK(testCtx, l2Client, txHash)
		require.NoError(t, err)
		l2Height, err := l2Client.BlockNumber(testCtx)
		require.NoError(t, err)
		require.NoError(t, e2eutils.WaitFor(testCtx, 500*time.Millisecond, func() (bool, error) {
			l2Header := testSuite.Indexer.L2Processor.LatestProcessedHeader()
			return l2Header != nil && l2Header.Number.Uint64() >= l2Height, nil
		}))

		aliceDeposits, err = testSuite.DB.Bridge.DepositsByAddress(aliceAddr)
		require.NoError(t, err)
		require.Equal(t, txHash, aliceDeposits[0].FinalizedL2TransactionHash)
		require.NotNil(t, aliceDeposits[0].Deposit.FinalizedL2EventGUID)
	})

	t.Run("indexes ETH withdrawals", func(t *testing.T) {})
}
