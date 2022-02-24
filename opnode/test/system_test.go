package test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/l2os"
	"github.com/ethereum-optimism/optimistic-specs/l2os/bindings/l2oo"
	"github.com/ethereum-optimism/optimistic-specs/l2os/txmgr"
	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/deposit"
	"github.com/ethereum-optimism/optimistic-specs/opnode/internal/testlog"
	rollupNode "github.com/ethereum-optimism/optimistic-specs/opnode/node"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/stretchr/testify/require"
)

func getGenesisHash(client *ethclient.Client) common.Hash {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	block, err := client.BlockByNumber(ctx, common.Big0)
	if err != nil {
		panic(err)
	}
	return block.Hash()
}

func endpoint(cfg *node.Config) string {
	return fmt.Sprintf("ws://%v", cfg.WSEndpoint())
}

// TestSystemE2E sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that L1 deposits are reflected on L2.
// All nodes are run in process (but are the full nodes, not mocked or stubbed).
func TestSystemE2E(t *testing.T) {
	log.Root().SetHandler(log.DiscardHandler()) // Comment this out to see geth l1/l2 logs

	const l2OutputHDPath = "m/44'/60'/0'/0/3"

	// System Config
	cfg := &systemConfig{
		mnemonic: "squirrel green gallery layer logic title habit chase clog actress language enrich body plate fun pledge gap abuse mansion define either blast alien witness",
		l1: gethConfig{
			nodeConfig: &node.Config{
				Name:   "l1geth",
				WSHost: "127.0.0.1",
				WSPort: 9090,
			},
			ethConfig: &ethconfig.Config{
				NetworkId: 900,
			},
		},
		l2: gethConfig{
			nodeConfig: &node.Config{
				Name:    "l2geth",
				DataDir: "",
				IPCPath: "",
				WSHost:  "127.0.0.1",
				WSPort:  9091,
			},
			ethConfig: &ethconfig.Config{
				NetworkId: 901,
			},
		},
		premine: map[string]int{
			"m/44'/60'/0'/0/0": 10000000,
			"m/44'/60'/0'/0/1": 10000000,
			"m/44'/60'/0'/0/2": 10000000,
			l2OutputHDPath:     10000000,
		},
		cliqueSigners:           []string{"m/44'/60'/0'/0/0"},
		depositContractAddress:  "0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001",
		l1InforPredeployAddress: "0x4242424242424242424242424242424242424242",
	}
	// Create genesis & assign it to ethconfigs
	initializeGenesis(cfg)

	// Start L1
	l1Node, l1Backend, err := l1Geth(cfg)
	require.Nil(t, err)
	defer l1Node.Close()

	err = l1Node.Start()
	require.Nil(t, err)

	err = l1Backend.StartMining(1)
	require.Nil(t, err)

	l1Client, err := ethclient.Dial(endpoint(cfg.l1.nodeConfig))
	require.Nil(t, err)
	l1GenesisHash := getGenesisHash(l1Client)

	// Start L2
	l2Node, _, err := l2Geth(cfg)
	require.Nil(t, err)
	defer l2Node.Close()

	err = l2Node.Start()
	require.Nil(t, err)

	l2Client, err := ethclient.Dial(endpoint(cfg.l2.nodeConfig))
	require.Nil(t, err)
	l2GenesisHash := getGenesisHash(l2Client)

	// Rollup Node
	nodeCfg := &rollupNode.Config{
		L2Hash:        l2GenesisHash,
		L1Hash:        l1GenesisHash,
		L1Num:         0,
		L1NodeAddr:    endpoint(cfg.l1.nodeConfig),
		L2EngineAddrs: []string{endpoint(cfg.l2.nodeConfig)},
		Rollup: rollup.Config{
			BlockTime:            1,
			MaxSequencerTimeDiff: 10,
			SeqWindowSize:        1,
			L1ChainID:            big.NewInt(901),
			// TODO pick defaults
			FeeRecipientAddress: common.Address{0xff, 0x01},
			BatchInboxAddress:   common.Address{0xff, 0x02},
			BatchSenderAddress:  common.Address{0xff, 0x03},
		},
	}
	nodeCfg.Rollup.Genesis = nodeCfg.GetGenesis()
	node, err := rollupNode.New(context.Background(), nodeCfg, testlog.Logger(t, log.LvlTrace))
	require.Nil(t, err)

	err = node.Start(context.Background())
	require.Nil(t, err)
	defer node.Stop()

	// Deploy StateRootOracle
	l2OutputPrivKey, err := cfg.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: l2OutputHDPath,
		},
	})
	require.Nil(t, err)
	l2OutputAddr := crypto.PubkeyToAddress(l2OutputPrivKey.PublicKey)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	nonce, err := l1Client.NonceAt(ctx, l2OutputAddr, nil)
	require.Nil(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(
		l2OutputPrivKey, cfg.l1.ethConfig.Genesis.Config.ChainID,
	)
	require.Nil(t, err)
	opts.Nonce = big.NewInt(int64(nonce))

	submissionFrequency := big.NewInt(10) // 10 seconds
	l2BlockTime := big.NewInt(2)          // 2 seconds
	l2ooAddr, tx, l2OutputOracle, err := l2oo.DeployMockL2OutputOracle(
		opts, l1Client, submissionFrequency, l2BlockTime, [32]byte{}, big.NewInt(0),
	)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = txmgr.WaitMined(ctx, l1Client, tx, time.Second, 1)
	require.Nil(t, err)

	initialSroTimestamp, err := l2OutputOracle.LatestBlockTimestamp(&bind.CallOpts{})
	require.Nil(t, err)

	// L2Output Submitter
	l2OutputSubmitter, err := l2os.NewL2OutputSubmitter(l2os.Config{
		L1EthRpc:                  endpoint(cfg.l1.nodeConfig),
		L2EthRpc:                  endpoint(cfg.l2.nodeConfig),
		L2OOAddress:               l2ooAddr.String(),
		PollInterval:              5 * time.Second,
		NumConfirmations:          1,
		ResubmissionTimeout:       5 * time.Second,
		SafeAbortNonceTooLowCount: 3,
		LogLevel:                  "debug",
		Mnemonic:                  cfg.mnemonic,
		L2OutputHDPath:            l2OutputHDPath,
	}, "")
	require.Nil(t, err)

	err = l2OutputSubmitter.Start()
	require.Nil(t, err)
	defer l2OutputSubmitter.Stop()

	// Send Transaction & wait for success
	contractAddr := common.HexToAddress(cfg.depositContractAddress)
	fromAddr := common.HexToAddress("0x30ec912c5b1d14aa6d1cb9aa7a6682415c4f7eb0")

	// start balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Contract
	depositContract, err := deposit.NewDeposit(contractAddr, l1Client)
	require.Nil(t, err)

	// Signer
	ks := l1Node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	opts, err = bind.NewKeyStoreTransactorWithChainID(ks, ks.Accounts()[0], big.NewInt(int64(cfg.l1.ethConfig.NetworkId)))
	require.Nil(t, err)

	// Setup for L1 Confirmation
	watchChan := make(chan *deposit.DepositTransactionDeposited)
	watcher, err := depositContract.WatchTransactionDeposited(&bind.WatchOpts{}, watchChan, []common.Address{fromAddr}, []common.Address{fromAddr})
	require.Nil(t, err, "with watcher")
	defer watcher.Unsubscribe()

	// Setup for L2 Confirmation
	headChan := make(chan *types.Header, 100)
	l2HeadSub, err := l2Client.SubscribeNewHead(context.Background(), headChan)
	require.Nil(t, err, "with l2 head sub")
	defer l2HeadSub.Unsubscribe()

	// Finally send TX
	mintAmount := big.NewInt(1_000_000_000_000)
	tx, err = depositContract.DepositTransaction(opts, fromAddr, mintAmount, big.NewInt(1_000_000), false, nil)
	require.Nil(t, err, "with deposit tx")

	// Wait for tx to be mined on L1 (or timeout)
	select {
	case <-watchChan:
		// continue
	case err := <-watcher.Err():
		t.Fatalf("Failed on watcher channel: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for L1 tx to succeed")

	}

	// Get the L1 Block of the tx
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	receipt, err := l1Client.TransactionReceipt(ctx, tx.Hash())
	require.Nil(t, err, "Could not get transaction receipt")

	// Wait (or timeout) for that block to show up on L2
	timeoutCh := time.After(3 * time.Second)
loop:
	for {
		select {
		case head := <-headChan:
			if head.Number.Cmp(receipt.BlockNumber) >= 0 {
				break loop
			}
		case err := <-l2HeadSub.Err():
			t.Fatalf("Error in l2 head subscription: %v", err)
		case <-timeoutCh:
			t.Fatal("Timeout waiting for l2 head")
		}
	}

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err := l2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, diff, mintAmount, "Did not get expected balance change")

	// Wait for batch submitter to update L2 output oracle.
	timeoutCh = time.After(15 * time.Second)
	for {
		l2ooTimestamp, err := l2OutputOracle.LatestBlockTimestamp(&bind.CallOpts{})
		require.Nil(t, err)

		// Wait for the L2 output oracle to have been changed from the initial
		// timestamp set in the contract constructor.
		if l2ooTimestamp.Cmp(initialSroTimestamp) > 0 {
			// Retrieve the l2 output committed at this updated timestamp.
			committedL2Output, err := l2OutputOracle.L2Outputs(&bind.CallOpts{}, l2ooTimestamp)
			require.Nil(t, err)

			// Compute the committed L2 output's L2 block number.
			l2ooBlockNumber, err := l2OutputOracle.ComputeL2BlockNumber(
				&bind.CallOpts{}, l2ooTimestamp,
			)
			require.Nil(t, err)

			// Fetch the corresponding L2 block and assert the committed L2
			// output matches the block's state root.
			//
			// NOTE: This assertion will change once the L2 output format is
			// finalized.
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			l2Block, err := l2Client.BlockByNumber(ctx, l2ooBlockNumber)
			require.Nil(t, err)
			require.Equal(t, l2Block.Root(), common.Hash(committedL2Output))
			break
		}

		select {
		case <-timeoutCh:
			t.Fatalf("State root oracle not updated")
		case <-time.After(time.Second):
		}
	}
}
