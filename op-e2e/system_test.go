package op_e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestL2OutputSubmitter(t *testing.T) {
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]

	rollupRPCClient, err := rpc.DialContext(context.Background(), cfg.Nodes["sequencer"].RPC.HttpEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(rollupRPCClient)

	//  OutputOracle is already deployed
	l2OutputOracle, err := bindings.NewL2OutputOracleCaller(sys.L2OOContractAddr, l1Client)
	require.Nil(t, err)

	initialOutputBlockNumber, err := l2OutputOracle.LatestBlockNumber(&bind.CallOpts{})
	require.Nil(t, err)

	// Wait until the second output submission from L2. The output submitter submits outputs from the
	// unsafe portion of the chain which gets reorged on startup. The sequencer has an out of date view
	// when it creates it's first block and uses and old L1 Origin. It then does not submit a batch
	// for that block and subsequently reorgs to match what the verifier derives when running the
	// reconcillation process.
	l2Verif := sys.Clients["verifier"]
	_, err = WaitForBlock(big.NewInt(6), l2Verif, 10*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err)

	// Wait for batch submitter to update L2 output oracle.
	timeoutCh := time.After(15 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		l2ooBlockNumber, err := l2OutputOracle.LatestBlockNumber(&bind.CallOpts{})
		require.Nil(t, err)

		// Wait for the L2 output oracle to have been changed from the initial
		// timestamp set in the contract constructor.
		if l2ooBlockNumber.Cmp(initialOutputBlockNumber) > 0 {
			// Retrieve the l2 output committed at this updated timestamp.
			committedL2Output, err := l2OutputOracle.GetL2Output(&bind.CallOpts{}, l2ooBlockNumber)
			require.NotEqual(t, [32]byte{}, committedL2Output.OutputRoot, "Empty L2 Output")
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
		case <-ticker.C:
		}
	}

}

// TestSystemE2E sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that L1 deposits are reflected on L2.
// All nodes are run in process (but are the full nodes, not mocked or stubbed).
func TestSystemE2E(t *testing.T) {
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LvlInfo)
	log.Info("genesis", "l2", sys.cfg.RollupConfig.Genesis.L2, "l1", sys.cfg.RollupConfig.Genesis.L1, "l2_time", sys.cfg.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.Wallet.PrivateKey(accounts.Account{
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
	l1Node := sys.Nodes["l1"]

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

	receipt, err := WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	reconstructedDep, err := derive.UnmarshalDepositLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = WaitForTransaction(tx.Hash(), l2Verif, 6*time.Duration(cfg.L1BlockTime)*time.Second)
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

	_, err = WaitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	receipt, err = WaitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on verifier")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "TX should have succeeded")

	// Verify blocks match after batch submission on verifiers and sequencers
	verifBlock, err := l2Verif.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	seqBlock, err := l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	require.Equal(t, verifBlock.NumberU64(), seqBlock.NumberU64(), "Verifier and sequencer blocks not the same after including a batch tx")
	require.Equal(t, verifBlock.ParentHash(), seqBlock.ParentHash(), "Verifier and sequencer blocks parent hashes not the same after including a batch tx")
	require.Equal(t, verifBlock.Hash(), seqBlock.Hash(), "Verifier and sequencer blocks not the same after including a batch tx")

	rollupRPCClient, err := rpc.DialContext(context.Background(), cfg.Nodes["sequencer"].RPC.HttpEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(rollupRPCClient)
	// basic check that sync status works
	seqStatus, err := rollupClient.SyncStatus(context.Background())
	require.Nil(t, err)
	require.LessOrEqual(t, seqBlock.NumberU64(), seqStatus.UnsafeL2.Number)
	// basic check that version endpoint works
	seqVersion, err := rollupClient.Version(context.Background())
	require.Nil(t, err)
	require.NotEqual(t, "", seqVersion)
}

// TestConfirmationDepth runs the rollup with both sequencer and verifier not immediately processing the tip of the chain.
func TestConfirmationDepth(t *testing.T) {
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)
	cfg.RollupConfig.SeqWindowSize = 4
	cfg.RollupConfig.MaxSequencerDrift = 3 * cfg.L1BlockTime
	seqConfDepth := uint64(2)
	verConfDepth := uint64(5)
	cfg.Nodes["sequencer"].Driver.SequencerConfDepth = seqConfDepth
	cfg.Nodes["sequencer"].Driver.VerifierConfDepth = 0
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = verConfDepth

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LvlInfo)
	log.Info("genesis", "l2", sys.cfg.RollupConfig.Genesis.L2, "l1", sys.cfg.RollupConfig.Genesis.L1, "l2_time", sys.cfg.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Wait enough time for the sequencer to submit a block with distance from L1 head, submit it,
	// and for the slower verifier to read a full sequence window and cover confirmation depth for reading and some margin
	<-time.After(time.Duration((cfg.RollupConfig.SeqWindowSize+verConfDepth+3)*cfg.L1BlockTime) * time.Second)

	// within a second, get both L1 and L2 verifier and sequencer block heads
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	l1Head, err := l1Client.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	l2SeqHead, err := l2Seq.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	l2VerHead, err := l2Verif.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	info, err := derive.L1InfoDepositTxData(l2SeqHead.Transactions()[0].Data())
	require.NoError(t, err)
	require.LessOrEqual(t, info.Number+seqConfDepth, l1Head.NumberU64(), "the L2 head block should have an origin older than the L1 head block by at least the sequencer conf depth")

	require.LessOrEqual(t, l2VerHead.Time()+cfg.L1BlockTime*verConfDepth, l2SeqHead.Time(), "the L2 verifier head should lag behind the sequencer without delay by at least the verifier conf depth")
}

func TestMintOnRevertedDeposit(t *testing.T) {
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}
	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Verif := sys.Clients["verifier"]

	// Find deposit contract
	depositContract, err := bindings.NewOptimismPortal(sys.DepositContractAddr, l1Client)
	require.Nil(t, err)
	l1Node := sys.Nodes["l1"]

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

	receipt, err := WaitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for deposit tx on L1")

	reconstructedDep, err := derive.UnmarshalDepositLogEvent(receipt.Logs[0])
	require.NoError(t, err, "Could not reconstruct L2 Deposit")
	tx = types.NewTx(reconstructedDep)
	receipt, err = WaitForTransaction(tx.Hash(), l2Verif, 3*time.Duration(cfg.L1BlockTime)*time.Second)
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
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}
	// Note this test zeroes the balance of the batch-submitter to make the batches unable to go into L1.
	// The test logs may look scary, but this is expected:
	// 'batcher unable to publish transaction    role=batcher   err="insufficient funds for gas * price + value"'

	cfg := DefaultSystemConfig(t)
	// small sequence window size so the test does not take as long
	cfg.RollupConfig.SeqWindowSize = 4

	// Specifically set batch submitter balance to stop batches from being included
	cfg.Premine[BSSHDPath] = 0

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.Wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: TransactorHDPath,
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
	receipt, err := WaitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	_, err = WaitForBlock(receipt.BlockNumber, l2Verif, time.Duration(cfg.RollupConfig.SeqWindowSize*cfg.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for block on verifier")

	// Assert that the transaction is not found on the verifier
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = l2Verif.TransactionReceipt(ctx, tx.Hash())
	require.Equal(t, ethereum.NotFound, err, "Found transaction in verifier when it should not have been included")

	// Wait a short time for the L2 reorg to occur on the sequencer as well.
	// The proper thing to do is to wait until the sequencer marks this block safe.
	<-time.After(2 * time.Second)

	// Assert that the reconciliation process did an L2 reorg on the sequencer to remove the invalid block
	ctx2, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	block, err := l2Seq.BlockByNumber(ctx2, receipt.BlockNumber)
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
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)
	// slow down L1 blocks so we can see the L2 blocks arrive well before the L1 blocks do.
	// Keep the seq window small so the L2 chain is started quick
	cfg.L1BlockTime = 10

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"verifier": []string{"sequencer"},
	}

	var published, received []common.Hash
	seqTracer, verifTracer := new(FnTracer), new(FnTracer)
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayload) {
		published = append(published, payload.BlockHash)
	}
	verifTracer.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) {
		received = append(received, payload.BlockHash)
	}
	cfg.Nodes["sequencer"].Tracer = seqTracer
	cfg.Nodes["verifier"].Tracer = verifTracer

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey, err := sys.Wallet.PrivateKey(accounts.Account{
		URL: accounts.URL{
			Path: TransactorHDPath,
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
	receiptSeq, err := WaitForTransaction(tx.Hash(), l2Seq, 3*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on sequencer")

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	receiptVerif, err := WaitForTransaction(tx.Hash(), l2Verif, 6*time.Duration(cfg.RollupConfig.BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L2 tx on verifier")

	require.Equal(t, receiptSeq, receiptVerif)

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(received))
	require.Equal(t, received, published[:len(received)])

	// Verify that the tx was received via p2p
	require.Contains(t, received, receiptVerif.BlockHash)
}

func TestL1InfoContract(t *testing.T) {
	if !VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	endVerifBlockNumber := big.NewInt(4)
	endSeqBlockNumber := big.NewInt(6)
	endVerifBlock, err := WaitForBlock(endVerifBlockNumber, l2Verif, time.Minute)
	require.Nil(t, err)
	endSeqBlock, err := WaitForBlock(endSeqBlockNumber, l2Seq, time.Minute)
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
