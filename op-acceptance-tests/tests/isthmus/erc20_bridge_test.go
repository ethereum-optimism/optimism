package isthmus

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	sdktypes "github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestERC20Bridge(t *testing.T) {
	chainIdx := uint64(0) // We'll use the first L2 chain for this test

	l2WalletGetter, l2WalletFundsValidator := validators.AcquireL2WalletWithFunds(
		chainIdx,
		sdktypes.NewBalance(big.NewInt(1.0*constants.ETH)),
	)
	l1WalletGetter, l1WalletFundsValidator := validators.AcquireL1WalletWithFunds(sdktypes.NewBalance(big.NewInt(1.0 * constants.ETH)))
	lowLevelSystemGetter, lowLevelSystemValidator := validators.AcquireLowLevelSystem()

	systest.SystemTest(t,
		erc20BridgeTestScenario(lowLevelSystemGetter, chainIdx, l1WalletGetter, l2WalletGetter),
		l2WalletFundsValidator,
		l1WalletFundsValidator,
		lowLevelSystemValidator,
	)
}

// erc20BridgeTestScenario tests depositing an ERC20 token from L1 to L2 through the bridge
func erc20BridgeTestScenario(lowLevelSystemGetter validators.LowLevelSystemGetter, chainIdx uint64, l1WalletGetter validators.WalletGetter, l2WalletGetter validators.WalletGetter) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()

		// Get the l2User wallet
		llsys := lowLevelSystemGetter(ctx)
		l1User := l1WalletGetter(ctx)
		l2User := l2WalletGetter(ctx)

		// Get the L1 chain
		l1Chain := llsys.L1()

		// Get the L2 chain
		l2Chain := llsys.L2s()[chainIdx]

		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started ERC20 bridge test")

		// Connect to L1 and L2
		l1Client, err := l1Chain.GethClient()
		require.NoError(t, err)
		t.Cleanup(func() { l1Client.Close() })

		l2Client, err := l2Chain.GethClient()
		require.NoError(t, err)
		t.Cleanup(func() { l2Client.Close() })

		// Print the L1 chain ID as a string
		logger.Info("L1 Chain ID", "id", l1Chain.ID())

		// Create transaction options for L1
		l1Opts, err := bind.NewKeyedTransactorWithChainID(l1User.PrivateKey(), l1Chain.ID())
		require.NoError(t, err)

		// Create transaction options for L2
		l2Opts, err := bind.NewKeyedTransactorWithChainID(l2User.PrivateKey(), l2Chain.ID())
		require.NoError(t, err)

		// Deploy a test ERC20 token on L1 (WETH)
		logger.Info("Deploying WETH token on L1")
		l1TokenAddress, tx, l1Token, err := bindings.DeployWETH(l1Opts, l1Client)
		require.NoError(t, err)

		// Wait for the token deployment transaction to be confirmed
		_, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
		require.NoError(t, err, "Failed to deploy L1 token")
		logger.Info("Deployed L1 token", "address", l1TokenAddress)

		// Mint some tokens to the user (deposit ETH to get WETH)
		mintAmount := big.NewInt(params.Ether) // 1 ETH
		l1Opts.Value = mintAmount
		tx, err = l1Token.Deposit(l1Opts)
		require.NoError(t, err)
		_, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
		require.NoError(t, err, "Failed to mint L1 tokens")
		l1Opts.Value = nil

		l1Balance, err := l1Token.BalanceOf(&bind.CallOpts{}, l1User.Address())
		require.NoError(t, err)
		require.Equal(t, mintAmount, l1Balance, "User should have the minted tokens on L1")
		logger.Info("User has tokens on L1", "balance", l1Balance)

		// Create the corresponding L2 token using the OptimismMintableERC20Factory
		logger.Info("Creating L2 token via OptimismMintableERC20Factory")
		optimismMintableTokenFactory, err := bindings.NewOptimismMintableERC20Factory(predeploys.OptimismMintableERC20FactoryAddr, l2Client)
		require.NoError(t, err)

		// Create the L2 token
		l2TokenName := "L2 Test Token"
		l2TokenSymbol := "L2TEST"
		tx, err = optimismMintableTokenFactory.CreateOptimismMintableERC20(l2Opts, l1TokenAddress, l2TokenName, l2TokenSymbol)
		require.NoError(t, err)
		l2TokenReceipt, err := wait.ForReceiptOK(ctx, l2Client, tx.Hash())
		require.NoError(t, err, "Failed to create L2 token")

		// Extract the L2 token address from the event logs
		var l2TokenAddress common.Address
		for _, log := range l2TokenReceipt.Logs {
			createdEvent, err := optimismMintableTokenFactory.ParseOptimismMintableERC20Created(*log)
			if err == nil && createdEvent != nil {
				l2TokenAddress = createdEvent.LocalToken
				break
			}
		}
		require.NotEqual(t, common.Address{}, l2TokenAddress, "Failed to find L2 token address from events")
		logger.Info("Created L2 token", "address", l2TokenAddress)

		// Get the L2 token contract
		l2Token, err := bindings.NewOptimismMintableERC20(l2TokenAddress, l2Client)
		require.NoError(t, err)

		// Check initial L2 token balance (should be 0)
		initialL2Balance, err := l2Token.BalanceOf(&bind.CallOpts{}, l2User.Address())
		require.NoError(t, err)
		require.True(t, big.NewInt(0).Cmp(initialL2Balance) == 0, "Initial L2 token balance should be 0, actual was %s", initialL2Balance.String())

		l1StandardBridgeAddress, ok := l2Chain.Addresses()["l1StandardBridgeProxy"]
		require.True(t, ok, fmt.Errorf("no L1 proxy address configured for this test"))

		l1StandardBridge, err := bindings.NewL1StandardBridge(l1StandardBridgeAddress, l1Client)
		require.NoError(t, err)

		// Approve the L1 bridge to spend tokens
		logger.Info("Approving L1 bridge to spend tokens")
		tx, err = l1Token.Approve(l1Opts, l1StandardBridgeAddress, mintAmount)
		require.NoError(t, err)
		_, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
		require.NoError(t, err, "Failed to approve L1 bridge")

		// Amount to bridge
		bridgeAmount := big.NewInt(params.Ether / 10) // 0.1 token
		minGasLimit := uint32(200000)                 // Minimum gas limit for the L2 transaction

		// Bridge the tokens from L1 to L2
		logger.Info("Bridging tokens from L1 to L2", "amount", bridgeAmount)
		tx, err = l1StandardBridge.DepositERC20To(
			l1Opts,
			l1TokenAddress,
			l2TokenAddress,
			l2User.Address(),
			bridgeAmount,
			minGasLimit,
			[]byte{}, // No extra data
		)
		require.NoError(t, err)
		depositReceipt, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
		require.NoError(t, err, "Failed to deposit tokens to L2")
		logger.Info("Deposit transaction confirmed on L1", "tx", tx.Hash().Hex())

		// Get the OptimismPortal contract to find the deposit event
		optimismPortal, err := bindings.NewOptimismPortal(l2Chain.Addresses()["optimismPortalProxy"], l1Client)
		require.NoError(t, err)

		// Find the TransactionDeposited event from the logs
		var depositFound bool
		for _, log := range depositReceipt.Logs {
			depositEvent, err := optimismPortal.ParseTransactionDeposited(*log)
			if err == nil && depositEvent != nil {
				logger.Info("Found deposit event", "from", depositEvent.From)
				depositFound = true
				break
			}
		}
		require.True(t, depositFound, "No deposit event found in transaction logs")

		// Wait for the deposit to be processed on L2
		// This may take some time as it depends on the L2 block time and the deposit processing
		logger.Info("Waiting for deposit to be processed on L2...")

		// Poll for the L2 balance to change
		err = wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
			l2Balance, err := l2Token.BalanceOf(&bind.CallOpts{}, l2User.Address())
			if err != nil {
				return false, err
			}
			return l2Balance.Cmp(initialL2Balance) > 0, nil
		})
		require.NoError(t, err, "Timed out waiting for L2 balance to change")

		// Verify the final L2 balance
		finalL2Balance, err := l2Token.BalanceOf(&bind.CallOpts{}, l2User.Address())
		require.NoError(t, err)
		require.True(t, bridgeAmount.Cmp(finalL2Balance) == 0, "L2 balance should match the bridged amount, L2 balance=%s, bridged amount=%s", finalL2Balance, bridgeAmount)
		logger.Info("Successfully verified tokens on L2", "balance", finalL2Balance)

		logger.Info("ERC20 bridge test completed successfully!")
	}
}
