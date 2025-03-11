package interop

import (
	"math/big"
	"testing"

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

func initiateMessageScenario(sourceChainIdx, destChainIdx uint64, walletGetter validators.WalletGetter) systest.InteropSystemTestFunc {
	return func(t systest.T, sys system.InteropSystem) {
		ctx := t.Context()

		logger := testlog.Logger(t, log.LevelInfo)
		logger = logger.With("test", "TestInitiateMessage", "devnet", sys.Identifier())

		chainA := sys.L2s()[sourceChainIdx]
		chainB := sys.L2s()[destChainIdx]

		logger = logger.With("sourceChain", chainA.ID(), "destChain", chainB.ID())

		// userA is funded at chainA and want to send message to chainB
		userA := walletGetter(ctx)

		// Initiate message
		dummyAddress := common.Address{0x13, 0x37}
		dummyMessage := []byte{0x13, 0x33, 0x33, 0x37}
		logger.Info("Initiate message", "address", dummyAddress, "message", dummyMessage)
		require.NoError(t, userA.InitiateMessage(chainB.ID(), dummyAddress, dummyMessage).Send(ctx).Wait())
	}
}

func TestInteropSystemInitiateMessage(t *testing.T) {
	sourceChainIdx := uint64(0)
	destChainIdx := uint64(1)
	walletGetter, fundsValidator := validators.AcquireL2WalletWithFunds(sourceChainIdx, sdktypes.NewBalance(big.NewInt(1.0*constants.ETH)))

	systest.InteropSystemTest(t,
		initiateMessageScenario(sourceChainIdx, destChainIdx, walletGetter),
		fundsValidator,
	)
}
