package op_e2e

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	metrics2 "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSystemBatchType run each system e2e test case in singular batch mode and span batch mode.
// If the test case tests batch submission and advancing safe head, it should be tested in both singular and span batch mode.
func TestSystemBatchType(t *testing.T) {
	tests := []struct {
		name string
		f    func(gt *testing.T, deltaTimeOffset *hexutil.Uint64)
	}{
		{"StopStartBatcher", StopStartBatcher},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SingularBatch", func(t *testing.T) {
			test.f(t, nil)
		})
	}

	deltaTimeOffset := hexutil.Uint64(0)
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SpanBatch", func(t *testing.T) {
			test.f(t, &deltaTimeOffset)
		})
	}
}

func TestMain(m *testing.M) {
	if config.ExternalL2Shim != "" {
		fmt.Println("Running tests with external L2 process adapter at ", config.ExternalL2Shim)
		// As these are integration tests which launch many other processes, the
		// default parallelism makes the tests flaky.  This change aims to
		// reduce the flakiness of these tests.
		maxProcs := runtime.NumCPU() / 4
		if maxProcs == 0 {
			maxProcs = 1
		}
		runtime.GOMAXPROCS(maxProcs)
	}

	os.Exit(m.Run())
}

func TestL2OutputSubmitter(t *testing.T) {
	InitParallel(t, SkipOnFaultProofs)

	cfg := DefaultSystemConfig(t)
	cfg.NonFinalizedProposals = true // speed up the time till we see output proposals

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]

	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	//  OutputOracle is already deployed
	l2OutputOracle, err := bindings.NewL2OutputOracleCaller(cfg.L1Deployments.L2OutputOracleProxy, l1Client)
	require.Nil(t, err)

	initialOutputBlockNumber, err := l2OutputOracle.LatestBlockNumber(&bind.CallOpts{})
	require.Nil(t, err)

	// Wait until the second output submission from L2. The output submitter submits outputs from the
	// unsafe portion of the chain which gets reorged on startup. The sequencer has an out of date view
	// when it creates it's first block and uses and old L1 Origin. It then does not submit a batch
	// for that block and subsequently reorgs to match what the verifier derives when running the
	// reconcillation process.
	l2Verif := sys.Clients["verifier"]
	_, err = geth.WaitForBlock(big.NewInt(6), l2Verif, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
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
			committedL2Output, err := l2OutputOracle.GetL2OutputAfter(&bind.CallOpts{}, l2ooBlockNumber)
			require.NotEqual(t, [32]byte{}, committedL2Output.OutputRoot, "Empty L2 Output")
			require.Nil(t, err)

			// Fetch the corresponding L2 block and assert the committed L2
			// output matches the block's state root.
			//
			// NOTE: This assertion will change once the L2 output format is
			// finalized.
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			l2Output, err := rollupClient.OutputAtBlock(ctx, l2ooBlockNumber.Uint64())
			require.Nil(t, err)
			require.Equal(t, l2Output.OutputRoot[:], committedL2Output.OutputRoot[:])
			break
		}

		select {
		case <-timeoutCh:
			t.Fatalf("State root oracle not updated")
		case <-ticker.C:
		}
	}
}

func TestL2OutputSubmitterFaultProofs(t *testing.T) {
	InitParallel(t, SkipOnL2OO)

	cfg := DefaultSystemConfig(t)
	cfg.NonFinalizedProposals = true // speed up the time till we see output proposals

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]

	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	disputeGameFactory, err := bindings.NewDisputeGameFactoryCaller(cfg.L1Deployments.DisputeGameFactoryProxy, l1Client)
	require.Nil(t, err)

	initialGameCount, err := disputeGameFactory.GameCount(&bind.CallOpts{})
	require.Nil(t, err)

	l2Verif := sys.Clients["verifier"]
	_, err = geth.WaitForBlock(big.NewInt(6), l2Verif, 10*time.Duration(cfg.DeployConfig.L2BlockTime)*time.Second)
	require.Nil(t, err)

	timeoutCh := time.After(15 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		latestGameCount, err := disputeGameFactory.GameCount(&bind.CallOpts{})
		require.Nil(t, err)

		if latestGameCount.Cmp(initialGameCount) > 0 {
			caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)
			committedL2Output, err := disputeGameFactory.GameAtIndex(&bind.CallOpts{}, new(big.Int).Sub(latestGameCount, common.Big1))
			require.Nil(t, err)
			proxy, err := contracts.NewFaultDisputeGameContract(context.Background(), metrics2.NoopContractMetrics, committedL2Output.Proxy, caller)
			require.Nil(t, err)
			claim, err := proxy.GetClaim(context.Background(), 0)
			require.Nil(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, gameBlockNumber, err := proxy.GetBlockRange(ctx)
			require.Nil(t, err)
			l2Output, err := rollupClient.OutputAtBlock(ctx, gameBlockNumber)
			require.Nil(t, err)
			require.EqualValues(t, l2Output.OutputRoot, claim.Value)
			break
		}

		select {
		case <-timeoutCh:
			t.Fatalf("State root oracle not updated")
		case <-ticker.C:
		}
	}
}

func TestSystemE2EDencunAtGenesis(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
	runE2ESystemTest(t, sys)
	head, err := sys.Clients["l1"].BlockByNumber(context.Background(), big.NewInt(0))
	require.NoError(t, err)
	require.NotNil(t, head.ExcessBlobGas(), "L1 is building dencun blocks since genesis")
}

// TestSystemE2EDencunAtGenesis tests if L2 finalizes when blobs are present on L1
func TestSystemE2EDencunAtGenesisWithBlobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	// cancun is on from genesis:
	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation // i.e. turn cancun on at genesis time + 0

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// send a blob-containing txn on l1
	ethPrivKey := sys.Cfg.Secrets.Alice
	txData := transactions.CreateEmptyBlobTx(true, sys.Cfg.L1ChainIDBig().Uint64())
	tx := types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L1ChainIDBig()), txData)
	// send blob-containing txn
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer sendCancel()

	l1Client := sys.Clients["l1"]
	err = l1Client.SendTransaction(sendCtx, tx)
	require.NoError(t, err, "Sending L1 empty blob tx")
	// Wait for transaction on L1
	blockContainsBlob, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.Nil(t, err, "Waiting for blob tx on L1")
	// end sending blob-containing txns on l1
	l2Client := sys.Clients["sequencer"]
	finalizedBlock, err := geth.WaitForL1OriginOnL2(sys.RollupConfig, blockContainsBlob.BlockNumber.Uint64(), l2Client, 30*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for L1 origin of blob tx on L2")
	finalizationTimeout := 30 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second
	_, err = geth.WaitForBlockToBeSafe(finalizedBlock.Header().Number, l2Client, finalizationTimeout)
	require.Nil(t, err, "Waiting for safety of L2 block")
}

// TestSystemE2E sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that L1 deposits are reflected on L2.
// All nodes are run in process (but are the full nodes, not mocked or stubbed).
func TestSystemE2E(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	runE2ESystemTest(t, sys)
	defer sys.Close()
}

func runE2ESystemTest(t *testing.T, sys *System) {
	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey := sys.Cfg.Secrets.Alice

	// Send Transaction & wait for success
	fromAddr := sys.Cfg.Secrets.Addresses().Alice

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Send deposit transaction
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, sys.Cfg.L1ChainIDBig())
	require.Nil(t, err)
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	SendDepositTx(t, sys.Cfg, l1Client, l2Verif, opts, func(l2Opts *DepositTxOpts) {})

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	endBalance, err := wait.ForBalanceChange(ctx, l2Verif, fromAddr, startBalance)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change")

	// Submit TX to L2 sequencer node
	receipt := SendL2Tx(t, sys.Cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.Value = big.NewInt(1_000_000_000)
		opts.Nonce = 1 // Already have deposit
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.VerifyOnClients(l2Verif)
	})

	// Verify blocks match after batch submission on verifiers and sequencers
	verifBlock, err := l2Verif.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	seqBlock, err := l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)
	require.Equal(t, verifBlock.NumberU64(), seqBlock.NumberU64(), "Verifier and sequencer blocks not the same after including a batch tx")
	require.Equal(t, verifBlock.ParentHash(), seqBlock.ParentHash(), "Verifier and sequencer blocks parent hashes not the same after including a batch tx")
	require.Equal(t, verifBlock.Hash(), seqBlock.Hash(), "Verifier and sequencer blocks not the same after including a batch tx")

	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))
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
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.SequencerWindowSize = 4
	cfg.DeployConfig.MaxSequencerDrift = 10 * cfg.DeployConfig.L1BlockTime
	seqConfDepth := uint64(2)
	verConfDepth := uint64(5)
	cfg.Nodes["sequencer"].Driver.SequencerConfDepth = seqConfDepth
	cfg.Nodes["sequencer"].Driver.VerifierConfDepth = 0
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = verConfDepth

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Wait enough time for the sequencer to submit a block with distance from L1 head, submit it,
	// and for the slower verifier to read a full sequence window and cover confirmation depth for reading and some margin
	<-time.After(time.Duration((cfg.DeployConfig.SequencerWindowSize+verConfDepth+3)*cfg.DeployConfig.L1BlockTime) * time.Second)

	// within a second, get both L1 and L2 verifier and sequencer block heads
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l1Head, err := l1Client.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	l2SeqHead, err := l2Seq.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	l2VerHead, err := l2Verif.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	seqInfo, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, l2SeqHead.Time(), l2SeqHead.Transactions()[0].Data())
	require.NoError(t, err)
	require.LessOrEqual(t, seqInfo.Number+seqConfDepth, l1Head.NumberU64(), "the seq L2 head block should have an origin older than the L1 head block by at least the sequencer conf depth")

	verInfo, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, l2VerHead.Time(), l2VerHead.Transactions()[0].Data())
	require.NoError(t, err)
	require.LessOrEqual(t, verInfo.Number+verConfDepth, l1Head.NumberU64(), "the ver L2 head block should have an origin older than the L1 head block by at least the verifier conf depth")
}

// TestPendingGasLimit tests the configuration of the gas limit of the pending block,
// and if it does not conflict with the regular gas limit on the verifier or sequencer.
func TestPendingGasLimit(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	// configure the L2 gas limit to be high, and the pending gas limits to be lower for resource saving.
	cfg.DeployConfig.L2GenesisBlockGasLimit = 30_000_000
	cfg.GethOptions["sequencer"] = append(cfg.GethOptions["sequencer"], []geth.GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.Miner.GasCeil = 10_000_000
			ethCfg.Miner.RollupComputePendingBlock = true
			return nil
		},
	}...)
	cfg.GethOptions["verifier"] = append(cfg.GethOptions["verifier"], []geth.GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.Miner.GasCeil = 9_000_000
			ethCfg.Miner.RollupComputePendingBlock = true
			return nil
		},
	}...)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l2Verif := sys.Clients["verifier"]
	l2Seq := sys.Clients["sequencer"]

	checkGasLimit := func(client *ethclient.Client, number *big.Int, expected uint64) *types.Header {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		header, err := client.HeaderByNumber(ctx, number)
		cancel()
		require.NoError(t, err)
		require.Equal(t, expected, header.GasLimit)
		return header
	}

	// check if the gaslimits are matching the expected values,
	// and that the verifier/sequencer can use their locally configured gas limit for the pending block.
	for {
		checkGasLimit(l2Seq, big.NewInt(-1), 10_000_000)
		checkGasLimit(l2Verif, big.NewInt(-1), 9_000_000)
		checkGasLimit(l2Seq, nil, 30_000_000)
		latestVerifHeader := checkGasLimit(l2Verif, nil, 30_000_000)

		// Stop once the verifier passes genesis:
		// this implies we checked a new block from the sequencer, on both sequencer and verifier nodes.
		if latestVerifHeader.Number.Uint64() > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// TestFinalize tests if L2 finalizes after sufficient time after L1 finalizes
func TestFinalize(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]

	l2Finalized, err := geth.WaitForBlockToBeFinalized(big.NewInt(12), l2Seq, 1*time.Minute)
	require.NoError(t, err, "must be able to fetch a finalized L2 block")
	require.NotZerof(t, l2Finalized.NumberU64(), "must have finalized L2 block")
}

func TestMissingBatchE2E(t *testing.T) {
	InitParallel(t)
	// Note this test zeroes the balance of the batch-submitter to make the batches unable to go into L1.
	// The test logs may look scary, but this is expected:
	// 'batcher unable to publish transaction    role=batcher   err="insufficient funds for gas * price + value"'

	cfg := DefaultSystemConfig(t)
	// small sequence window size so the test does not take as long
	cfg.DeployConfig.SequencerWindowSize = 4

	// Specifically set batch submitter balance to stop batches from being included
	cfg.Premine[cfg.Secrets.Addresses().Batcher] = big.NewInt(0)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	seqRollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)
	seqRollupClient := sources.NewRollupClient(client.NewBaseRPCClient(seqRollupRPCClient))

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit TX to L2 sequencer node
	receipt := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)
	})

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	_, err = geth.WaitForBlock(receipt.BlockNumber, l2Verif, time.Duration((sys.RollupConfig.SeqWindowSize+4)*cfg.DeployConfig.L1BlockTime)*time.Second)
	require.Nil(t, err, "Waiting for block on verifier")

	// Assert that the transaction is not found on the verifier
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = l2Verif.TransactionReceipt(ctx, receipt.TxHash)
	require.Equal(t, ethereum.NotFound, err, "Found transaction in verifier when it should not have been included")

	// Wait a short time for the L2 reorg to occur on the sequencer as well.
	err = waitForSafeHead(ctx, receipt.BlockNumber.Uint64(), seqRollupClient)
	require.Nil(t, err, "timeout waiting for L2 reorg on sequencer safe head")

	// Assert that the reconciliation process did an L2 reorg on the sequencer to remove the invalid block
	ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	block, err := l2Seq.BlockByNumber(ctx2, receipt.BlockNumber)
	if err != nil {
		require.Equal(t, "not found", err.Error(), "A not found error indicates the chain must have re-orged back before it")
	} else {
		require.NotEqual(t, block.Hash(), receipt.BlockHash, "L2 Sequencer did not reorg out transaction on it's safe chain")
	}
}

func L1InfoFromState(ctx context.Context, contract *bindings.L1Block, l2Number *big.Int, ecotone bool) (*derive.L1BlockInfo, error) {
	var err error
	out := &derive.L1BlockInfo{}
	opts := bind.CallOpts{
		BlockNumber: l2Number,
		Context:     ctx,
	}

	out.Number, err = contract.Number(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get number: %w", err)
	}

	out.Time, err = contract.Timestamp(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %w", err)
	}

	out.BaseFee, err = contract.Basefee(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get base fee: %w", err)
	}

	blockHashBytes, err := contract.Hash(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get block hash: %w", err)
	}
	out.BlockHash = common.BytesToHash(blockHashBytes[:])

	out.SequenceNumber, err = contract.SequenceNumber(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequence number: %w", err)
	}

	if !ecotone {
		overhead, err := contract.L1FeeOverhead(&opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get l1 fee overhead: %w", err)
		}
		out.L1FeeOverhead = eth.Bytes32(common.BigToHash(overhead))

		scalar, err := contract.L1FeeScalar(&opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get l1 fee scalar: %w", err)
		}
		out.L1FeeScalar = eth.Bytes32(common.BigToHash(scalar))
	}

	batcherHash, err := contract.BatcherHash(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch sender: %w", err)
	}
	out.BatcherAddr = common.BytesToAddress(batcherHash[:])

	if ecotone {
		blobBaseFeeScalar, err := contract.BlobBaseFeeScalar(&opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get blob basefee scalar: %w", err)
		}
		out.BlobBaseFeeScalar = blobBaseFeeScalar

		baseFeeScalar, err := contract.BaseFeeScalar(&opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get basefee scalar: %w", err)
		}
		out.BaseFeeScalar = baseFeeScalar

		blobBaseFee, err := contract.BlobBaseFee(&opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get blob basefee: %w", err)
		}
		out.BlobBaseFee = blobBaseFee
	}

	return out, nil
}

// TestSystemMockP2P sets up a L1 Geth node, a rollup node, and a L2 geth node and then confirms that
// the nodes can sync L2 blocks before they are confirmed on L1.
func TestSystemMockP2P(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	// Disable batcher, so we don't sync from L1 & set a large sequence window so we only have unsafe blocks
	cfg.DisableBatcher = true
	cfg.DeployConfig.SequencerWindowSize = 100_000
	cfg.DeployConfig.MaxSequencerDrift = 100_000
	// disable at the start, so we don't miss any gossiped blocks.
	cfg.Nodes["sequencer"].Driver.SequencerStopped = true

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"verifier": {"sequencer"},
	}

	var published, received []common.Hash
	seqTracer, verifTracer := new(FnTracer), new(FnTracer)
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		published = append(published, payload.ExecutionPayload.BlockHash)
	}
	verifTracer.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
		received = append(received, payload.ExecutionPayload.BlockHash)
	}
	cfg.Nodes["sequencer"].Tracer = seqTracer
	cfg.Nodes["verifier"].Tracer = verifTracer

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Enable the sequencer now that everyone is ready to receive payloads.
	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.Nil(t, err)

	verifierPeerID := sys.RollupNodes["verifier"].P2P().Host().ID()
	check := func() bool {
		sequencerBlocksTopicPeers := sys.RollupNodes["sequencer"].P2P().GossipOut().AllBlockTopicsPeers()
		return slices.Contains[[]peer.ID](sequencerBlocksTopicPeers, verifierPeerID)
	}

	// poll to see if the verifier node is connected & meshed on gossip.
	// Without this verifier, we shouldn't start sending blocks around, or we'll miss them and fail the test.
	backOffStrategy := retry.Exponential()
	for i := 0; i < 10; i++ {
		if check() {
			break
		}
		time.Sleep(backOffStrategy.Duration(i))
	}
	require.True(t, check(), "verifier must be meshed with sequencer for gossip test to proceed")

	require.NoError(t, rollupRPCClient.Call(nil, "admin_startSequencer", sys.L2GenesisCfg.ToBlock().Hash()))

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit TX to L2 sequencer node
	receiptSeq := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)

		// Wait until the block it was first included in shows up in the safe chain on the verifier
		opts.VerifyOnClients(l2Verif)
	})

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(received))
	require.Subset(t, published, received)

	// Verify that the tx was received via p2p
	require.Contains(t, received, receiptSeq.BlockHash)
}

func TestSystemP2PAltSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	// remove default verifier node
	delete(cfg.Nodes, "verifier")
	// Add more verifier nodes
	cfg.Nodes["alice"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Nodes["bob"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Loggers["alice"] = testlog.Logger(t, log.LevelInfo).New("role", "alice")
	cfg.Loggers["bob"] = testlog.Logger(t, log.LevelInfo).New("role", "bob")

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"sequencer": {"alice", "bob"},
		"alice":     {"sequencer", "bob"},
		"bob":       {"alice", "sequencer"},
	}
	// Enable the P2P req-resp based sync
	cfg.P2PReqRespSync = true

	// Disable batcher, so there will not be any L1 data to sync from
	cfg.DisableBatcher = true

	var published []string
	seqTracer := new(FnTracer)
	// The sequencer still publishes the blocks to the tracer, even if they do not reach the network due to disabled P2P
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		published = append(published, payload.ExecutionPayload.ID().String())
	}
	// Blocks are now received via the RPC based alt-sync method
	cfg.Nodes["sequencer"].Tracer = seqTracer

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit a TX to L2 sequencer node
	receiptSeq := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)
	})

	// Gossip is able to respond to IWANT messages for the duration of heartbeat_time * message_window = 0.5 * 12 = 6
	// Wait till we pass that, and then we'll have missed some blocks that cannot be retrieved in any way from gossip
	time.Sleep(time.Second * 10)

	// set up our syncer node, connect it to alice/bob
	cfg.Loggers["syncer"] = testlog.Logger(t, log.LevelInfo).New("role", "syncer")
	snapLog := log.NewLogger(log.DiscardHandler())

	// Create a peer, and hook up alice and bob
	h, err := sys.newMockNetPeer()
	require.NoError(t, err)
	_, err = sys.Mocknet.LinkPeers(sys.RollupNodes["alice"].P2P().Host().ID(), h.ID())
	require.NoError(t, err)
	_, err = sys.Mocknet.LinkPeers(sys.RollupNodes["bob"].P2P().Host().ID(), h.ID())
	require.NoError(t, err)

	// Configure the new rollup node that'll be syncing
	var syncedPayloads []string
	syncNodeCfg := &rollupNode.Config{
		Driver:    driver.Config{VerifierConfDepth: 0},
		Rollup:    *sys.RollupConfig,
		P2PSigner: nil,
		RPC: rollupNode.RPCConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P:                 &p2p.Prepared{HostP2P: h, EnableReqRespSync: true},
		Metrics:             rollupNode.MetricsConfig{Enabled: false}, // no metrics server
		Pprof:               oppprof.CLIConfig{},
		L1EpochPollInterval: time.Second * 10,
		Tracer: &FnTracer{
			OnUnsafeL2PayloadFn: func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
				syncedPayloads = append(syncedPayloads, payload.ExecutionPayload.ID().String())
			},
		},
	}
	configureL1(syncNodeCfg, sys.EthInstances["l1"])
	syncerL2Engine, _, err := geth.InitL2("syncer", big.NewInt(int64(cfg.DeployConfig.L2ChainID)), sys.L2GenesisCfg, cfg.JWTFilePath)
	require.NoError(t, err)
	require.NoError(t, syncerL2Engine.Start())

	configureL2(syncNodeCfg, syncerL2Engine, cfg.JWTSecret)

	syncerNode, err := rollupNode.New(ctx, syncNodeCfg, cfg.Loggers["syncer"], snapLog, "", metrics.NewMetrics(""))
	require.NoError(t, err)
	err = syncerNode.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, syncerNode.Stop(ctx))
	}()

	// connect alice and bob to our new syncer node
	_, err = sys.Mocknet.ConnectPeers(sys.RollupNodes["alice"].P2P().Host().ID(), syncerNode.P2P().Host().ID())
	require.NoError(t, err)
	_, err = sys.Mocknet.ConnectPeers(sys.RollupNodes["bob"].P2P().Host().ID(), syncerNode.P2P().Host().ID())
	require.NoError(t, err)

	rpc := syncerL2Engine.Attach()
	l2Verif := ethclient.NewClient(rpc)

	// It may take a while to sync, but eventually we should see the sequenced data show up
	receiptVerif, err := wait.ForReceiptOK(ctx, l2Verif, receiptSeq.TxHash)
	require.Nil(t, err, "Waiting for L2 tx on verifier")

	require.Equal(t, receiptSeq, receiptVerif)

	// Verify that the tx was received via P2P sync
	require.Contains(t, syncedPayloads, eth.BlockID{Hash: receiptVerif.BlockHash, Number: receiptVerif.BlockNumber.Uint64()}.String())

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(syncedPayloads))
	require.Subset(t, published, syncedPayloads)
}

// TestSystemDenseTopology sets up a dense p2p topology with 3 verifier nodes and 1 sequencer node.
func TestSystemDenseTopology(t *testing.T) {
	t.Skip("Skipping dense topology test to avoid flakiness. @refcell address in p2p scoring pr.")

	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	// slow down L1 blocks so we can see the L2 blocks arrive well before the L1 blocks do.
	// Keep the seq window small so the L2 chain is started quick
	cfg.DeployConfig.L1BlockTime = 10

	// Append additional nodes to the system to construct a dense p2p network
	cfg.Nodes["verifier2"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Nodes["verifier3"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Loggers["verifier2"] = testlog.Logger(t, log.LevelInfo).New("role", "verifier")
	cfg.Loggers["verifier3"] = testlog.Logger(t, log.LevelInfo).New("role", "verifier")

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"verifier":  {"sequencer", "verifier2", "verifier3"},
		"verifier2": {"sequencer", "verifier", "verifier3"},
		"verifier3": {"sequencer", "verifier", "verifier2"},
	}

	// Set peer scoring for each node, but without banning
	for _, node := range cfg.Nodes {
		params, err := p2p.GetScoringParams("light", &node.Rollup)
		require.NoError(t, err)
		node.P2P = &p2p.Config{
			ScoringParams:  params,
			BanningEnabled: false,
		}
	}

	var published, received1, received2, received3 []common.Hash
	seqTracer, verifTracer, verifTracer2, verifTracer3 := new(FnTracer), new(FnTracer), new(FnTracer), new(FnTracer)
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		published = append(published, payload.ExecutionPayload.BlockHash)
	}
	verifTracer.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
		received1 = append(received1, payload.ExecutionPayload.BlockHash)
	}
	verifTracer2.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
		received2 = append(received2, payload.ExecutionPayload.BlockHash)
	}
	verifTracer3.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
		received3 = append(received3, payload.ExecutionPayload.BlockHash)
	}
	cfg.Nodes["sequencer"].Tracer = seqTracer
	cfg.Nodes["verifier"].Tracer = verifTracer
	cfg.Nodes["verifier2"].Tracer = verifTracer2
	cfg.Nodes["verifier3"].Tracer = verifTracer3

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	l2Verif2 := sys.Clients["verifier2"]
	l2Verif3 := sys.Clients["verifier3"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit TX to L2 sequencer node
	receiptSeq := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)

		// Wait until the block it was first included in shows up in the safe chain on the verifiers
		opts.VerifyOnClients(l2Verif, l2Verif2, l2Verif3)
	})

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(received1))
	require.GreaterOrEqual(t, len(published), len(received2))
	require.GreaterOrEqual(t, len(published), len(received3))
	require.ElementsMatch(t, published, received1[:len(published)])
	require.ElementsMatch(t, published, received2[:len(published)])
	require.ElementsMatch(t, published, received3[:len(published)])

	// Verify that the tx was received via p2p
	require.Contains(t, received1, receiptSeq.BlockHash)
	require.Contains(t, received2, receiptSeq.BlockHash)
	require.Contains(t, received3, receiptSeq.BlockHash)
}

func TestL1InfoContract(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	endVerifBlockNumber := big.NewInt(4)
	endSeqBlockNumber := big.NewInt(6)
	endVerifBlock, err := geth.WaitForBlock(endVerifBlockNumber, l2Verif, time.Minute)
	require.Nil(t, err)
	endSeqBlock, err := geth.WaitForBlock(endSeqBlockNumber, l2Seq, time.Minute)
	require.Nil(t, err)

	seqL1Info, err := bindings.NewL1Block(cfg.L1InfoPredeployAddress, l2Seq)
	require.Nil(t, err)

	verifL1Info, err := bindings.NewL1Block(cfg.L1InfoPredeployAddress, l2Verif)
	require.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fillInfoLists := func(start *types.Block, contract *bindings.L1Block, client *ethclient.Client) ([]*derive.L1BlockInfo, []*derive.L1BlockInfo) {
		var txList, stateList []*derive.L1BlockInfo
		for b := start; ; {
			var infoFromTx *derive.L1BlockInfo
			infoFromTx, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, b.Time(), b.Transactions()[0].Data())
			require.NoError(t, err)
			txList = append(txList, infoFromTx)

			ecotone := sys.RollupConfig.IsEcotone(b.Time()) && !sys.RollupConfig.IsEcotoneActivationBlock(b.Time())
			infoFromState, err := L1InfoFromState(ctx, contract, b.Number(), ecotone)
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

	l1blocks := make(map[common.Hash]*derive.L1BlockInfo)
	maxL1Hash := l1InfosFromSequencerTransactions[0].BlockHash
	for h := maxL1Hash; ; {
		b, err := l1Client.BlockByHash(ctx, h)
		require.Nil(t, err)

		l1blocks[h] = &derive.L1BlockInfo{
			Number:         b.NumberU64(),
			Time:           b.Time(),
			BaseFee:        b.BaseFee(),
			BlockHash:      h,
			SequenceNumber: 0, // ignored, will be overwritten
			BatcherAddr:    sys.RollupConfig.Genesis.SystemConfig.BatcherAddr,
		}
		if sys.RollupConfig.IsEcotone(b.Time()) && !sys.RollupConfig.IsEcotoneActivationBlock(b.Time()) {
			scalars, err := sys.RollupConfig.Genesis.SystemConfig.EcotoneScalars()
			require.NoError(t, err)
			l1blocks[h].BlobBaseFeeScalar = scalars.BlobBaseFeeScalar
			l1blocks[h].BaseFeeScalar = scalars.BaseFeeScalar
			if excess := b.ExcessBlobGas(); excess != nil {
				l1blocks[h].BlobBaseFee = eip4844.CalcBlobFee(*excess)
			} else {
				l1blocks[h].BlobBaseFee = big.NewInt(1)
			}
		} else {
			l1blocks[h].L1FeeOverhead = sys.RollupConfig.Genesis.SystemConfig.Overhead
			l1blocks[h].L1FeeScalar = sys.RollupConfig.Genesis.SystemConfig.Scalar
		}

		h = b.ParentHash()
		if b.NumberU64() == 0 {
			break
		}
	}

	checkInfoList := func(name string, list []*derive.L1BlockInfo) {
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

// calcGasFees determines the actual cost of the transaction given a specific base fee
// This does not include the L1 data fee charged from L2 transactions.
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
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FinalizationPeriodSeconds = 2 // 2s finalization period

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	// Create L1 signer
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Start L2 balance
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeDeposit, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	// Send deposit tx
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	SendDepositTx(t, cfg, l1Client, l2Verif, opts, func(l2Opts *DepositTxOpts) {
		l2Opts.Value = common.Big0
	})

	// Confirm L2 balance
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endBalanceAfterDeposit, err := wait.ForBalanceChange(ctx, l2Verif, fromAddr, startBalanceBeforeDeposit)
	require.Nil(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalanceAfterDeposit, startBalanceBeforeDeposit)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change after mint")

	// Start L2 balance for withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeWithdrawal, err := l2Seq.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	withdrawAmount := big.NewInt(500_000_000_000)
	tx, receipt := SendWithdrawal(t, cfg, l2Seq, ethPrivKey, func(opts *WithdrawalTxOpts) {
		opts.Value = withdrawAmount
		opts.VerifyOnClients(l2Verif)
	})

	// Verify L2 balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	header, err := l2Verif.HeaderByNumber(ctx, receipt.BlockNumber)
	require.Nil(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endBalanceAfterWithdrawal, err := wait.ForBalanceChange(ctx, l2Seq, fromAddr, startBalanceBeforeWithdrawal)
	require.Nil(t, err)

	// Take fee into account
	diff = new(big.Int).Sub(startBalanceBeforeWithdrawal, endBalanceAfterWithdrawal)
	fees := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
	fees = fees.Add(fees, receipt.L1Fee)
	diff = diff.Sub(diff, fees)
	require.Equal(t, withdrawAmount, diff)

	// Take start balance on L1
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	startBalanceBeforeFinalize, err := l1Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)

	proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt := ProveAndFinalizeWithdrawal(t, cfg, sys, "verifier", ethPrivKey, receipt)

	// Verify balance after withdrawal
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endBalanceAfterFinalize, err := wait.ForBalanceChange(ctx, l1Client, fromAddr, startBalanceBeforeFinalize)
	require.Nil(t, err)

	// Ensure that withdrawal - gas fees are added to the L1 balance
	// Fun fact, the fee is greater than the withdrawal amount
	// NOTE: The gas fees include *both* the ProveWithdrawalTransaction and FinalizeWithdrawalTransaction transactions.
	diff = new(big.Int).Sub(endBalanceAfterFinalize, startBalanceBeforeFinalize)
	proveFee := new(big.Int).Mul(new(big.Int).SetUint64(proveReceipt.GasUsed), proveReceipt.EffectiveGasPrice)
	finalizeFee := new(big.Int).Mul(new(big.Int).SetUint64(finalizeReceipt.GasUsed), finalizeReceipt.EffectiveGasPrice)
	fees = new(big.Int).Add(proveFee, finalizeFee)
	if e2eutils.UseFaultProofs() {
		resolveClaimFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveClaimReceipt.GasUsed), resolveClaimReceipt.EffectiveGasPrice)
		resolveFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveReceipt.GasUsed), resolveReceipt.EffectiveGasPrice)
		fees = new(big.Int).Add(fees, resolveClaimFee)
		fees = new(big.Int).Add(fees, resolveFee)
	}
	withdrawAmount = withdrawAmount.Sub(withdrawAmount, fees)
	require.Equal(t, withdrawAmount, diff)
}

type stateGetterAdapter struct {
	ctx      context.Context
	t        *testing.T
	client   *ethclient.Client
	blockNum *big.Int
}

func (sga *stateGetterAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	sga.t.Helper()
	val, err := sga.client.StorageAt(sga.ctx, addr, key, sga.blockNum)
	require.NoError(sga.t, err)
	var res common.Hash
	copy(res[:], val)
	return res
}

// TestFees checks that L1/L2 fees are handled.
func TestFees(t *testing.T) {
	t.Run("pre-regolith", func(t *testing.T) {
		InitParallel(t)
		cfg := DefaultSystemConfig(t)
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		cfg.DeployConfig.L2GenesisRegolithTimeOffset = nil
		cfg.DeployConfig.L2GenesisCanyonTimeOffset = nil
		cfg.DeployConfig.L2GenesisDeltaTimeOffset = nil
		cfg.DeployConfig.L2GenesisEcotoneTimeOffset = nil
		testFees(t, cfg)
	})
	t.Run("regolith", func(t *testing.T) {
		InitParallel(t)
		cfg := DefaultSystemConfig(t)
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		cfg.DeployConfig.L2GenesisRegolithTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisCanyonTimeOffset = nil
		cfg.DeployConfig.L2GenesisDeltaTimeOffset = nil
		cfg.DeployConfig.L2GenesisEcotoneTimeOffset = nil
		testFees(t, cfg)
	})
	t.Run("ecotone", func(t *testing.T) {
		InitParallel(t)
		cfg := DefaultSystemConfig(t)
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		cfg.DeployConfig.L2GenesisRegolithTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisCanyonTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisDeltaTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisEcotoneTimeOffset = new(hexutil.Uint64)
		testFees(t, cfg)
	})
	t.Run("fjord", func(t *testing.T) {
		InitParallel(t)
		cfg := DefaultSystemConfig(t)
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		cfg.DeployConfig.L2GenesisRegolithTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisCanyonTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisDeltaTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisEcotoneTimeOffset = new(hexutil.Uint64)
		cfg.DeployConfig.L2GenesisFjordTimeOffset = new(hexutil.Uint64)
		testFees(t, cfg)
	})
}

func testFees(t *testing.T, cfg SystemConfig) {
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	l1 := sys.Clients["l1"]

	// Wait for first block after genesis. The genesis block has zero L1Block values and will throw off the GPO checks
	_, err = geth.WaitForBlock(big.NewInt(1), l2Verif, time.Minute)
	require.NoError(t, err)

	config := sys.L2Genesis().Config

	sga := &stateGetterAdapter{
		ctx:    context.Background(),
		t:      t,
		client: l2Seq,
	}

	l1CostFn := types.NewL1CostFunc(config, sga)

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	require.NotEqual(t, cfg.DeployConfig.L2OutputOracleProposer, fromAddr)
	require.NotEqual(t, cfg.DeployConfig.BatchSenderAddress, fromAddr)

	// Find gaspriceoracle contract
	gpoContract, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, l2Seq)
	require.Nil(t, err)

	if !sys.RollupConfig.IsEcotone(sys.L2GenesisCfg.Timestamp) {
		overhead, err := gpoContract.Overhead(&bind.CallOpts{})
		require.Nil(t, err, "reading gpo overhead")
		require.Equal(t, overhead.Uint64(), cfg.DeployConfig.GasPriceOracleOverhead, "wrong gpo overhead")

		scalar, err := gpoContract.Scalar(&bind.CallOpts{})
		require.Nil(t, err, "reading gpo scalar")
		feeScalar := cfg.DeployConfig.FeeScalar()
		require.Equal(t, scalar, new(big.Int).SetBytes(feeScalar[:]), "wrong gpo scalar")
	} else {
		_, err := gpoContract.Overhead(&bind.CallOpts{})
		require.ErrorContains(t, err, "deprecated")
		_, err = gpoContract.Scalar(&bind.CallOpts{})
		require.ErrorContains(t, err, "deprecated")
	}

	decimals, err := gpoContract.Decimals(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo decimals")

	require.Equal(t, decimals.Uint64(), uint64(6), "wrong gpo decimals")

	// BaseFee Recipient
	baseFeeRecipientStartBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.BaseFeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.Nil(t, err)

	// L1Fee Recipient
	l1FeeRecipientStartBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.L1FeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.Nil(t, err)

	sequencerFeeVaultStartBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.SequencerFeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.Nil(t, err)

	genesisBlock, err := l2Seq.BlockByNumber(context.Background(), big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.NoError(t, err)

	coinbaseStartBalance, err := l2Seq.BalanceAt(context.Background(), genesisBlock.Coinbase(), big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.NoError(t, err)

	// Simple transfer from signer to random account
	startBalance, err := l2Seq.BalanceAt(context.Background(), fromAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.Nil(t, err)
	require.Greater(t, startBalance.Uint64(), big.NewInt(params.Ether).Uint64())

	transferAmount := big.NewInt(params.Ether)
	gasTip := big.NewInt(10)
	receipt := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = transferAmount
		opts.GasTipCap = gasTip
		opts.Gas = 21000
		opts.GasFeeCap = big.NewInt(200)
		opts.VerifyOnClients(l2Verif)
	})

	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	header, err := l2Seq.HeaderByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)

	coinbaseEndBalance, err := l2Seq.BalanceAt(context.Background(), header.Coinbase, header.Number)
	require.Nil(t, err)

	endBalance, err := l2Seq.BalanceAt(context.Background(), fromAddr, header.Number)
	require.Nil(t, err)

	baseFeeRecipientEndBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.BaseFeeVaultAddr, header.Number)
	require.Nil(t, err)

	l1Header, err := l1.HeaderByNumber(context.Background(), nil)
	require.Nil(t, err)

	l1FeeRecipientEndBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.L1FeeVaultAddr, header.Number)
	require.Nil(t, err)

	sequencerFeeVaultEndBalance, err := l2Seq.BalanceAt(context.Background(), predeploys.SequencerFeeVaultAddr, header.Number)
	require.Nil(t, err)

	// Diff fee recipient + coinbase balances
	baseFeeRecipientDiff := new(big.Int).Sub(baseFeeRecipientEndBalance, baseFeeRecipientStartBalance)
	l1FeeRecipientDiff := new(big.Int).Sub(l1FeeRecipientEndBalance, l1FeeRecipientStartBalance)
	sequencerFeeVaultDiff := new(big.Int).Sub(sequencerFeeVaultEndBalance, sequencerFeeVaultStartBalance)
	coinbaseDiff := new(big.Int).Sub(coinbaseEndBalance, coinbaseStartBalance)

	// Tally L2 Fee
	l2Fee := gasTip.Mul(gasTip, new(big.Int).SetUint64(receipt.GasUsed))
	require.Equal(t, sequencerFeeVaultDiff, coinbaseDiff, "coinbase is always sequencer fee vault")
	require.Equal(t, l2Fee, coinbaseDiff, "l2 fee mismatch")
	require.Equal(t, l2Fee, sequencerFeeVaultDiff)

	// Tally BaseFee
	baseFee := new(big.Int).Mul(header.BaseFee, new(big.Int).SetUint64(receipt.GasUsed))
	require.Equal(t, baseFee, baseFeeRecipientDiff, "base fee mismatch")

	// Tally L1 Fee
	tx, _, err := l2Seq.TransactionByHash(context.Background(), receipt.TxHash)
	require.NoError(t, err, "Should be able to get transaction")
	bytes, err := tx.MarshalBinary()
	require.Nil(t, err)

	l1Fee := l1CostFn(tx.RollupCostData(), header.Time)
	require.Equalf(t, l1Fee, l1FeeRecipientDiff, "L1 fee mismatch: start balance %v, end balance %v", l1FeeRecipientStartBalance, l1FeeRecipientEndBalance)

	gpoEcotone, err := gpoContract.IsEcotone(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsEcotone(header.Time), gpoEcotone, "GPO and chain must have same ecotone view")

	gpoFjord, err := gpoContract.IsFjord(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsFjord(header.Time), gpoFjord, "GPO and chain must have same fjord view")

	gpoL1Fee, err := gpoContract.GetL1Fee(&bind.CallOpts{}, bytes)
	require.Nil(t, err)

	adjustedGPOFee := gpoL1Fee
	if sys.RollupConfig.IsFjord(header.Time) {
		// The fastlz size of the transaction is 102 bytes
		require.Equal(t, uint64(102), tx.RollupCostData().FastLzSize)
		// Which results in both the fjord cost function and GPO using the minimum value for the fastlz regression:
		// Geth Linear Regression: -42.5856 + 102 * 0.8365 = 42.7374
		// GPO Linear Regression: -42.5856 + 170 * 0.8365 = 99.6194
		// The additional 68 (170 vs. 102) is due to the GPO adding 68 bytes to account for the signature.
		require.Greater(t, types.MinTransactionSize.Uint64(), uint64(99))
		// Because of this, we don't need to do any adjustment as the GPO and cost func are both bounded to the minimum value.
		// However, if the fastlz regression output is ever larger than the minimum, this will require an adjustment.
	} else if sys.RollupConfig.IsRegolith(header.Time) {
		// if post-regolith, adjust the GPO fee by removing the overhead it adds because of signature data
		artificialGPOOverhead := big.NewInt(68 * 16) // it adds 68 bytes to cover signature and RLP data
		l1BaseFee := big.NewInt(7)                   // we assume the L1 basefee is the minimum, 7
		// in our case we already include that, so we subtract it, to do a 1:1 comparison
		adjustedGPOFee = new(big.Int).Sub(gpoL1Fee, new(big.Int).Mul(artificialGPOOverhead, l1BaseFee))
	}
	require.Equal(t, l1Fee, adjustedGPOFee, "GPO reports L1 fee mismatch")

	require.Equal(t, receipt.L1Fee, l1Fee, "l1 fee in receipt is correct")
	if !sys.RollupConfig.IsEcotone(header.Time) { // FeeScalar receipt attribute is removed as of Ecotone
		require.Equal(t,
			new(big.Float).Mul(
				new(big.Float).SetInt(l1Header.BaseFee),
				new(big.Float).Mul(new(big.Float).SetInt(receipt.L1GasUsed), receipt.FeeScalar),
			),
			new(big.Float).SetInt(receipt.L1Fee), "fee field in receipt matches gas used times scalar times base fee")
	}

	// Calculate total fee
	baseFeeRecipientDiff.Add(baseFeeRecipientDiff, coinbaseDiff)
	totalFee := new(big.Int).Add(baseFeeRecipientDiff, l1FeeRecipientDiff)
	balanceDiff := new(big.Int).Sub(startBalance, endBalance)
	balanceDiff.Sub(balanceDiff, transferAmount)
	require.Equal(t, balanceDiff, totalFee, "balances should add up")
}

func StopStartBatcher(t *testing.T, deltaTimeOffset *hexutil.Uint64) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = deltaTimeOffset
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")
	defer sys.Close()

	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["verifier"].HTTPEndpoint())
	require.NoError(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// retrieve the initial sync status
	seqStatus, err := rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)

	nonce := uint64(0)
	sendTx := func() *types.Receipt {
		// Submit TX to L2 sequencer node
		receipt := SendL2Tx(t, cfg, l2Seq, cfg.Secrets.Alice, func(opts *TxOpts) {
			opts.ToAddr = &common.Address{0xff, 0xff}
			opts.Value = big.NewInt(1_000_000_000)
			opts.Nonce = nonce
		})
		nonce++
		return receipt
	}
	// send a transaction
	receipt := sendTx()

	// wait until the block the tx was first included in shows up in the safe chain on the verifier
	safeBlockInclusionDuration := time.Duration(6*cfg.DeployConfig.L1BlockTime) * time.Second
	_, err = geth.WaitForBlock(receipt.BlockNumber, l2Verif, safeBlockInclusionDuration)
	require.NoError(t, err, "Waiting for block on verifier")
	require.NoError(t, wait.ForProcessingFullBatch(context.Background(), rollupClient))

	// ensure the safe chain advances
	newSeqStatus, err := rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Greater(t, newSeqStatus.SafeL2.Number, seqStatus.SafeL2.Number, "Safe chain did not advance")

	// stop the batch submission
	err = sys.BatchSubmitter.Driver().StopBatchSubmitting(context.Background())
	require.NoError(t, err)

	// wait for any old safe blocks being submitted / derived
	time.Sleep(safeBlockInclusionDuration)

	// get the initial sync status
	seqStatus, err = rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)

	// send another tx
	sendTx()
	time.Sleep(safeBlockInclusionDuration)

	// ensure that the safe chain does not advance while the batcher is stopped
	newSeqStatus, err = rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, newSeqStatus.SafeL2.Number, seqStatus.SafeL2.Number, "Safe chain advanced while batcher was stopped")

	// start the batch submission
	err = sys.BatchSubmitter.Driver().StartBatchSubmitting()
	require.NoError(t, err)
	time.Sleep(safeBlockInclusionDuration)

	// send a third tx
	receipt = sendTx()

	// wait until the block the tx was first included in shows up in the safe chain on the verifier
	_, err = geth.WaitForBlock(receipt.BlockNumber, l2Verif, safeBlockInclusionDuration)
	require.NoError(t, err, "Waiting for block on verifier")
	require.NoError(t, wait.ForProcessingFullBatch(context.Background(), rollupClient))

	// ensure that the safe chain advances after restarting the batcher
	newSeqStatus, err = rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Greater(t, newSeqStatus.SafeL2.Number, seqStatus.SafeL2.Number, "Safe chain did not advance after batcher was restarted")
}

func TestBatcherMultiTx(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.MaxPendingTransactions = 0 // no limit on parallel txs
	// ensures that batcher txs are as small as possible
	cfg.BatcherMaxL1TxSizeBytes = derive.FrameV0OverHeadSize + 1 /*version bytes*/ + 1
	cfg.DisableBatcher = true
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]

	_, err = geth.WaitForBlock(big.NewInt(10), l2Seq, time.Duration(cfg.DeployConfig.L2BlockTime*15)*time.Second)
	require.NoError(t, err, "Waiting for L2 blocks")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l1Number, err := l1Client.BlockNumber(ctx)
	require.NoError(t, err)

	// start batch submission
	err = sys.BatchSubmitter.Driver().StartBatchSubmitting()
	require.NoError(t, err)

	totalTxCount := 0
	// wait for up to 10 L1 blocks, usually only 3 is required, but it's
	// possible additional L1 blocks will be created before the batcher starts,
	// so we wait additional blocks.
	for i := int64(0); i < 10; i++ {
		block, err := geth.WaitForBlock(big.NewInt(int64(l1Number)+i), l1Client, time.Duration(cfg.DeployConfig.L1BlockTime*5)*time.Second)
		require.NoError(t, err, "Waiting for l1 blocks")
		totalTxCount += len(block.Transactions())

		if totalTxCount >= 10 {
			return
		}
	}

	t.Fatal("Expected at least 10 transactions from the batcher")
}

func latestBlock(t *testing.T, client *ethclient.Client) uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	blockAfter, err := client.BlockNumber(ctx)
	require.Nil(t, err, "Error getting latest block")
	return blockAfter
}

// TestPendingBlockIsLatest tests that we serve the latest block as pending block
func TestPendingBlockIsLatest(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
	l2Seq := sys.Clients["sequencer"]

	t.Run("block", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			// TODO(CLI-4044): pending-block ID change
			pending, err := l2Seq.BlockByNumber(context.Background(), big.NewInt(-1))
			require.NoError(t, err)
			latest, err := l2Seq.BlockByNumber(context.Background(), nil)
			require.NoError(t, err)
			if pending.NumberU64() == latest.NumberU64() {
				require.Equal(t, pending.Hash(), latest.Hash(), "pending must exactly match latest block")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending block with same number as latest block")
	})
	t.Run("header", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			// TODO(CLI-4044): pending-block ID change
			pending, err := l2Seq.HeaderByNumber(context.Background(), big.NewInt(-1))
			require.NoError(t, err)
			latest, err := l2Seq.HeaderByNumber(context.Background(), nil)
			require.NoError(t, err)
			if pending.Number.Uint64() == latest.Number.Uint64() {
				require.Equal(t, pending.Hash(), latest.Hash(), "pending must exactly match latest header")
				return
			}
			// re-try until we have the same number, as the requests are not an atomic bundle, and the sequencer may create a block.
		}
		t.Fatal("failed to get pending header with same number as latest header")
	})
}

func TestRuntimeConfigReload(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	// to speed up the test, make it reload the config more often, and do not impose a long conf depth
	cfg.Nodes["verifier"].RuntimeConfigReloadInterval = time.Second * 5
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = 1

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
	initialRuntimeConfig := sys.RollupNodes["verifier"].RuntimeConfig()

	// close the EL node, since we want to block derivation, to solely rely on the reloading mechanism for updates.
	sys.EthInstances["verifier"].Close()

	l1 := sys.Clients["l1"]

	// Change the system-config via L1
	sysCfgContract, err := bindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1)
	require.NoError(t, err)
	newUnsafeBlocksSigner := common.Address{0x12, 0x23, 0x45}
	require.NotEqual(t, initialRuntimeConfig.P2PSequencerAddress(), newUnsafeBlocksSigner, "changing to a different address")
	opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.SysCfgOwner, cfg.L1ChainIDBig())
	require.Nil(t, err)
	// the unsafe signer address is part of the runtime config
	tx, err := sysCfgContract.SetUnsafeBlockSigner(opts, newUnsafeBlocksSigner)
	require.NoError(t, err)

	// wait for the change to confirm
	_, err = wait.ForReceiptOK(context.Background(), l1, tx.Hash())
	require.NoError(t, err)

	// wait for the address to change
	_, err = retry.Do(context.Background(), 10, retry.Fixed(time.Second*10), func() (struct{}, error) {
		v := sys.RollupNodes["verifier"].RuntimeConfig().P2PSequencerAddress()
		if v == newUnsafeBlocksSigner {
			return struct{}{}, nil
		}
		return struct{}{}, fmt.Errorf("no change yet, seeing %s but looking for %s", v, newUnsafeBlocksSigner)
	})
	require.NoError(t, err)
}

func TestRecommendedProtocolVersionChange(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	require.NotEqual(t, common.Address{}, cfg.L1Deployments.ProtocolVersions, "need ProtocolVersions contract deployment")
	// to speed up the test, make it reload the config more often, and do not impose a long conf depth
	cfg.Nodes["verifier"].RuntimeConfigReloadInterval = time.Second * 5
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = 1

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
	runtimeConfig := sys.RollupNodes["verifier"].RuntimeConfig()

	// Change the superchain-config via L1
	l1 := sys.Clients["l1"]

	_, build, major, minor, patch, preRelease := params.OPStackSupport.Parse()
	newRecommendedProtocolVersion := params.ProtocolVersionV0{Build: build, Major: major + 1, Minor: minor, Patch: patch, PreRelease: preRelease}.Encode()
	require.NotEqual(t, runtimeConfig.RecommendedProtocolVersion(), newRecommendedProtocolVersion, "changing to a different protocol version")

	protVersions, err := bindings.NewProtocolVersions(cfg.L1Deployments.ProtocolVersionsProxy, l1)
	require.NoError(t, err)

	// ProtocolVersions contract is owned by same key as SystemConfig in devnet
	opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.SysCfgOwner, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Change recommended protocol version
	tx, err := protVersions.SetRecommended(opts, new(big.Int).SetBytes(newRecommendedProtocolVersion[:]))
	require.NoError(t, err)

	// wait for the change to confirm
	_, err = wait.ForReceiptOK(context.Background(), l1, tx.Hash())
	require.NoError(t, err)

	// wait for the recommended protocol version to change
	_, err = retry.Do(context.Background(), 10, retry.Fixed(time.Second*10), func() (struct{}, error) {
		v := sys.RollupNodes["verifier"].RuntimeConfig().RecommendedProtocolVersion()
		if v == newRecommendedProtocolVersion {
			return struct{}{}, nil
		}
		return struct{}{}, fmt.Errorf("no change yet, seeing %s but looking for %s", v, newRecommendedProtocolVersion)
	})
	require.NoError(t, err)
}

func TestRequiredProtocolVersionChangeAndHalt(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	// to speed up the test, make it reload the config more often, and do not impose a long conf depth
	cfg.Nodes["verifier"].RuntimeConfigReloadInterval = time.Second * 5
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = 1
	// configure halt in verifier op-node
	cfg.Nodes["verifier"].RollupHalt = "major"
	// configure halt in verifier op-geth node
	cfg.GethOptions["verifier"] = append(cfg.GethOptions["verifier"], []geth.GethOption{
		func(ethCfg *ethconfig.Config, nodeCfg *node.Config) error {
			ethCfg.RollupHaltOnIncompatibleProtocolVersion = "major"
			return nil
		},
	}...)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
	runtimeConfig := sys.RollupNodes["verifier"].RuntimeConfig()

	// Change the superchain-config via L1
	l1 := sys.Clients["l1"]

	_, build, major, minor, patch, preRelease := params.OPStackSupport.Parse()
	newRequiredProtocolVersion := params.ProtocolVersionV0{Build: build, Major: major + 1, Minor: minor, Patch: patch, PreRelease: preRelease}.Encode()
	require.NotEqual(t, runtimeConfig.RequiredProtocolVersion(), newRequiredProtocolVersion, "changing to a different protocol version")

	protVersions, err := bindings.NewProtocolVersions(cfg.L1Deployments.ProtocolVersionsProxy, l1)
	require.NoError(t, err)

	// ProtocolVersions contract is owned by same key as SystemConfig in devnet
	opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.SysCfgOwner, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Change required protocol version
	tx, err := protVersions.SetRequired(opts, new(big.Int).SetBytes(newRequiredProtocolVersion[:]))
	require.NoError(t, err)

	// wait for the change to confirm
	_, err = wait.ForReceiptOK(context.Background(), l1, tx.Hash())
	require.NoError(t, err)

	// wait for the required protocol version to take effect by halting the verifier that opted in, and halting the op-geth node that opted in.
	_, err = retry.Do(context.Background(), 10, retry.Fixed(time.Second*10), func() (struct{}, error) {
		if !sys.RollupNodes["verifier"].Stopped() {
			return struct{}{}, errors.New("verifier rollup node is not closed yet")
		}
		return struct{}{}, nil
	})
	require.NoError(t, err)
	t.Log("verified that op-node closed!")
	// Checking if the engine is down is not trivial in op-e2e.
	// In op-geth we have halting tests covering the Engine API, in op-e2e we instead check if the API stops.
	_, err = retry.Do(context.Background(), 10, retry.Fixed(time.Second*10), func() (struct{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := sys.Clients["verifier"].ChainID(ctx)
		cancel()
		if err != nil && !errors.Is(err, ctx.Err()) { // waiting for client to stop responding to chainID requests
			return struct{}{}, nil
		}
		return struct{}{}, errors.New("verifier rollup node is not closed yet")
	})
	require.NoError(t, err)
	t.Log("verified that op-geth closed!")
}
