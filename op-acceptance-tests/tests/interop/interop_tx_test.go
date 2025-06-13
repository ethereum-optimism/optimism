package interop

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	sdktypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// SatisfyExecMsgConstraint waits supervisor to index initiating message,
// and destination chain has advanced enough to satisfy messaging time invariant.
func SatisfyExecMsgConstraint(t systest.T, logger log.Logger, sys system.InteropSystem, initBlockNum uint64, initTimestamp uint64) {
	ctx := t.Context()

	supervisor, err := sys.Supervisor(ctx)
	require.NoError(t, err)
	sourceChain, destChain := sys.L2s()[0], sys.L2s()[1]

	elclientDest, err := destChain.Nodes()[0].GethClient()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		syncStatus, err := supervisor.SyncStatus(ctx)
		require.NoError(t, err)
		chainAView, ok := syncStatus.Chains[eth.ChainIDFromBig(sourceChain.ID())]
		require.True(t, ok)
		chainBView, ok := syncStatus.Chains[eth.ChainIDFromBig(destChain.ID())]
		require.True(t, ok)
		blockB, err := elclientDest.BlockByNumber(ctx, big.NewInt(int64(chainBView.LocalUnsafe.Number)))
		require.NoError(t, err)
		checkA := chainAView.LocalUnsafe.Number >= initBlockNum
		checkB := blockB.Header().Time >= initTimestamp
		logger.Info("wait until supervisor indexes source chain block with initiating message", "check", checkA, "supervisor", chainAView.LocalUnsafe.Number, "chainA", initBlockNum)
		logger.Info("wait until supervisor indexes dest chain head which passes timestamp invariant", "check", checkB, "supervisor", blockB.Header().Time, "chainA", initTimestamp)
		return checkA && checkB
	}, 20*time.Second, 2*time.Second)
}

func messagePassingScenario(sourceChainIdx, destChainIdx uint64, sourceWalletGetter, destWalletGetter validators.WalletGetter) systest.InteropSystemTestFunc {
	return func(t systest.T, sys system.InteropSystem) {
		ctx := t.Context()

		logger := testlog.Logger(t, log.LevelInfo)
		logger = logger.With("test", "TestMessagePassing", "devnet", sys.Identifier())

		chainA := sys.L2s()[sourceChainIdx]
		chainB := sys.L2s()[destChainIdx]

		logger.Info("chain info", "sourceChain", chainA.ID(), "destChain", chainB.ID())
		elclientA, err := chainA.Nodes()[0].GethClient()
		require.NoError(t, err)
		elclientB, err := chainB.Nodes()[0].GethClient()
		require.NoError(t, err)

		logger.Info("wait until both chains pass genesis")
		require.Eventually(t, func() bool {
			chainAUnsafeHeadNum, err := elclientA.BlockNumber(ctx)
			require.NoError(t, err)
			chainBUnsafeHeadNum, err := elclientB.BlockNumber(ctx)
			require.NoError(t, err)
			check := chainAUnsafeHeadNum > 0 && chainBUnsafeHeadNum > 0
			logger.Info("wait until both chains pass genesis", "check", check, "chainA", chainAUnsafeHeadNum, "chainB", chainBUnsafeHeadNum)
			return check
		}, 3*time.Minute, 2*time.Second)

		// userA is funded at chainA and want to initialize message at chain A
		userA := sourceWalletGetter(ctx)
		// userB is funded at chainB and want to execute message to chainB
		userB := destWalletGetter(ctx)

		sha256PrecompileAddr := common.BytesToAddress([]byte{0x2})
		dummyMessage := []byte("l33t message")

		// Initiate message
		logger.Info("Initiate message", "address", sha256PrecompileAddr, "message", dummyMessage)
		initResult := userA.InitiateMessage(chainB.ID(), sha256PrecompileAddr, dummyMessage).Send(ctx)
		require.NoError(t, initResult.Wait())

		initReceipt, ok := initResult.Info().(system.Receipt)
		require.True(t, ok)
		logger.Info("Initiate message", "txHash", initReceipt.TxHash().Hex())
		logs := initReceipt.Logs()
		// We are directly calling sendMessage, so we expect single log for SentMessage event
		require.Equal(t, 1, len(logs), "expected single log")
		log := logs[0]

		// Build sentMessage for message execution
		blockNumber := initReceipt.BlockNumber()
		blockA, err := chainA.Nodes()[0].BlockByNumber(ctx, blockNumber)
		require.NoError(t, err)
		blockTimeA := big.NewInt(int64(blockA.Time()))
		logger.Info("Initiate message was included at", "timestamp", blockTimeA.String())

		sentMessage := []byte{}
		for _, topic := range log.Topics {
			sentMessage = append(sentMessage, topic.Bytes()...)
		}
		sentMessage = append(sentMessage, log.Data...)

		// Build identifier for message execution
		logIndex := big.NewInt(int64(log.Index))
		identifier := bindings.Identifier{
			Origin:      constants.L2ToL2CrossDomainMessenger,
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   blockTimeA,
			ChainId:     chainA.ID(),
		}

		SatisfyExecMsgConstraint(t, logger, sys, identifier.BlockNumber.Uint64(), identifier.Timestamp.Uint64())

		// Execute message
		logger.Info("Execute message", "address", sha256PrecompileAddr, "message", dummyMessage)
		execResult := userB.ExecuteMessage(identifier, sentMessage).Send(ctx)
		require.NoError(t, execResult.Wait())

		execReceipt, ok := execResult.Info().(system.Receipt)
		require.True(t, ok)

		execTxHash := execReceipt.TxHash()
		logger.Info("Execute message", "txHash", execTxHash.Hex())

		blockNumberB := execReceipt.BlockNumber()
		blockB, err := chainB.Nodes()[0].BlockByNumber(ctx, blockNumberB)
		require.NoError(t, err)
		blockTimeB := big.NewInt(int64(blockB.Time()))
		logger.Info("Execute message was included at", "timestamp", blockTimeB.String())

		// Validation that message has passed and got executed successfully
		gethClient, err := chainB.Nodes()[0].GethClient()
		require.NoError(t, err)

		trace, err := wait.DebugTraceTx(ctx, gethClient, execTxHash)
		require.NoError(t, err)

		precompile := vm.PrecompiledContractsHomestead[sha256PrecompileAddr]
		expected, err := precompile.Run(dummyMessage)
		require.NoError(t, err)
		logger.Info("sha256 computed offchain", "value", hex.EncodeToString(expected))

		// length of sha256 image is 32
		output := trace.CallTrace.Output
		require.GreaterOrEqual(t, len(output), 32)
		actual := []byte(output[len(output)-32:])
		logger.Info("sha256 computed onchain", "value", hex.EncodeToString(actual))

		require.Equal(t, expected, actual)
	}
}

// TestMessagePassing checks the basic functionality of message passing two interoperable chains.
// Scenario: Source chain initiates message to make destination chain execute sha256 precompile.
func TestMessagePassing(t *testing.T) {
	sourceChainIdx := uint64(0)
	destChainIdx := uint64(1)
	sourceWalletGetter, sourcefundsValidator := validators.AcquireL2WalletWithFunds(sourceChainIdx, sdktypes.NewBalance(big.NewInt(0.1*constants.ETH)))
	destWalletGetter, destfundsValidator := validators.AcquireL2WalletWithFunds(destChainIdx, sdktypes.NewBalance(big.NewInt(0.1*constants.ETH)))

	systest.InteropSystemTest(t,
		messagePassingScenario(sourceChainIdx, destChainIdx, sourceWalletGetter, destWalletGetter),
		sourcefundsValidator,
		destfundsValidator,
	)
}
