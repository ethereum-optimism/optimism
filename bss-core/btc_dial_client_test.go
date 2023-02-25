package bsscore_test

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/bss-core/dial"
	"github.com/stretchr/testify/require"
)

// TestValidateConfig asserts the behavior of ValidateConfig by testing expected
// error and success configurations.
func TestDialBtcClientConfig(t *testing.T) {

	t.Run("btc client test", func(t *testing.T) {

		btcClient, err := dial.BTCClientWithTimeout("regtest.dctrl.wtf/", true)

		// Get the current block count.
		blockCount, err := btcClient.GetBlockCount()
		if err != nil {
			panic("oh no")
		}
		fmt.Printf("Block count: %d", blockCount)

		require.IsType(uint64, blockCount)
	})

}
