package e2e_tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestInteropL2CDM(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	// Send CDM From B -> A
	cdm, err := bindings.NewInteropL2CrossDomainMessenger(predeploys.InteropL2CrossDomainMessengerAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)
	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)

	// 0.1 ETH transfer using the cdm (with some arbitrary calldata)
	sender, senderAddr := testSuite.OpCfg.Secrets.Bob, testSuite.OpCfg.Secrets.Addresses().Bob
	senderOpts, _ := bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdB)))
	senderOpts.Value = big.NewInt(params.Ether / 10)

	calldata := []byte{1, 2, 3}
	tx, err := cdm.SendMessage(senderOpts, common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))), senderAddr, calldata, 25_000)
	require.NoError(t, err)

	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	// wait for a root update
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	// Relay the message via the inbox

	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{Start: num, End: &num, Context: context.Background()}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())
	t.Log("passed message:", msgPassIter.Event)

	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)

	// ** Send ETH to the inbox such that it can perform the relay **
	postieOpts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, big.NewInt(int64(testSuite.ChainIdA)))
	postieOpts.Value = big.NewInt(params.Ether)
	require.NoError(t, err)
	tx, err = inbox.Receive(postieOpts)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)

	senderOpts, err = bind.NewKeyedTransactorWithChainID(sender, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	tx, err = inbox.RunCrossL2Message(senderOpts,
		bindings.TypesSuperchainMessage{
			Nonce:       msgPassIter.Event.Nonce,
			SourceChain: common.BigToHash(big.NewInt(int64(testSuite.ChainIdB))),
			TargetChain: msgPassIter.Event.TargetChain,
			From:        msgPassIter.Event.From,
			To:          msgPassIter.Event.To,
			GasLimit:    msgPassIter.Event.GasLimit,
			Data:        msgPassIter.Event.Data,
			Value:       msgPassIter.Event.Value,
		},
		outboxRoot,
		genMPTProof(t, outboxRoot, msgPassIter.Event, testSuite.OpSysB.Clients["sequencer"]),
	)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
}
