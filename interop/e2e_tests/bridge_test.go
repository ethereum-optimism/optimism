package e2e_tests

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestInteropL2CDM(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change from chain A's perspective
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

func TestInteropL2SB(t *testing.T) {
	testSuite := createE2ETestSuite(t)

	// wait for the first storage root of chain B to change from chain A's perspective
	var oldStorageRoot common.Hash
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		oldStorageRoot = testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
		return oldStorageRoot != common.Hash{}, nil
	}))

	// Deploy WETH9 on L1
	l1Opts, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, testSuite.OpSysB.Cfg.L1ChainIDBig())
	require.NoError(t, err)
	weth9Address, tx, WETH9, err := bindings.DeployWETH9(l1Opts, testSuite.OpSysB.Clients["l1"])
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["l1"], tx.Hash())
	require.NoError(t, err)
	t.Log("Deployed WETH9 on L1")

	// Deploy WETH9 on Chain B (manual deployment since factory uses existing bridge)
	optsB, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, big.NewInt(int64(testSuite.ChainIdB)))
	require.NoError(t, err)
	l2TokenAddrB, tx, l2TokenB, err := bindings.DeployInteropOptimismMintableERC20(optsB, testSuite.OpSysB.Clients["sequencer"], weth9Address, "WETH", "WETH", 18)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	t.Log("Deployed WETH9 on Chain B")

	// Deploy WETH9 on Chain A (manual based deployment since factory uses existing bridge)
	optsA, err := bind.NewKeyedTransactorWithChainID(testSuite.OpCfg.Secrets.Alice, big.NewInt(int64(testSuite.ChainIdA)))
	require.NoError(t, err)
	l2TokenAddrA, tx, l2TokenA, err := bindings.DeployInteropOptimismMintableERC20(optsA, testSuite.OpSysA.Clients["sequencer"], weth9Address, "WETH", "WETH", 18)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	t.Log("Deployed WETH9 on Chain A")

	require.Equal(t, l2TokenAddrB, l2TokenAddrA, "L2 Token Addresses should match")

	// Mint WETH on L1 and bridge 100 units to Chain B
	l1Opts.Value = big.NewInt(params.Ether)
	tx, err = WETH9.Fallback(l1Opts, []byte{})
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["l1"], tx.Hash())
	require.NoError(t, err)

	l1Opts.Value = nil
	tx, err = WETH9.Approve(l1Opts, testSuite.OpCfg.L1Deployments.L1StandardBridgeProxy, new(big.Int).SetUint64(math.MaxUint64))
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["l1"], tx.Hash())
	require.NoError(t, err)

	l1Sb, err := bindings.NewL1StandardBridge(testSuite.OpCfg.L1Deployments.L1StandardBridgeProxy, testSuite.OpSysB.Clients["l1"])
	require.NoError(t, err)
	tx, err = transactions.PadGasEstimate(l1Opts, 1.1, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return l1Sb.BridgeERC20(opts, weth9Address, l2TokenAddrB, big.NewInt(100), 100000, []byte{})
	})
	require.NoError(t, err)

	depRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["l1"], tx.Hash())
	require.NoError(t, err)
	depositTx, err := derive.UnmarshalDepositLogEvent(depRec.Logs[3]) // should be the fourth log after WETH9 & L1StandardBridge events
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], types.NewTx(depositTx).Hash())
	require.NoError(t, err)

	l2Balance, err := l2TokenB.BalanceOf(&bind.CallOpts{}, optsB.From)
	require.NoError(t, err)
	require.Equal(t, int64(100), l2Balance.Int64())
	t.Log("Minted & Bridged WETH9 to Chain B")

	// Interop bridge WETH9 from B To A (50 of 100 units)
	l2Sb, err := bindings.NewInteropL2StandardBridge(predeploys.InteropL2StandardBridgeAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)
	tx, err = l2Sb.BridgeERC20To(optsB, common.BigToHash(big.NewInt(int64(testSuite.ChainIdA))), l2TokenAddrB, weth9Address, optsA.From, big.NewInt(50), 200_000, nil)
	require.NoError(t, err)

	outbox, err := bindings.NewCrossL2Outbox(predeploys.CrossL2OutboxAddr, testSuite.OpSysB.Clients["sequencer"])
	require.NoError(t, err)
	msgRec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysB.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	num := msgRec.BlockNumber.Uint64()
	msgPassIter, err := outbox.FilterMessagePassed(&bind.FilterOpts{Start: num, End: &num, Context: context.Background()}, nil, nil, nil)
	require.NoError(t, err)
	require.True(t, msgPassIter.Next())
	t.Log("Passed Message Bridging WETH9 From B")

	// Relay Message on A
	require.NoError(t, wait.For(context.Background(), time.Second/2, func() (bool, error) {
		return testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB) != oldStorageRoot, nil
	}))

	outboxRoot := testSuite.PostieA.OutboxStorageRoot(testSuite.ChainIdB)
	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, testSuite.OpSysA.Clients["sequencer"])
	require.NoError(t, err)
	tx, err = inbox.RunCrossL2Message(optsA,
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
	rec, err := wait.ForReceiptOK(context.Background(), testSuite.OpSysA.Clients["sequencer"], tx.Hash())
	require.NoError(t, err)
	t.Log("Relayed message on A")

	for _, log := range rec.Logs {
		t.Log(log.Address)
	}

	// Balance should be 50 on both A and B (50 of 100 bridged from B->A)
	l2Balance, err = l2TokenB.BalanceOf(&bind.CallOpts{}, optsB.From)
	require.NoError(t, err)
	require.Equal(t, int64(50), l2Balance.Int64())

	l2Balance, err = l2TokenA.BalanceOf(&bind.CallOpts{}, optsA.From)
	require.NoError(t, err)
	require.Equal(t, int64(50), l2Balance.Int64())

	t.Log("Funds bridged")
}
