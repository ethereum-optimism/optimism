package op_e2e

import (
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	gethutils "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestSystem4844E2E runs the SystemE2E test with 4844 enabled on L1,
// and active on the rollup in the op-batcher and verifier.
func TestSystem4844E2E(t *testing.T) {
	t.Run("single-blob", func(t *testing.T) { testSystem4844E2E(t, false) })
	t.Run("multi-blob", func(t *testing.T) { testSystem4844E2E(t, true) })
}

func testSystem4844E2E(t *testing.T, multiBlob bool) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DataAvailabilityType = batcherFlags.BlobsType
	const maxBlobs = 6
	var maxL1TxSize int
	if multiBlob {
		cfg.BatcherTargetNumFrames = 6
		cfg.BatcherUseMaxTxSizeForBlobs = true
		// leads to 6 blobs for an L2 block with a user tx with 400 random bytes
		// while all other L2 blocks take 1 blob (deposit tx)
		maxL1TxSize = derive.FrameV0OverHeadSize + 100
		cfg.BatcherMaxL1TxSizeBytes = uint64(maxL1TxSize)
	}

	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisActivation

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Send Transaction & wait for success
	fromAddr := cfg.Secrets.Addresses().Alice
	log.Info("alice", "addr", fromAddr)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	// Send deposit transaction
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.NoError(t, err)
	mintAmount := big.NewInt(1_000_000_000_000)
	opts.Value = mintAmount
	SendDepositTx(t, cfg, l1Client, l2Verif, opts, func(l2Opts *DepositTxOpts) {})

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	endBalance, err := wait.ForBalanceChange(ctx, l2Verif, fromAddr, startBalance)
	require.NoError(t, err)

	diff := new(big.Int).Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change")

	// Submit TX to L2 sequencer node
	receipt := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.Value = big.NewInt(1_000_000_000)
		opts.Nonce = 1 // Already have deposit
		opts.ToAddr = &common.Address{0xff, 0xff}
		// put some random data in the tx to make it fill up 6 blobs (multi-blob case)
		opts.Data = testutils.RandomData(rand.New(rand.NewSource(420)), 400)
		opts.Gas, err = core.IntrinsicGas(opts.Data, nil, false, true, true, false)
		require.NoError(t, err)
		opts.VerifyOnClients(l2Verif)
	})

	// Verify blocks match after batch submission on verifiers and sequencers
	verifBlock, err := l2Verif.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, verifBlock.Hash(), receipt.BlockHash, "must be same block")
	seqBlock, err := l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, seqBlock.Hash(), receipt.BlockHash, "must be same block")
	require.Equal(t, verifBlock.NumberU64(), seqBlock.NumberU64(), "Verifier and sequencer blocks not the same after including a batch tx")
	require.Equal(t, verifBlock.ParentHash(), seqBlock.ParentHash(), "Verifier and sequencer blocks parent hashes not the same after including a batch tx")
	require.Equal(t, verifBlock.Hash(), seqBlock.Hash(), "Verifier and sequencer blocks not the same after including a batch tx")

	rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
	require.NoError(t, err)
	rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))
	// basic check that sync status works
	seqStatus, err := rollupClient.SyncStatus(context.Background())
	require.NoError(t, err)
	require.LessOrEqual(t, seqBlock.NumberU64(), seqStatus.UnsafeL2.Number)
	// basic check that version endpoint works
	seqVersion, err := rollupClient.Version(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, "", seqVersion)

	// quick check that the batch submitter works
	require.Eventually(t, func() bool {
		// wait for chain to be marked as "safe" (i.e. confirm batch-submission works)
		stat, err := rollupClient.SyncStatus(context.Background())
		require.NoError(t, err)
		return stat.SafeL2.Number >= receipt.BlockNumber.Uint64()
	}, time.Second*20, time.Second, "expected L2 to be batch-submitted and labeled as safe")

	// check that the L2 tx is still canonical
	seqBlock, err = l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, seqBlock.Hash(), receipt.BlockHash, "receipt block must match canonical block at tx inclusion height")

	// find L1 block that contained the blob(s) batch tx
	tip, err := l1Client.HeaderByNumber(context.Background(), nil)
	require.NoError(t, err)
	var blobTx *types.Transaction
	blobBlock, err := gethutils.FindBlock(l1Client, int(tip.Number.Int64()), 0, 5*time.Second,
		func(b *types.Block) (bool, error) {
			for _, tx := range b.Transactions() {
				if tx.Type() != types.BlobTxType {
					continue
				}
				// expect to find at least one tx with multiple blobs in multi-blob case
				if !multiBlob || len(tx.BlobHashes()) > 1 {
					blobTx = tx
					return true, nil
				}
			}
			return false, nil
		})
	require.NoError(t, err)

	numBlobs := len(blobTx.BlobHashes())
	if !multiBlob {
		require.NotZero(t, numBlobs, "single-blob: expected to find L1 blob tx")
	} else {
		require.Equal(t, maxBlobs, numBlobs, "multi-blob: expected to find L1 blob tx with 6 blobs")
		// blob tx should have filled up all but last blob
		bcl := sys.L1BeaconHTTPClient()
		hashes := toIndexedBlobHashes(blobTx.BlobHashes()...)
		sidecars, err := bcl.BeaconBlobSideCars(context.Background(), false, sys.L1Slot(blobBlock.Time()), hashes)
		require.NoError(t, err)
		require.Len(t, sidecars.Data, maxBlobs)
		for i := 0; i < maxBlobs-1; i++ {
			data, err := sidecars.Data[i].Blob.ToData()
			require.NoError(t, err)
			require.Len(t, data, maxL1TxSize)
		}
		// last blob should only be partially filled
		data, err := sidecars.Data[maxBlobs-1].Blob.ToData()
		require.NoError(t, err)
		require.Less(t, len(data), maxL1TxSize)
	}
}

func toIndexedBlobHashes(hs ...common.Hash) []eth.IndexedBlobHash {
	hashes := make([]eth.IndexedBlobHash, 0, len(hs))
	for i, hash := range hs {
		hashes = append(hashes, eth.IndexedBlobHash{Index: uint64(i), Hash: hash})
	}
	return hashes
}
