package jovian

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestConfigurableMinBaseFee verifies that the configurable minimum base fee feature works correctly
func TestConfigurableMinBaseFee(t *testing.T) {
	// Define which L2 chain we'll test
	chainIdx := uint64(0)

	// Get validators and getters for accessing the system and wallets
	walletGetter, walletValidator := validators.AcquireL2WalletWithFunds(chainIdx, types.NewBalance(big.NewInt(params.Ether)))

	// Run jovian test - require Jovian fork to be active
	_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Jovian)
	
	systest.SystemTest(t,
		configurableMinBaseFeeTestScenario(walletGetter, chainIdx),
		walletValidator,
		forkValidator,
	)
}

// configurableMinBaseFeeTestScenario creates a test scenario for verifying configurable minimum base fee
func configurableMinBaseFeeTestScenario(
	walletGetter validators.WalletGetter,
	chainIdx uint64,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()

		// Get the L2 client
		l2Chain := sys.L2s()[chainIdx]
		l2Client, err := l2Chain.Nodes()[0].GethClient()
		require.NoError(t, err)

		// Get the genesis config to ensure Jovian is active
		chainConfig, err := l2Chain.Config()
		require.NoError(t, err)
		require.NotNil(t, chainConfig.JovianTime, "Jovian fork must be configured")

		// Wait for at least block 1 to avoid genesis block edge cases
		header, err := l2Client.HeaderByNumber(ctx, big.NewInt(1))
		require.NoError(t, err)

		// Verify that we're on Jovian fork
		require.True(t, chainConfig.IsJovian(header.Time), "Chain must be running on Jovian fork for this test")

		// For Jovian, the minimum base fee is encoded in the extra data (last byte)
		// Extract minBaseFeeLog2 from block extra data
		require.Len(t, header.Extra, 10, "Jovian blocks should have 10 bytes of extra data")
		minBaseFeeLog2 := uint8(header.Extra[9])
		
		// Convert log2 value to actual minimum base fee (2^minBaseFeeLog2)
		minBaseFee := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(minBaseFeeLog2)), nil)
		require.Greater(t, minBaseFee.Uint64(), uint64(0), "Minimum base fee should be greater than 0")

		// Verify the minimum base fee is properly enforced
		// The base fee should never go below the minimum
		currentBaseFee := header.BaseFee
		require.True(t, currentBaseFee.Cmp(minBaseFee) >= 0, 
			"Current base fee (%s) should be >= minimum base fee (%s)", 
			currentBaseFee.String(), minBaseFee.String())

		// Wait for a few more blocks and verify base fee constraint is maintained
		for i := 0; i < 5; i++ {
			// Wait for next block
			nextBlockNum := new(big.Int).Add(header.Number, big.NewInt(1))
			
			// Poll for the next block (simple polling)
			var nextHeader *gethTypes.Header
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

			// Verify base fee constraint
			require.True(t, nextHeader.BaseFee.Cmp(minBaseFee) >= 0,
				"Block %d base fee (%s) should be >= minimum base fee (%s)",
				nextHeader.Number.Uint64(), nextHeader.BaseFee.String(), minBaseFee.String())

			header = nextHeader
		}

		// Test that the minBaseFee value is encoded in the block extra data
		// Jovian blocks should have 10 bytes of extra data (9 from Holocene + 1 for minBaseFee)
		require.Len(t, header.Extra, 10, "Jovian blocks should have 10 bytes of extra data")
		
		// The last byte should encode the minBaseFee parameter
		minBaseFeeExtraData := header.Extra[9]
		require.NotEqual(t, byte(0), minBaseFeeExtraData, "MinBaseFee extra data should be non-zero")

		t.Logf("Successfully verified configurable minimum base fee feature:")
		t.Logf("  - Minimum base fee: %s", minBaseFee.String())
		t.Logf("  - Current base fee: %s", header.BaseFee.String())
		t.Logf("  - Extra data length: %d bytes", len(header.Extra))
		t.Logf("  - MinBaseFee extra data: 0x%02x", minBaseFeeExtraData)
	}
}