package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	e2etest_utils "github.com/ethereum-optimism/optimism/indexer/e2e_tests/utils"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/params"

	"github.com/stretchr/testify/require"
)

func TestE2EBridgeL1CrossDomainMessenger(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	l1CrossDomainMessenger, err := bindings.NewL1CrossDomainMessenger(testSuite.OpCfg.L1Deployments.L1CrossDomainMessengerProxy, testSuite.L1Client)
	require.NoError(t, err)

	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	// Attach 1ETH and random calldata to the sent messages
	calldata := []byte{byte(1), byte(2), byte(3)}
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpCfg.L1ChainIDBig())
	require.NoError(t, err)
	l1Opts.Value = big.NewInt(params.Ether)

	// (1) Send the Message
	sentMsgTx, err := l1CrossDomainMessenger.SendMessage(l1Opts, aliceAddr, calldata, 100_000)
	require.NoError(t, err)
	sentMsgReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L1Client, sentMsgTx.Hash())
	require.NoError(t, err)

	depositInfo, err := e2etest_utils.ParseDepositInfo(sentMsgReceipt)
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= sentMsgReceipt.BlockNumber.Uint64(), nil
	}))

	parsedMessage, err := e2etest_utils.ParseCrossDomainMessage(sentMsgReceipt)
	require.NoError(t, err)

	// nonce for this message is zero but the current cross domain message version is 1.
	nonceBytes := [31]byte{0: byte(1)}
	nonce := new(big.Int).SetBytes(nonceBytes[:])

	sentMessage, err := testSuite.DB.BridgeMessages.L1BridgeMessage(parsedMessage.MessageHash)
	require.NoError(t, err)
	require.NotNil(t, sentMessage)
	require.NotNil(t, sentMessage.SentMessageEventGUID)
	require.Equal(t, depositInfo.DepositTx.SourceHash, sentMessage.TransactionSourceHash)
	require.Equal(t, nonce.Uint64(), sentMessage.Nonce.Uint64())
	require.Equal(t, uint64(100_000), sentMessage.GasLimit.Uint64())
	require.Equal(t, uint64(params.Ether), sentMessage.Tx.Amount.Uint64())
	require.Equal(t, aliceAddr, sentMessage.Tx.FromAddress)
	require.Equal(t, aliceAddr, sentMessage.Tx.ToAddress)
	require.ElementsMatch(t, calldata, sentMessage.Tx.Data)

	// (2) Process RelayedMessage on inclusion
	//   - We dont assert that `RelayedMessageEventGUID` is nil prior to inclusion since there isn't a
	//   a straightforward way of pausing/resuming the processors at the right time. The codepath is the
	//   same for L2->L1 messages which does check for this so we are still covered
	transaction, err := testSuite.DB.BridgeTransactions.L1TransactionDeposit(sentMessage.TransactionSourceHash)
	require.NoError(t, err)

	// wait for processor catchup
	l2DepositReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, transaction.L2TransactionHash)
	require.NoError(t, err)
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.BridgeProcessor.LastFinalizedL2Header
		return l2Header != nil && l2Header.Number.Uint64() >= l2DepositReceipt.BlockNumber.Uint64(), nil
	}))

	sentMessage, err = testSuite.DB.BridgeMessages.L1BridgeMessage(parsedMessage.MessageHash)
	require.NoError(t, err)
	require.NotNil(t, sentMessage)
	require.NotNil(t, sentMessage.RelayedMessageEventGUID)

	event, err := testSuite.DB.ContractEvents.L2ContractEvent(*sentMessage.RelayedMessageEventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, event.TransactionHash, transaction.L2TransactionHash)
}

func TestE2EBridgeL2CrossDomainMessenger(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	optimismPortal, err := bindings.NewOptimismPortal(testSuite.OpCfg.L1Deployments.OptimismPortalProxy, testSuite.L1Client)
	require.NoError(t, err)
	l2CrossDomainMessenger, err := bindings.NewL2CrossDomainMessenger(predeploys.L2CrossDomainMessengerAddr, testSuite.L2Client)
	require.NoError(t, err)

	aliceAddr := testSuite.OpCfg.Secrets.Addresses().Alice

	// Attach 1ETH and random calldata to the sent messages
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

	// (1) Send the Message
	sentMsgTx, err := l2CrossDomainMessenger.SendMessage(l2Opts, aliceAddr, calldata, 100_000)
	require.NoError(t, err)
	sentMsgReceipt, err := wait.ForReceiptOK(context.Background(), testSuite.L2Client, sentMsgTx.Hash())
	require.NoError(t, err)

	msgPassed, err := withdrawals.ParseMessagePassed(sentMsgReceipt)
	require.NoError(t, err)
	withdrawalHash, err := withdrawals.WithdrawalHash(msgPassed)
	require.NoError(t, err)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l2Header := testSuite.Indexer.BridgeProcessor.LastL2Header
		return l2Header != nil && l2Header.Number.Uint64() >= sentMsgReceipt.BlockNumber.Uint64(), nil
	}))

	parsedMessage, err := e2etest_utils.ParseCrossDomainMessage(sentMsgReceipt)
	require.NoError(t, err)

	// nonce for this message is zero but the current message version is 1.
	nonceBytes := [31]byte{0: byte(1)}
	nonce := new(big.Int).SetBytes(nonceBytes[:])

	sentMessage, err := testSuite.DB.BridgeMessages.L2BridgeMessage(parsedMessage.MessageHash)
	require.NoError(t, err)
	require.NotNil(t, sentMessage)
	require.NotNil(t, sentMessage.SentMessageEventGUID)
	require.Equal(t, withdrawalHash, sentMessage.TransactionWithdrawalHash)
	require.Equal(t, nonce.Uint64(), sentMessage.Nonce.Uint64())
	require.Equal(t, uint64(100_000), sentMessage.GasLimit.Uint64())
	require.Equal(t, uint64(params.Ether), sentMessage.Tx.Amount.Uint64())
	require.Equal(t, aliceAddr, sentMessage.Tx.FromAddress)
	require.Equal(t, aliceAddr, sentMessage.Tx.ToAddress)
	require.ElementsMatch(t, calldata, sentMessage.Tx.Data)

	// (2) Process RelayedMessage on withdrawal finalization
	require.Nil(t, sentMessage.RelayedMessageEventGUID)
	_, finalizedReceipt := op_e2e.ProveAndFinalizeWithdrawal(t, *testSuite.OpCfg, testSuite.OpSys, "sequencer", testSuite.OpCfg.Secrets.Alice, sentMsgReceipt)

	// wait for processor catchup
	require.NoError(t, wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		l1Header := testSuite.Indexer.BridgeProcessor.LastFinalizedL1Header
		return l1Header != nil && l1Header.Number.Uint64() >= finalizedReceipt.BlockNumber.Uint64(), nil
	}))

	// message is marked as relayed
	sentMessage, err = testSuite.DB.BridgeMessages.L2BridgeMessage(parsedMessage.MessageHash)
	require.NoError(t, err)
	require.NotNil(t, sentMessage)
	require.NotNil(t, sentMessage.RelayedMessageEventGUID)

	event, err := testSuite.DB.ContractEvents.L1ContractEvent(*sentMessage.RelayedMessageEventGUID)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, event.TransactionHash, finalizedReceipt.TxHash)

}
