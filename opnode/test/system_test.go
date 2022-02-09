package test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/contracts/deposit"
	"github.com/ethereum-optimism/optimistic-specs/opnode/internal/testlog"
	rollupNode "github.com/ethereum-optimism/optimistic-specs/opnode/node"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
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
		},
		cliqueSigners:           []string{"m/44'/60'/0'/0/0"},
		depositContractAddress:  "0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001",
		l1InforPredeployAddress: "0x4242424242424242424242424242424242424242",
	}
	// Create genesis & assign it to ethconfigs
	initializeGenesis(cfg)

	// Start L1
	l1Node, l1Backend, err := l1Geth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer l1Node.Close()
	err = l1Node.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = l1Backend.StartMining(1)
	if err != nil {
		t.Fatal(err)
	}
	l1Client, err := ethclient.Dial(endpoint(cfg.l1.nodeConfig))
	if err != nil {
		t.Fatal(err)
	}
	l1GenesisHash := getGenesisHash(l1Client)

	// Start L2
	l2Node, _, err := l2Geth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer l2Node.Close()
	err = l2Node.Start()
	if err != nil {
		t.Fatal(err)
	}
	l2Client, err := ethclient.Dial(endpoint(cfg.l2.nodeConfig))
	if err != nil {
		t.Fatal(err)
	}
	l2GenesisHash := getGenesisHash(l2Client)

	// Rollup Node
	nodeCfg := &rollupNode.Config{
		L2Hash:        l2GenesisHash,
		L1Hash:        l1GenesisHash,
		L1Num:         0,
		L1NodeAddrs:   []string{endpoint(cfg.l1.nodeConfig)},
		L2EngineAddrs: []string{endpoint(cfg.l2.nodeConfig)},
	}
	node, err := rollupNode.New(context.Background(), nodeCfg, testlog.Logger(t, log.LvlTrace))
	if err != nil {
		t.Fatalf("Failed to create the new node: %v", err)
	}
	err = node.Start()
	defer node.Stop()
	if err != nil {
		t.Fatal(err)
	}

	// Send Transaction & wait for success
	contractAddr := common.HexToAddress(cfg.depositContractAddress)
	fromAddr := common.HexToAddress("0x30ec912c5b1d14aa6d1cb9aa7a6682415c4f7eb0")

	// start balance
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Contract
	depositContract, err := deposit.NewDeposit(contractAddr, l1Client)
	if err != nil {
		t.Fatal(err)
	}

	// Signer
	ks := l1Node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	opts, err := bind.NewKeyStoreTransactorWithChainID(ks, ks.Accounts()[0], big.NewInt(int64(cfg.l1.ethConfig.NetworkId)))
	if err != nil {
		t.Fatal(err)
	}

	// Setup for L1 Confirmation
	watchChan := make(chan *deposit.DepositTransactionDeposited)
	watcher, err := depositContract.WatchTransactionDeposited(&bind.WatchOpts{}, watchChan, []common.Address{fromAddr}, []common.Address{fromAddr})
	if err != nil {
		t.Fatalf("with watcher: %v", err)
	}
	defer watcher.Unsubscribe()

	// Setup for L2 Confirmation
	headChan := make(chan *types.Header, 100)
	l2HeadSub, err := l2Client.SubscribeNewHead(context.Background(), headChan)
	if err != nil {
		t.Fatalf("with l2 head sub: %v", err)
	}
	defer l2HeadSub.Unsubscribe()

	// Finally send TX
	mintAmount := big.NewInt(1_000_000_000_000)
	tx, err := depositContract.DepositTransaction(opts, fromAddr, mintAmount, big.NewInt(1_000_000), false, nil)
	if err != nil {
		t.Fatalf("with deposit txt: %v", err)
	}

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
	if err != nil {
		t.Fatalf("Could not get tranaction receipt: %v", err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	if diff.Cmp(mintAmount) != 0 {
		t.Fatalf("Did not get expected balance change. start: %v, end: %v, mint: %v", startBalance, endBalance, mintAmount)
	}

}
