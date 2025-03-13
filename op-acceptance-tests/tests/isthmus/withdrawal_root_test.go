package isthmus

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/lmittmann/w3"
)

func TestWithdrawalsRoot(t *testing.T) {
	chainIdx := uint64(0) // We'll use the first L2 chain for this test

	walletGetter, fundsValidator := validators.AcquireL2WalletWithFunds(
		chainIdx,
		types.NewBalance(big.NewInt(1.0*constants.ETH)),
	)
	llsysGetter, llsysValidator := validators.AcquireLowLevelSystem()
	_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Isthmus)

	systest.SystemTest(t,
		withdrawalRootTestScenario(chainIdx, walletGetter, llsysGetter),
		fundsValidator,
		llsysValidator,
		forkValidator,
	)
}

func withdrawalRootTestScenario(chainIdx uint64, walletGetter validators.WalletGetter, llsysGetter validators.LowLevelSystemGetter) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()

		llsys := llsysGetter(ctx)
		chain := llsys.L2s()[chainIdx]
		gethCl, err := chain.GethClient()
		require.NoError(t, err)

		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started test")

		user := walletGetter(ctx)

		// Sad eth clients
		rpcCl, err := client.NewRPC(ctx, logger, chain.RPCURL())
		require.NoError(t, err)
		t.Cleanup(rpcCl.Close)
		ethCl, err := sources.NewEthClient(rpcCl, logger, nil, sources.DefaultEthClientConfig(10))
		require.NoError(t, err)

		// Determine pre-state
		preBlock, err := gethCl.BlockByNumber(ctx, nil)
		require.NoError(t, err)
		logger.Info("Got pre-state block", "hash", preBlock.Hash(), "number", preBlock.Number())

		preBlockHash := preBlock.Hash()
		preProof, err := ethCl.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, nil, preBlockHash.String())
		require.NoError(t, err)
		preWithdrawalsRoot := preProof.StorageHash

		logger.Info("Got pre proof", "storage hash", preWithdrawalsRoot)

		// check isthmus withdrawals-root in the block matches the state
		gotPre := preBlock.WithdrawalsRoot()
		require.NotNil(t, gotPre)
		require.Equal(t, preWithdrawalsRoot, *gotPre, "withdrawals root in block is what we expect")

		chainID := (*big.Int)(chain.ID())
		signer := gtypes.LatestSignerForChainID(chainID)
		priv := user.PrivateKey()
		require.NoError(t, err)

		// construct call input, ugly but no bindings...
		funcInitiateWithdrawal := w3.MustNewFunc(`initiateWithdrawal(address, uint256, bytes memory)`, "")
		args, err := funcInitiateWithdrawal.EncodeArgs(
			common.Address{},
			big.NewInt(1_000_000),
			[]byte{},
		)
		require.NoError(t, err)

		// Try to simulate the transaction first to check for errors
		gasLimit, err := gethCl.EstimateGas(ctx, ethereum.CallMsg{
			From:  user.Address(),
			To:    &predeploys.L2ToL1MessagePasserAddr,
			Value: big.NewInt(0),
			Data:  args,
		})
		require.NoError(t, err, "Gas estimation failed")

		nonce, err := gethCl.PendingNonceAt(ctx, user.Address())
		require.NoError(t, err)

		gasPrice, err := gethCl.SuggestGasPrice(ctx)
		require.NoError(t, err, "failed to suggest gas price")

		tip, err := gethCl.SuggestGasTipCap(ctx)
		require.NoError(t, err, "error getting gas tip cap")

		tx, err := gtypes.SignNewTx(priv, signer, &gtypes.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: new(big.Int).Add(tip, new(big.Int).Mul(gasPrice, big.NewInt(2))),
			Gas:       gasLimit,
			To:        &predeploys.L2ToL1MessagePasserAddr,
			Value:     big.NewInt(0),
			Data:      args,
		})
		require.NoError(t, err, "sign tx")

		err = gethCl.SendTransaction(ctx, tx)
		require.NoError(t, err, "send tx")

		// Find when the withdrawal waskincluded
		rec, err := wait.ForReceipt(ctx, gethCl, tx.Hash(), gtypes.ReceiptStatusSuccessful)
		require.NoError(t, err)

		// Load the storage at this particular block
		postBlockHash := rec.BlockHash
		postProof, err := ethCl.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, nil, postBlockHash.String())
		require.NoError(t, err, "Error getting L2ToL1MessagePasser contract proof")
		postWithdrawalsRoot := postProof.StorageHash

		// Check that the withdrawals-root changed
		require.NotEqual(t, preWithdrawalsRoot, postWithdrawalsRoot, "withdrawals storage root changes")

		postBlock, err := gethCl.BlockByHash(ctx, postBlockHash)
		require.NoError(t, err)
		logger.Info("Got post-state block", "hash", postBlock.Hash(), "number", postBlock.Number())

		gotPost := postBlock.WithdrawalsRoot()
		require.NotNil(t, gotPost)
		require.Equal(t, postWithdrawalsRoot, *gotPost, "block contains new withdrawals root")

		logger.Info("Withdrawals root test passed")
	}
}
