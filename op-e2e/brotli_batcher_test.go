package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func setupAliceAccount(t *testing.T, cfg SystemConfig, sys *System, ethPrivKey *ecdsa.PrivateKey) {
	l1Client := sys.Clients["l1"]
	l2Verif := sys.Clients["verifier"]

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
}

func TestBrotliBatcherFjord(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DataAvailabilityType = batcherFlags.BlobsType

	genesisActivation := hexutil.Uint64(0)
	cfg.DeployConfig.L1CancunTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisActivation
	cfg.DeployConfig.L2GenesisFjordTimeOffset = &genesisActivation

	// set up batcher to use brotli
	sys, err := cfg.Start(t, SystemConfigOption{"compressionAlgo", "brotli", nil})
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account and set up the account
	ethPrivKey := cfg.Secrets.Alice
	setupAliceAccount(t, cfg, sys, ethPrivKey)

	// Submit TX to L2 sequencer node
	receipt := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.Value = big.NewInt(1_000_000_000)
		opts.Nonce = 1 // Already have deposit
		opts.ToAddr = &common.Address{0xff, 0xff}
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

	// quick check that the batch submitter works
	require.Eventually(t, func() bool {
		// wait for chain to be marked as "safe" (i.e. confirm batch-submission works)
		stat, err := rollupClient.SyncStatus(context.Background())
		require.NoError(t, err)
		return stat.SafeL2.Number >= receipt.BlockNumber.Uint64()
	}, time.Second*20, time.Second, "expected L2 to be batch-submitted and labeled as safe")

	// check that the L2 tx is still canonical
	// safe and canonical => the block was batched successfully with brotli
	seqBlock, err = l2Seq.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.NoError(t, err)
	require.Equal(t, seqBlock.Hash(), receipt.BlockHash, "receipt block must match canonical block at tx inclusion height")
}
