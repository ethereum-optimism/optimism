package op_e2e

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-node/node"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/predeploy"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-proposer/rollupclient"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

// Init testing to enable test flags
var _ = func() bool {
	testing.Init()
	return true
}()

var verboseGethNodes bool

func init() {
	flag.BoolVar(&verboseGethNodes, "gethlogs", false, "Enable logs on geth nodes")
	flag.Parse()
}

// Temporary until the contract is deployed properly instead of as a pre-deploy to a specific address
var MockDepositContractAddr = common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001")

const (
	cliqueSignerHDPath = "m/44'/60'/0'/0/0"
	transactorHDPath   = "m/44'/60'/0'/0/1"
	l2OutputHDPath     = "m/44'/60'/0'/0/3"
	bssHDPath          = "m/44'/60'/0'/0/4"
	p2pSignerHDPath    = "m/44'/60'/0'/0/5"
)

var (
	batchInboxAddress = common.Address{0xff, 0x02}
)

func defaultSystemConfig(t *testing.T) SystemConfig {
	return SystemConfig{
		Mnemonic: "squirrel green gallery layer logic title habit chase clog actress language enrich body plate fun pledge gap abuse mansion define either blast alien witness",
		Premine: map[string]int{
			cliqueSignerHDPath: 10000000,
			transactorHDPath:   10000000,
			l2OutputHDPath:     10000000,
			bssHDPath:          10000000,
		},
		DepositCFG: DepositContractConfig{
			FinalizationPeriod: big.NewInt(60 * 60 * 24),
		},
		L2OOCfg: L2OOContractConfig{
			// L2 Start time is set based off of the L2 Genesis time
			SubmissionFrequency:   big.NewInt(2),
			L2BlockTime:           big.NewInt(1),
			HistoricalTotalBlocks: big.NewInt(0),
		},
		L2OutputHDPath:             l2OutputHDPath,
		BatchSubmitterHDPath:       bssHDPath,
		P2PSignerHDPath:            p2pSignerHDPath,
		DeployerHDPath:             l2OutputHDPath,
		CliqueSignerDerivationPath: cliqueSignerHDPath,
		L1InfoPredeployAddress:     derive.L1InfoPredeployAddr,
		L1BlockTime:                2,
		L1WsAddr:                   "127.0.0.1",
		L1WsPort:                   9090,
		L1ChainID:                  big.NewInt(900),
		L2ChainID:                  big.NewInt(901),
		Nodes: map[string]*rollupNode.Config{
			"verifier": {
				L1NodeAddr:    "ws://127.0.0.1:9090",
				L2EngineAddrs: []string{"ws://127.0.0.1:9091"},
				L2NodeAddr:    "ws://127.0.0.1:9091",
				L1TrustRPC:    false,
			},
			"sequencer": {
				L1NodeAddr:    "ws://127.0.0.1:9090",
				L2EngineAddrs: []string{"ws://127.0.0.1:9092"},
				L2NodeAddr:    "ws://127.0.0.1:9092",
				L1TrustRPC:    false,
				Sequencer:     true,
				// Submitter PrivKey is set in system start for rollup nodes where sequencer = true
				RPC: node.RPCConfig{
					ListenAddr: "127.0.0.1",
					ListenPort: 9093,
				},
			},
		},
		Loggers: map[string]log.Logger{
			"verifier":  testlog.Logger(t, log.LvlError).New("role", "verifier"),
			"sequencer": testlog.Logger(t, log.LvlError).New("role", "sequencer"),
		},
		RollupConfig: rollup.Config{
			BlockTime:         1,
			MaxSequencerDrift: 10,
			SeqWindowSize:     2,
			L1ChainID:         big.NewInt(900),
			L2ChainID:         big.NewInt(901),
			// TODO pick defaults
			P2PSequencerAddress: common.Address{}, // TODO configure sequencer p2p key
			FeeRecipientAddress: common.Address{0xff, 0x01},
			BatchInboxAddress:   batchInboxAddress,
			// Batch Sender address is filled out in system start
			DepositContractAddress: MockDepositContractAddr,
		},
		P2PTopology: nil, // no P2P connectivity by default
	}
}

func TestL2OutputSubmitter(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := defaultSystemConfig(t)

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]

	rollupRPCClient, err := rpc.DialContext(context.Background(), fmt.Sprintf("http://%s:%d", cfg.Nodes["sequencer"].RPC.ListenAddr, cfg.Nodes["sequencer"].RPC.ListenPort))
	require.Nil(t, err)
	rollupClient := rollupclient.NewRollupClient(rollupRPCClient)

	//  StateRootOracle is already deployed
	l2OutputOracle, err := bindings.NewL2OutputOracleCaller(sys.L2OOContractAddr, l1Client)
	require.Nil(t, err)

	initialSroTimestamp, err := l2OutputOracle.LatestBlockTimestamp(&bind.CallOpts{})
	require.Nil(t, err)

	// Wait until the second output submission from L2. The output submitter submits outputs from the
	// unsafe portion of the chain which gets reorged on startup. The sequencer has an out of date view
	// when it creates it's first block and uses and old L1 Origin. It then does not submit a batch
	// for that block and subsequently reorgs to match what the verifier derives when running the
	// reconcillation process.
	l2Verif := sys.Clients["verifier"]
	_, err = waitForBlock(big.NewInt(6), l2Verif, 10*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err)

	// Wait for batch submitter to update L2 output oracle.
	timeoutCh := time.After(15 * time.Second)
	for {
		l2ooTimestamp, err := l2OutputOracle.LatestBlockTimestamp(&bind.CallOpts{})
		require.Nil(t, err)

		// Wait for the L2 output oracle to have been changed from the initial
		// timestamp set in the contract constructor.
		if l2ooTimestamp.Cmp(initialSroTimestamp) > 0 {
			// Retrieve the l2 output committed at this updated timestamp.
			committedL2Output, err := l2OutputOracle.GetL2Output(&bind.CallOpts{}, l2ooTimestamp)
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			l2Output, err := rollupClient.OutputAtBlock(ctx, l2ooBlockNumber)
			require.Nil(t, err)
			require.Len(t, l2Output, 2)

			require.Equal(t, l2Output[1][:], committedL2Output.OutputRoot[:])
			break
		}

		select {
		case <-timeoutCh:
			t.Fatalf("State root oracle not updated")
		case <-time.After(time.Second):
		}
	}

}

// TestSystemE2E sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that L1 deposits are reflected on L2.
// All nodes are run in process (but are the full nodes, not mocked or stubbed).
func TestSystemE2E(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := defaultSystemConfig(t)

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: "m/44'/60'/0'/0/0",
		},
	})
	require.Nil(t, err)

	// Send Transaction & wait for success
	fromAddr := common.HexToAddress("0x30ec912c5b1d14aa6d1cb9aa7a6682415c4f7eb0")

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)
	l1Node := sys.nodes["l1"]

	// Create signer
	ks := l1Node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	opts, err := bind.NewKeyStoreTransactorWithChainID(ks, ks.Accounts()[0], cfg.L1ChainID)
	require.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Finally send TX
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	tx, err := depositContract.DepositTransaction(opts, fromAddr, common.Big0, 1_000_000, false, nil)
	require.Nil(t, err, "with deposit tx")

	receipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	reconstructedDep, err := derive.UnmarshalLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = waitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, diff, mintAmount, "Did not get expected balance change")

	// Submit TX to L2 sequencer node
	toAddr := common.Address{0xff, 0xff}
	tx = types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainID,
		Nonce:     1, // Already have deposit
		To:        &toAddr,
		Value:     big.NewInt(1_000_000_000),
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(200),
		Gas:       21000,
	})
	err = l2Seq.SendTransaction(context.Background(), tx)
	require.Nil(t, err, "Sending L2 tx to sequencer")

	_, err = waitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	receipt, err = waitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on verifier")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "TX should have succeeded")

	// Verify blocks match after batch submission on verifiers and sequencers
	verifBlock, err := l2Verif.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	seqBlock, err := l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	require.Equal(t, verifBlock.Hash(), seqBlock.Hash(), "Verifier and sequencer blocks not the same after including a batch tx")
}

func TestMintOnRevertedDeposit(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}
	cfg := defaultSystemConfig(t)

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Verif := sys.Clients["verifier"]

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)
	l1Node := sys.nodes["l1"]

	// create signer
	ks := l1Node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	opts, err := bind.NewKeyStoreTransactorWithChainID(ks, ks.Accounts()[0], cfg.L1ChainID)
	require.Nil(t, err)
	fromAddr := opts.From

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	cancel()
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	startNonce, err := l2Verif.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	cancel()

	toAddr := common.Address{0xff, 0xff}
	mintAmount := big.NewInt(9_000_000)
	opts.Value = mintAmount
	value := new(big.Int).Mul(common.Big2, startBalance) // trigger a revert by transferring more than we have available
	tx, err := depositContract.DepositTransaction(opts, toAddr, value, 1_000_000, false, nil)
	require.Nil(t, err, "with deposit tx")

	receipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	reconstructedDep, err := derive.UnmarshalLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = waitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusFailed)

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	endBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	cancel()
	require.Nil(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	toAddrBalance, err := l2Verif.BalanceAt(ctx, toAddr, nil)
	require.NoError(t, err)
	cancel()

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change")
	require.Equal(t, common.Big0.Int64(), toAddrBalance.Int64(), "The recipient account balance should be zero")

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	endNonce, err := l2Verif.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	cancel()
	require.Equal(t, startNonce+1, endNonce, "Nonce of deposit sender should increment on L2, even if the deposit fails")
}

func TestMissingBatchE2E(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}
	cfg := defaultSystemConfig(t)
	// Specifically set batch submitter balance to stop batches from being included
	cfg.Premine[bssHDPath] = 0

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: transactorHDPath,
		},
	})
	require.Nil(t, err)

	// Submit TX to L2 sequencer node
	toAddr := common.Address{0xff, 0xff}
	tx := types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainID,
		Nonce:     0,
		To:        &toAddr,
		Value:     big.NewInt(1_000_000_000),
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(200),
		Gas:       21000,
	})
	err = l2Seq.SendTransaction(context.Background(), tx)
	require.Nil(t, err, "Sending L2 tx to sequencer")

	// Let it show up on the unsafe chain
	receipt, err := waitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	_, err = waitForBlock(receipt.BlockNumber, l2Verif, 4*time.Second)
	require.Nil(t, err, "Waiting for block on verifier")

	// Assert that the transaction is not found on the verifier
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = l2Verif.TransactionReceipt(ctx, tx.Hash())
	require.Equal(t, ethereum.NotFound, err, "Found transaction in verifier when it should not have been included")

	// Wait a short time for the L2 reorg to occur on the sequencer.
	// The proper thing to do is to wait until the sequencer marks this block safe.
	<-time.After(200 * time.Millisecond)

	// Assert that the reconciliation process did an L2 reorg on the sequencer to remove the invalid block
	block, err := l2Seq.BlockByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err, "Get block from sequencer")
	require.NotEqual(t, block.Hash(), receipt.BlockHash, "L2 Sequencer did not reorg out transaction on it's safe chain")
}

func L1InfoFromState(ctx context.Context, contract *bindings.L1Block, l2Number *big.Int) (derive.L1BlockInfo, error) {
	var err error
	var out derive.L1BlockInfo
	opts := bind.CallOpts{
		BlockNumber: l2Number,
		Context:     ctx,
	}

	out.Number, err = contract.Number(&opts)
	if err != nil {
		return derive.L1BlockInfo{}, fmt.Errorf("failed to get number: %w", err)
	}

	out.Time, err = contract.Timestamp(&opts)
	if err != nil {
		return derive.L1BlockInfo{}, fmt.Errorf("failed to get timestamp: %w", err)
	}

	out.BaseFee, err = contract.Basefee(&opts)
	if err != nil {
		return derive.L1BlockInfo{}, fmt.Errorf("failed to get timestamp: %w", err)
	}

	blockHashBytes, err := contract.Hash(&opts)
	if err != nil {
		return derive.L1BlockInfo{}, fmt.Errorf("failed to get block hash: %w", err)
	}
	out.BlockHash = common.BytesToHash(blockHashBytes[:])

	out.SequenceNumber, err = contract.SequenceNumber(&opts)
	if err != nil {
		return derive.L1BlockInfo{}, fmt.Errorf("failed to get sequence number: %w", err)
	}

	return out, nil
}

// TestSystemMockP2P sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that
// the nodes can sync L2 blocks before they are confirmed on L1.
func TestSystemMockP2P(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := defaultSystemConfig(t)
	// slow down L1 blocks so we can see the L2 blocks arrive well before the L1 blocks do.
	// Keep the seq window small so the L2 chain is started quick
	cfg.L1BlockTime = 10

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"verifier": []string{"sequencer"},
	}

	var published, received []common.Hash
	seqTracer, verifTracer := new(FnTracer), new(FnTracer)
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *l2.ExecutionPayload) {
		published = append(published, payload.BlockHash)
	}
	verifTracer.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *l2.ExecutionPayload) {
		received = append(received, payload.BlockHash)
	}
	cfg.Nodes["sequencer"].Tracer = seqTracer
	cfg.Nodes["verifier"].Tracer = verifTracer

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: transactorHDPath,
		},
	})
	require.Nil(t, err)

	// Submit TX to L2 sequencer node
	toAddr := common.Address{0xff, 0xff}
	tx := types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainID,
		Nonce:     0,
		To:        &toAddr,
		Value:     big.NewInt(1_000_000_000),
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(200),
		Gas:       21000,
	})
	err = l2Seq.SendTransaction(context.Background(), tx)
	require.Nil(t, err, "Sending L2 tx to sequencer")

	// Wait for tx to be mined on the L2 sequencer chain
	receiptSeq, err := waitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	receiptVerif, err := waitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on verifier")

	require.Equal(t, receiptSeq, receiptVerif)

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(received))
	require.Equal(t, received, published[:len(received)])

	// Verify that the tx was received via p2p
	require.Contains(t, received, receiptVerif.BlockHash)
}

func TestL1InfoContract(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := defaultSystemConfig(t)

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	endVerifBlockNumber := big.NewInt(4)
	endSeqBlockNumber := big.NewInt(6)
	endVerifBlock, err := waitForBlock(endVerifBlockNumber, l2Verif, time.Minute)
	require.Nil(t, err)
	endSeqBlock, err := waitForBlock(endSeqBlockNumber, l2Seq, time.Minute)
	require.Nil(t, err)

	seqL1Info, err := bindings.NewL1Block(cfg.L1InfoPredeployAddress, l2Seq)
	require.Nil(t, err)

	verifL1Info, err := bindings.NewL1Block(cfg.L1InfoPredeployAddress, l2Verif)
	require.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fillInfoLists := func(start *types.Block, contract *bindings.L1Block, client *ethclient.Client) ([]derive.L1BlockInfo, []derive.L1BlockInfo) {
		var txList, stateList []derive.L1BlockInfo
		for b := start; ; {
			var infoFromTx derive.L1BlockInfo
			require.NoError(t, infoFromTx.UnmarshalBinary(b.Transactions()[0].Data()))
			txList = append(txList, infoFromTx)

			infoFromState, err := L1InfoFromState(ctx, contract, b.Number())
			require.Nil(t, err)
			stateList = append(stateList, infoFromState)

			// Genesis L2 block contains no L1 Deposit TX
			if b.NumberU64() == 1 {
				return txList, stateList
			}
			b, err = client.BlockByHash(ctx, b.ParentHash())
			require.Nil(t, err)
		}
	}

	l1InfosFromSequencerTransactions, l1InfosFromSequencerState := fillInfoLists(endSeqBlock, seqL1Info, l2Seq)
	l1InfosFromVerifierTransactions, l1InfosFromVerifierState := fillInfoLists(endVerifBlock, verifL1Info, l2Verif)

	l1blocks := make(map[common.Hash]derive.L1BlockInfo)
	maxL1Hash := l1InfosFromSequencerTransactions[0].BlockHash
	for h := maxL1Hash; ; {
		b, err := l1Client.BlockByHash(ctx, h)
		require.Nil(t, err)

		l1blocks[h] = derive.L1BlockInfo{
			Number:         b.NumberU64(),
			Time:           b.Time(),
			BaseFee:        b.BaseFee(),
			BlockHash:      h,
			SequenceNumber: 0, // ignored, will be overwritten
		}

		h = b.ParentHash()
		if b.NumberU64() == 0 {
			break
		}
	}

	checkInfoList := func(name string, list []derive.L1BlockInfo) {
		for _, info := range list {
			if expected, ok := l1blocks[info.BlockHash]; ok {
				expected.SequenceNumber = info.SequenceNumber // the seq nr is not part of the L1 info we know in advance, so we ignore it.
				require.Equal(t, expected, info)
			} else {
				t.Fatalf("Did not find block hash for L1 Info: %v in test %s", info, name)
			}
		}
	}

	checkInfoList("On sequencer with tx", l1InfosFromSequencerTransactions)
	checkInfoList("On sequencer with state", l1InfosFromSequencerState)
	checkInfoList("On verifier with tx", l1InfosFromVerifierTransactions)
	checkInfoList("On verifier with state", l1InfosFromVerifierState)

}

// calcGasFees determines the actual cost of the transaction given a specific basefee
func calcGasFees(gasUsed uint64, gasTipCap *big.Int, gasFeeCap *big.Int, baseFee *big.Int) *big.Int {
	x := new(big.Int).Add(gasTipCap, baseFee)
	// If tip + basefee > gas fee cap, clamp it to the gas fee cap
	if x.Cmp(gasFeeCap) > 0 {
		x = gasFeeCap
	}
	return x.Mul(x, new(big.Int).SetUint64(gasUsed))
}

// TestWithdrawals checks that a deposit and then withdrawal execution succeeds. It verifies the
// balance changes on L1 and L2 and has to include gas fees in the balance checks.
// It does not check that the withdrawal can be executed prior to the end of the finality period.
func TestWithdrawals(t *testing.T) {
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := defaultSystemConfig(t)
	cfg.DepositCFG.FinalizationPeriod = big.NewInt(2) // 2s finalization period

	sys, err := cfg.start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: transactorHDPath,
		},
	})
	require.Nil(t, err)
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)

	// Create L1 signer
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainID)
	require.Nil(t, err)

	// Start L2 balance
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Finally send TX
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	tx, err := depositContract.DepositTransaction(opts, fromAddr, common.Big0, 1_000_000, false, nil)
	require.Nil(t, err, "with deposit tx")

	receipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	// Bind L2 Withdrawer Contract
	l2withdrawer, err := bindings.NewL2ToL1MessagePasser(predeploy.WithdrawalContractAddress, l2Seq)
	require.Nil(t, err, "binding withdrawer on L2")

	// Wait for deposit to arrive
	reconstructedDep, err := derive.UnmarshalLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = waitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	// Confirm L2 balance
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change after mint")

	// Start L2 balance for withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err = l2Seq.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Intiate Withdrawal
	withdrawAmount := big.NewInt(500_000_000_000)
	l2opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L2ChainID)
	require.Nil(t, err)
	l2opts.Value = withdrawAmount
	tx, err = l2withdrawer.InitiateWithdrawal(l2opts, fromAddr, big.NewInt(21000), nil)
	require.Nil(t, err, "sending initiate withdraw tx")

	receipt, err = waitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "withdrawal initiated on L2 sequencer")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	// Verify L2 balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err := l2Seq.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err = l2Seq.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Take fee into account
	diff = new(big.Int).Sub(startBalance, endBalance)
	fees := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	diff = diff.Sub(diff, fees)
	require.Equal(t, withdrawAmount, diff)

	// Take start balance on L1
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	startBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Wait for finalization and then create the Finalized Withdrawal Transaction
	l2OutputOracle, err := bindings.NewL2OutputOracleCaller(sys.L2OOContractAddr, l1Client)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Duration(cfg.L1BlockTime)*time.Second)
	defer cancel()
	timestamp, err := withdrawals.WaitForFinalizationPeriod(ctx, l1Client, sys.DepositContractAddr, header.Time)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	blockNumber, err := l2OutputOracle.ComputeL2BlockNumber(&bind.CallOpts{Context: ctx}, new(big.Int).SetUint64(timestamp))
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err = l2Seq.HeaderByNumber(ctx, blockNumber)
	require.Nil(t, err)

	rpc, err := rpc.Dial(cfg.Nodes["sequencer"].L2NodeAddr)
	require.Nil(t, err)
	l2client := withdrawals.NewClient(rpc)

	// Now create withdrawal
	params, err := withdrawals.FinalizeWithdrawalParameters(context.Background(), l2client, tx.Hash(), header)
	require.Nil(t, err)

	portal, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)

	opts.Value = nil
	tx, err = portal.FinalizeWithdrawalTransaction(
		opts,
		params.Nonce,
		params.Sender,
		params.Target,
		params.Value,
		params.GasLimit,
		params.Data,
		params.Timestamp,
		params.OutputRootProof,
		params.WithdrawalProof,
	)

	require.Nil(t, err)

	receipt, err = waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "finalize withdrawal")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Verify balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	header, err = l1Client.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	endBalance, err = l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Ensure that withdrawal - gas fees are added to the L1 balance
	// Fun fact, the fee is greater than the withdrawal amount
	diff = new(big.Int).Sub(endBalance, startBalance)
	fees = calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	withdrawAmount = withdrawAmount.Sub(withdrawAmount, fees)
	require.Equal(t, withdrawAmount, diff)
}
