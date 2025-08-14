package jovian

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestConfigurableMinBaseFee verifies that the configurable minimum base fee feature works correctly
func TestConfigurableMinBaseFee(t *testing.T) {
	t.Run("MinBaseFeeOff", func(t *testing.T) {
		// when minBaseFee is 0, base fees can drop normally
		runMinBaseFeeTest(t, configurableMinBaseFeeOffTestScenario)
	})

	t.Run("MinBaseFeeOn", func(t *testing.T) {
		// when it's enabled with a non-zero value
		runMinBaseFeeTest(t, configurableMinBaseFeeOnTestScenario)
	})
}

// runMinBaseFeeTest runs a min base fee test with common setup
func runMinBaseFeeTest(t *testing.T, scenario func(validators.WalletGetter, uint64) systest.SystemTestFunc) {
	// Define which L2 chain we'll test
	chainIdx := uint64(0)

	// Get validators and getters for accessing the system and wallets
	walletGetter, walletValidator := validators.AcquireL2WalletWithFunds(chainIdx, types.NewBalance(big.NewInt(params.Ether)))

	_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Jovian)
	systest.SystemTest(t,
		scenario(walletGetter, chainIdx),
		walletValidator,
		forkValidator,
	)
}

// configurableMinBaseFeeOffTestScenario creates a test scenario for when minBaseFee is 0 (disabled)
func configurableMinBaseFeeOffTestScenario(
	walletGetter validators.WalletGetter,
	chainIdx uint64,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()
		l2Client, chainConfig := getL2ClientAndConfig(t, sys, chainIdx)

		// Wait for at least block 1 to avoid genesis block edge cases
		initialHeader, err := l2Client.HeaderByNumber(ctx, big.NewInt(1))
		require.NoError(t, err)

		// Verify that we're on Jovian fork
		require.True(t, chainConfig.IsJovian(initialHeader.Time), "Chain must be running on Jovian fork for this test")

		// Verify Jovian extra data encoding (should be 10 bytes)
		require.Len(t, initialHeader.Extra, 10, "Jovian blocks should have 10 bytes of extra data")

		// Ensure minBaseFeeFactors is 0 (disabled)
		initialMinBaseFeeFactors := uint8(initialHeader.Extra[9])
		if initialMinBaseFeeFactors != 0 {
			// Set it to 0 to test the "off" state
			setMinBaseFeeFactors(t, ctx, sys, chainIdx, 0)
			time.Sleep(3 * time.Second)
		}

		// Collect headers from multiple blocks
		headers := collectBlockHeaders(t, ctx, l2Client, initialHeader, 5)

		// With minBaseFee disabled (0), base fees should be able to decrease
		foundDecrease := checkForBaseFeeDecrease(t, headers)
		require.True(t, foundDecrease, "Expected base fee to decrease when minBaseFee is disabled")

		// Verify minBaseFeeFactors remains 0 in all blocks
		for _, h := range headers {
			require.Len(t, h.Extra, 10, "Jovian blocks should have 10 bytes of extra data")
			minBaseFeeFactors := uint8(h.Extra[9])
			require.Equal(t, uint8(0), minBaseFeeFactors, "MinBaseFeeFactors should be 0 when disabled")
		}

		t.Logf("Successfully verified that base fees can decrease when minBaseFee is disabled:")
		t.Logf("  - Initial base fee: %s", headers[0].BaseFee.String())
		t.Logf("  - Final base fee: %s", headers[len(headers)-1].BaseFee.String())
		t.Logf("  - Found decreasing base fee trend")
	}
}

// configurableMinBaseFeeOnTestScenario creates a test scenario for verifying configurable minimum base fee when enabled
func configurableMinBaseFeeOnTestScenario(
	walletGetter validators.WalletGetter,
	chainIdx uint64,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()
		l2Client, chainConfig := getL2ClientAndConfig(t, sys, chainIdx)
		require.NotNil(t, chainConfig.JovianTime, "Jovian fork must be configured")

		// Wait for at least block 1 to avoid genesis block edge cases
		initialHeader, err := l2Client.HeaderByNumber(ctx, big.NewInt(1))
		require.NoError(t, err)

		// Verify that we're on Jovian fork
		require.True(t, chainConfig.IsJovian(initialHeader.Time), "Chain must be running on Jovian fork for this test")

		// Verify initial state: minBaseFeeFactors default value is zero
		require.Len(t, initialHeader.Extra, 10, "Jovian blocks should have 10 bytes of extra data")
		initialMinBaseFeeFactors := uint8(initialHeader.Extra[9])
		require.Equal(t, uint8(0), initialMinBaseFeeFactors, "MinBaseFee should initially be zero")

		// Set minBaseFeeFactors via SystemConfig contract
		minBaseFeeFactors := eip1559.EncodeMinBaseFeeFactors(1, 9) // 1 * 10^9 = 1 gwei minimum base fee
		setMinBaseFeeFactors(t, ctx, sys, chainIdx, minBaseFeeFactors)

		// Wait a bit for the L2 to process the L1 change
		time.Sleep(3 * time.Second)

		// Get a header after the configuration
		configuredHeader, err := l2Client.HeaderByNumber(ctx, nil)
		require.NoError(t, err)

		// Convert significand and exponent to actual minimum base fee (significand * 10^exponent)
		significand, exponent := eip1559.DecodeMinBaseFeeFactors(minBaseFeeFactors)
		minBaseFee := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
		minBaseFee.Mul(minBaseFee, big.NewInt(int64(significand)))

		// Verify the minimum base fee constraint is enforced from this point forward
		remainingHeaders := collectBlockHeaders(t, ctx, l2Client, configuredHeader, 5)
		for _, h := range remainingHeaders {
			require.True(t, h.BaseFee.Cmp(minBaseFee) >= 0,
				"Block %d base fee (%s) should be >= minimum base fee (%s)",
				h.Number.Uint64(), h.BaseFee.String(), minBaseFee.String())
		}

		finalHeader := remainingHeaders[len(remainingHeaders)-1]
		t.Logf("Successfully verified configurable minimum base fee feature:")
		t.Logf("  - Initial minBaseFeeFactors: 0x%02x", initialMinBaseFeeFactors)
		t.Logf("  - Configured minBaseFeeFactors: 0x%02x", minBaseFeeFactors)
		t.Logf("  - Minimum base fee: %s", minBaseFee.String())
		t.Logf("  - Final base fee: %s", finalHeader.BaseFee.String())
	}
}

// Helper functions

// getL2ClientAndConfig returns the L2 client and chain config for the specified chain
func getL2ClientAndConfig(t systest.T, sys system.System, chainIdx uint64) (*ethclient.Client, *params.ChainConfig) {
	l2Chain := sys.L2s()[chainIdx]
	l2Client, err := l2Chain.Nodes()[0].GethClient()
	require.NoError(t, err)

	chainConfig, err := l2Chain.Config()
	require.NoError(t, err)

	return l2Client, chainConfig
}

// collectBlockHeaders collects headers from multiple consecutive blocks
func collectBlockHeaders(t systest.T, ctx context.Context, l2Client *ethclient.Client, initialHeader *gethTypes.Header, numBlocks int) []*gethTypes.Header {
	if initialHeader == nil {
		require.Fail(t, "initialHeader cannot be nil")
		return nil
	}

	var headers []*gethTypes.Header
	headers = append(headers, initialHeader)

	for i := 0; i < numBlocks; i++ {
		if len(headers) == 0 {
			require.Fail(t, "headers slice is empty, cannot continue")
			return nil
		}
		nextBlockNum := new(big.Int).Add(headers[len(headers)-1].Number, big.NewInt(1))

		// Poll for the next block (simple polling)
		var nextHeader *gethTypes.Header
		var err error
		for attempts := 0; attempts < 20; attempts++ { // Wait up to 20 seconds
			nextHeader, err = l2Client.HeaderByNumber(ctx, nextBlockNum)
			if err == nil {
				break
			}
			// Wait 1 second before trying again
			select {
			case <-ctx.Done():
				require.NoError(t, ctx.Err())
			case <-time.After(time.Second):
			}
		}
		require.NoError(t, err, "Should be able to get next block header")
		headers = append(headers, nextHeader)
	}

	return headers
}

// checkForBaseFeeDecrease checks if any base fee decreased between consecutive blocks
func checkForBaseFeeDecrease(t systest.T, headers []*gethTypes.Header) bool {
	for i := 1; i < len(headers); i++ {
		if headers[i].BaseFee.Cmp(headers[i-1].BaseFee) < 0 {
			t.Logf("Base fee decreased from %s to %s between blocks %d and %d",
				headers[i-1].BaseFee.String(), headers[i].BaseFee.String(),
				headers[i-1].Number.Uint64(), headers[i].Number.Uint64())
			return true
		}
	}
	return false
}

// setMinBaseFeeFactors configures the minimum base fee via SystemConfig contract
func setMinBaseFeeFactors(t systest.T, ctx context.Context, sys system.System, chainIdx uint64, minBaseFeeFactors uint8) {
	// Get L1 client
	l1Client, err := sys.L1().Nodes()[0].GethClient()
	require.NoError(t, err)

	// Get L2 chain for L1 addresses
	l2Chain := sys.L2s()[chainIdx]
	l1Addresses := l2Chain.L1Addresses()

	// Get SystemConfig proxy address
	systemConfigAddr, exists := l1Addresses["SystemConfigProxy"]
	require.True(t, exists, "SystemConfigProxy address must exist")

	// Get L1 wallet for transactions
	l1Wallets := l2Chain.L1Wallets()

	// Try different wallet names
	var wallet system.Wallet
	for name, w := range l1Wallets {
		wallet = w
		t.Logf("Found L1 wallet: %s", name)
		break
	}
	require.NotNil(t, wallet, "Must have at least one L1 wallet")

	// Get chain ID
	chainID, err := l1Client.ChainID(ctx)
	require.NoError(t, err)

	// Create transactor using the wallet's private key
	privKey := wallet.PrivateKey()
	// types.Key is actually *ecdsa.PrivateKey
	ecdsaKey := (*ecdsa.PrivateKey)(privKey)
	opts, err := bind.NewKeyedTransactorWithChainID(ecdsaKey, chainID)
	require.NoError(t, err)

	// Bind to SystemConfig contract
	sysconfig, err := bindings.NewSystemConfig(systemConfigAddr, l1Client)
	require.NoError(t, err)

	// Set the minBaseFeeFactors value
	significand, exponent := eip1559.DecodeMinBaseFeeFactors(minBaseFeeFactors)
	tx, err := sysconfig.SetMinBaseFee(opts, significand, exponent)
	require.NoError(t, err, "SetMinBaseFee transaction")

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, l1Client, tx)
	require.NoError(t, err, "waiting for SetMinBaseFee transaction")
	require.Equal(t, uint64(1), receipt.Status, "SetMinBaseFee transaction should succeed")

	t.Logf("Successfully set minBaseFee to %d * 10^%d in tx %s", significand, exponent, tx.Hash().Hex())
}
