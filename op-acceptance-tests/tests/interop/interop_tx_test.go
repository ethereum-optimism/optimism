package interop

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	sdktypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func messagePassingScenario(sourceChainIdx, destChainIdx uint64, sourceWalletGetter, destWalletGetter validators.WalletGetter) systest.InteropSystemTestFunc {
	return func(t systest.T, sys system.InteropSystem) {
		ctx := t.Context()

		logger := testlog.Logger(t, log.LevelInfo)
		logger = logger.With("test", "TestInitiateMessage", "devnet", sys.Identifier())

		chainA := sys.L2s()[sourceChainIdx]
		chainB := sys.L2s()[destChainIdx]

		logger = logger.With("sourceChain", chainA.ID(), "destChain", chainB.ID())

		// userA is funded at chainA and want to initialize message at chain A
		userA := sourceWalletGetter(ctx)
		// userB is funded at chainB and want to execute message to chainB
		userB := destWalletGetter(ctx)

		// Initiate message
		dummyAddress := common.Address{0x13, 0x37}
		dummyMessage := []byte{0x13, 0x33, 0x33, 0x37}
		logger.Info("Initiate message", "address", dummyAddress, "message", dummyMessage)
		initResult := userA.InitiateMessage(chainB.ID(), dummyAddress, dummyMessage).Send(ctx)
		require.NoError(t, initResult.Wait())

		initReceipt, ok := initResult.Info().(system.Receipt)
		require.True(t, ok)
		logger.Info("Execute message", "txHash", initReceipt.TxHash().Hex())
		logs := initReceipt.Logs()
		require.Equal(t, 1, len(logs), "expected single log")
		log := logs[0]

		blockNumber := initReceipt.BlockNumber()
		block, err := chainA.Node().BlockByNumber(ctx, blockNumber)
		require.NoError(t, err)
		blockTime := big.NewInt(int64(block.Time()))
		sentMessage := []byte{}
		for _, topic := range log.Topics {
			sentMessage = append(sentMessage, topic.Bytes()...)
		}
		sentMessage = append(sentMessage, log.Data...)
		logger.Info("Execute message", "sentMessage", sentMessage)

		logIndex := big.NewInt(int64(log.Index))
		identifier := bindings.Identifier{
			Origin:      constants.L2ToL2CrossDomainMessenger,
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   blockTime,
			ChainId:     chainA.ID(),
		}
		logger.Info("Execute message", "identifier", identifier)

		logger.Info("Execute message", "address", dummyAddress, "message", dummyMessage)
		execResult := userB.ExecuteMessage(identifier, sentMessage).Send(ctx)
		require.NoError(t, execResult.Wait())

		execReceipt, ok := execResult.Info().(system.Receipt)
		require.True(t, ok)

		logger.Info("Execute message", "txHash", execReceipt.TxHash().Hex())
	}
}

func TestInteropSystemInitiateMessage(t *testing.T) {
	sourceChainIdx := uint64(0)
	destChainIdx := uint64(1)
	sourceWalletGetter, sourcefundsValidator := validators.AcquireL2WalletWithFunds(sourceChainIdx, sdktypes.NewBalance(big.NewInt(1.0*constants.ETH)))
	destWalletGetter, destfundsValiator := validators.AcquireL2WalletWithFunds(destChainIdx, sdktypes.NewBalance(big.NewInt(1.0*constants.ETH)))
	systest.InteropSystemTest(t,
		messagePassingScenario(sourceChainIdx, destChainIdx, sourceWalletGetter, destWalletGetter),
		sourcefundsValidator,
		destfundsValiator,
	)
}
