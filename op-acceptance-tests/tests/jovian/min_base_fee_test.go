package jovian

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestConfigurableMinBaseFee verifies that the configurable minimum base fee feature works correctly
func TestConfigurableMinBaseFee(t *testing.T) {
	t.Run("JovianOff", func(t *testing.T) {
		testConfigurableMinBaseFeeJovianOff(t)
	})

	t.Run("JovianOn", func(t *testing.T) {
		testConfigurableMinBaseFeeJovianOn(t)
	})
}

// testConfigurableMinBaseFeeJovianOff tests that when Jovian is off, base fees can drop normally
func testConfigurableMinBaseFeeJovianOff(t *testing.T) {
	runMinBaseFeeTest(t, false, configurableMinBaseFeeJovianOffTestScenario)
}

// testConfigurableMinBaseFeeJovianOn tests the minBaseFee feature when Jovian is enabled
func testConfigurableMinBaseFeeJovianOn(t *testing.T) {
	runMinBaseFeeTest(t, true, configurableMinBaseFeeTestScenario)
}

// runMinBaseFeeTest runs a min base fee test with common setup
func runMinBaseFeeTest(t *testing.T, jovianEnabled bool, scenario func(validators.WalletGetter, uint64) systest.SystemTestFunc) {
	// Define which L2 chain we'll test
	chainIdx := uint64(0)

	// Get validators and getters for accessing the system and wallets
	walletGetter, walletValidator := validators.AcquireL2WalletWithFunds(chainIdx, types.NewBalance(big.NewInt(params.Ether)))

	// Configure fork validator based on Jovian requirement
	if jovianEnabled {
		_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Jovian)
		systest.SystemTest(t,
			scenario(walletGetter, chainIdx),
			walletValidator,
			forkValidator,
		)
	} else {
		_, forkValidator := validators.AcquireL2WithoutFork(chainIdx, rollup.Jovian)
		systest.SystemTest(t,
			scenario(walletGetter, chainIdx),
			walletValidator,
			forkValidator,
		)
	}
}

// configurableMinBaseFeeJovianOffTestScenario creates a test scenario for when Jovian is off
func configurableMinBaseFeeJovianOffTestScenario(
	walletGetter validators.WalletGetter,
	chainIdx uint64,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()
		l2Client, chainConfig := getL2ClientAndConfig(t, sys, chainIdx)

		// Wait for at least block 1 to avoid genesis block edge cases
		initialHeader, err := l2Client.HeaderByNumber(ctx, big.NewInt(1))
		require.NoError(t, err)

		// Verify that we're NOT on Jovian fork
		if chainConfig.JovianTime != nil {
			require.False(t, chainConfig.IsJovian(initialHeader.Time), "Chain must NOT be running on Jovian fork for this test")
		}

		// Verify no Jovian extra data encoding (should not be 10 bytes)
		require.NotEqual(t, 10, len(initialHeader.Extra), "Should not have Jovian extra data encoding when Jovian is off")

		// Collect headers from multiple blocks
		headers := collectBlockHeaders(t, ctx, l2Client, initialHeader, 5)

		// With low gas usage and no minimum enforced, base fees should decrease
		foundDecrease := checkForBaseFeeDecrease(t, headers)
		require.True(t, foundDecrease, "Expected base fee to decrease when Jovian is off and no minimum is enforced")

		// Verify no Jovian extra data encoding in any block
		for _, h := range headers {
			require.NotEqual(t, 10, len(h.Extra), "Should not have Jovian extra data encoding when Jovian is off")
		}

		t.Logf("Successfully verified that base fees can decrease when Jovian is off:")
		t.Logf("  - Initial base fee: %s", headers[0].BaseFee.String())
		t.Logf("  - Final base fee: %s", headers[len(headers)-1].BaseFee.String())
		t.Logf("  - Found decreasing base fee trend")
	}
}

// configurableMinBaseFeeTestScenario creates a test scenario for verifying configurable minimum base fee
func configurableMinBaseFeeTestScenario(
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

		// Verify initial state: minBaseFeeLog2 default value is zero
		require.Len(t, initialHeader.Extra, 10, "Jovian blocks should have 10 bytes of extra data")
		initialMinBaseFeeLog2 := uint8(initialHeader.Extra[9])
		require.Equal(t, uint8(0), initialMinBaseFeeLog2, "MinBaseFee should initially be zero")

		// Wait for more blocks to see if minBaseFeeLog2 gets configured to a non-zero value
		// In a real deployment, this would happen via SystemConfig updates
		headers := collectBlockHeaders(t, ctx, l2Client, initialHeader, 10)

		// Find the first block where minBaseFeeLog2 is set to non-zero
		var nonZeroHeader *gethTypes.Header
		var minBaseFeeLog2 uint8
		for _, h := range headers {
			if chainConfig.IsConfigurableMinBaseFee(h.Time) {
				log2Value := uint8(h.Extra[9])
				if log2Value > 0 {
					nonZeroHeader = h
					minBaseFeeLog2 = log2Value
					t.Logf("Found non-zero minBaseFeeLog2: %d at block %d", log2Value, h.Number.Uint64())
					break
				}
			}
		}

		// Convert log2 value to actual minimum base fee (2^minBaseFeeLog2)
		minBaseFee := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(minBaseFeeLog2)), nil)

		// Verify the minimum base fee constraint is enforced from this point forward
		remainingHeaders := collectBlockHeaders(t, ctx, l2Client, nonZeroHeader, 5)
		for _, h := range remainingHeaders {
			require.True(t, h.BaseFee.Cmp(minBaseFee) >= 0,
				"Block %d base fee (%s) should be >= minimum base fee (%s)",
				h.Number.Uint64(), h.BaseFee.String(), minBaseFee.String())
		}

		finalHeader := remainingHeaders[len(remainingHeaders)-1]
		t.Logf("Successfully verified configurable minimum base fee feature:")
		t.Logf("  - Initial minBaseFeeLog2: %d", initialMinBaseFeeLog2)
		t.Logf("  - Configured minBaseFeeLog2: %d", minBaseFeeLog2)
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
	var headers []*gethTypes.Header
	headers = append(headers, initialHeader)

	for i := 0; i < numBlocks; i++ {
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
