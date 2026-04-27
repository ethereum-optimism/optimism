package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestConfigCheckConcurrency(t *testing.T) {
	validConfig := func() *Config {
		chainID := eth.ChainIDFromUInt64(901)
		return &Config{
			L2RPCs:              []string{"http://127.0.0.1:8545"},
			RollupConfigs:       map[eth.ChainID]*rollup.Config{chainID: testRollupConfig(901, 0, 1000)},
			BackfillDuration:    time.Hour,
			MessageExpiryWindow: uint64(time.Hour.Seconds()),
			PollInterval:        time.Second,
			ValidationInterval:  time.Second,
			RPCConcurrency:      100,
			FetchConcurrency:    64,
		}
	}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, validConfig().Check())
	})

	t.Run("rpc concurrency must be positive", func(t *testing.T) {
		cfg := validConfig()
		cfg.RPCConcurrency = 0
		require.ErrorContains(t, cfg.Check(), "rpc-concurrency must be positive")
	})

	t.Run("fetch concurrency must be positive", func(t *testing.T) {
		cfg := validConfig()
		cfg.FetchConcurrency = 0
		require.ErrorContains(t, cfg.Check(), "fetch-concurrency must be positive")
	})

	t.Run("fetch concurrency cannot exceed rpc concurrency", func(t *testing.T) {
		cfg := validConfig()
		cfg.RPCConcurrency = 4
		cfg.FetchConcurrency = 5
		require.ErrorContains(t, cfg.Check(), "fetch-concurrency must be less than or equal to rpc-concurrency")
	})
}
