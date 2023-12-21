package e2e_tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/indexer/node"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

func TestPostieStorageRootUpdates(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	// initiate an message on chain B
	// NOTE: the destination chain does not matter for now as postie will update for any change
	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, err := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	require.NoError(t, err)
	senderOpts.Value = big.NewInt(params.Ether / 2)

	tx, err := outbox.InitiateMessage(senderOpts, common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))), senderAddr, big.NewInt(25_000), []byte{})
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	// wait for a changed root
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	clnt := node.FromRPCClient(testSuite.OpSysB.RawClients["sequencer"], node.NewMetrics(metrics.NewRegistry(), ""))
	root, err := clnt.StorageHash(predeploys.CrossL2OutboxAddr, nil)
	require.NoError(t, err)
	require.Equal(t, root, testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB))

	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	includedRoot, err := inbox.Roots(&bind.CallOpts{}, common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))), root)
	require.NoError(t, err)
	require.True(t, includedRoot)
}

func TestPostieInboxRelay(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	// Transfer 0.5 ETH from Bob's account from Chain B -> A
	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, _ := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	senderOpts.Value = big.NewInt(params.Ether / 2)
	tx, err := outbox.InitiateMessage(senderOpts, common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))), senderAddr, big.NewInt(25_000), []byte{})
	require.NoError(t, err)

	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	require.Len(t, msgRec.Logs, 1, "expecting a MessagePassed log event")

	// Get the MessagePassed event, so we can get the message-root easily,
	// without re-implementing the logic that computes it.
	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{
		Start:   num,
		End:     &num,
		Context: context.Background(),
	}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())

	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	// Relay this message onto chain A
	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
	t.Logf("outbox root: %s", outboxRoot)

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	tx, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
			From:        senderAddr,
			To:          senderAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        []byte{},
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
}

// This test is attempting to mint a Gov token on Chain A
// which will revert. The assertions made are:
// - The CrossL2MessageRelayed event says the msg target execution was unsuccessful
// - Even though the msg target reverted, the msg cannot be replayed
func TestPostieInboxFailedExecutionReplay(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	// Transfer 0.5 ETH from Bob's account from Chain B -> A
	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, _ := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	senderOpts.Value = big.NewInt(params.Ether / 2)
	// mint(senderAddr, 1)
	txInput := []byte(fmt.Sprintf("0x40c10f19000000000000000000000000%s0000000000000000000000000000000000000000000000000000000000000001", senderAddr))
	tx, err := outbox.InitiateMessage(
		senderOpts,
		common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
		predeploys.GovernanceTokenAddr,
		big.NewInt(25_000),
		txInput,
	)
	require.NoError(t, err)

	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	require.Len(t, msgRec.Logs, 1, "expecting a MessagePassed log event")

	// Get the MessagePassed event, so we can get the message-root easily,
	// without re-implementing the logic that computes it.
	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{
		Start:   num,
		End:     &num,
		Context: context.Background(),
	}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())

	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	// Relay this message onto chain A
	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
	t.Logf("outbox root: %s", outboxRoot)

	// ** Send ETH to the inbox such that it can perform the relay **
	postieOpts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, big.NewInt(int64(testSuite.ChainIdA)))
	postieOpts.Value = big.NewInt(params.Ether)
	require.NoError(t, err)
	tx, err = inbox.Receive(postieOpts)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	// Setup a listener for CrossL2MessageRelayed events, and assert
	// CrossL2MessageRelayed.success is false
	sink := make(chan *bindings.CrossL2InboxCrossL2MessageRelayed)
	opts := &bind.WatchOpts{Start: nil, Context: context.Background()}
	subscription, err := inbox.WatchCrossL2MessageRelayed(opts, sink, [][32]byte{})
	require.NoError(t, err)
	go func() {
		for event := range sink {
			require.False(t, event.Success)
		}
	}()
	defer subscription.Unsubscribe()

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	tx, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
			From:        senderAddr,
			To:          predeploys.GovernanceTokenAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        txInput,
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	_, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
			From:        senderAddr,
			To:          predeploys.GovernanceTokenAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        txInput,
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.ErrorContains(t, err, "CrossL2Inbox: message has already been consumed")
}

// This test asserts that if a msg target execution reverts due to "SafeCall: Not enough gas" error
// from the SafeCall.callWithMinGas call, that it can be replayed and will succeed given enough gas
func TestPostieInboxSafeCallRevertReplay(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	// Transfer 0.5 ETH from Bob's account from Chain B -> A
	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, _ := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	senderOpts.Value = big.NewInt(params.Ether / 2)
	tx, err := outbox.InitiateMessage(
		senderOpts,
		common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
		senderAddr,
		big.NewInt(25_000),
		[]byte{},
	)
	require.NoError(t, err)

	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	require.Len(t, msgRec.Logs, 1, "expecting a MessagePassed log event")

	// Get the MessagePassed event, so we can get the message-root easily,
	// without re-implementing the logic that computes it.
	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{
		Start:   num,
		End:     &num,
		Context: context.Background(),
	}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())

	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	// Relay this message onto chain A
	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
	t.Logf("outbox root: %s", outboxRoot)

	// ** Send ETH to the inbox such that it can perform the relay **
	postieOpts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, big.NewInt(int64(testSuite.ChainIdA)))
	postieOpts.Value = big.NewInt(params.Ether)
	require.NoError(t, err)
	tx, err = inbox.Receive(postieOpts)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	senderOpts.GasLimit = 200_000
	tx, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
			From:        senderAddr,
			To:          senderAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        []byte{},
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.NoError(t, err)

	_, err = wait.ForReceiptFail(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	// Fetching the tx execution trace, manually parsing out the revert reason
	// and asserting it matches "SafeCall: Not enough gas"
	var result map[string]interface{}
	err = testSuite.OpSysA.Clients["sequencer"].Client().CallContext(context.Background(), &result, "debug_traceTransaction", tx.Hash())
	require.NoError(t, err)
	returnValue, ok := result["returnValue"].(string)
	require.True(t, ok)
	stringType, err := abi.NewType("string", "string", nil)
	require.NoError(t, err)
	arguments := abi.Arguments{
		{
			Type: stringType,
		},
	}
	encodedData, err := arguments.Pack("SafeCall: Not enough gas")
	require.NoError(t, err)
	hexEncodedData := hex.EncodeToString(encodedData)
	require.Contains(t, returnValue, hexEncodedData)

	// Replay the msg and assert it executes without error
	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	tx, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))),
			From:        senderAddr,
			To:          senderAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        []byte{},
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
}

// Asserting that a message cannot be relayed on a chain if the _msg.targetChain does
// not match block.chainid
func TestPostieInboxWrongChainId(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	// Transfer 0.5 ETH from Bob's account from Chain B -> Chain ID 42
	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, _ := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	senderOpts.Value = big.NewInt(params.Ether / 2)
	arbitraryChainID := big.NewInt(int64(42))
	tx, err := outbox.InitiateMessage(senderOpts, common.BigToHash(arbitraryChainID), senderAddr, big.NewInt(25_000), []byte{})
	require.NoError(t, err)

	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	require.Len(t, msgRec.Logs, 1, "expecting a MessagePassed log event")

	// Get the MessagePassed event, so we can get the message-root easily,
	// without re-implementing the logic that computes it.
	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{
		Start:   num,
		End:     &num,
		Context: context.Background(),
	}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())

	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	// Relay this message onto chain A
	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
	t.Logf("outbox root: %s", outboxRoot)

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	_, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       big.NewInt(0), // first message
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: common.BigToHash(arbitraryChainID),
			From:        senderAddr,
			To:          senderAddr,
			GasLimit:    big.NewInt(25_000),
			Data:        []byte{},
			Value:       big.NewInt(params.Ether / 2),
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.ErrorContains(t, err, "CrossL2Inbox: _msg.targetChain doesn't match block.chainid")
}
