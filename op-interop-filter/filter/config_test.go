package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestConfigCheck_AssumeValidBefore(t *testing.T) {
	const backfill = 168 * time.Hour
	tests := []struct {
		name              string
		assumeValidBefore time.Duration
		wantErr           string
	}{
		{name: "negative", assumeValidBefore: -time.Minute, wantErr: "assume-valid-before must not be negative"},
		{name: "zero", assumeValidBefore: 0},
		{name: "equal to backfill-duration", assumeValidBefore: backfill, wantErr: "assume-valid-before must be less than backfill-duration"},
		{name: "less than backfill-duration", assumeValidBefore: 30 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				L2RPCs:              []string{"http://localhost:8545"},
				RollupConfigs:       map[eth.ChainID]*rollup.Config{eth.ChainIDFromUInt64(901): {}},
				BackfillDuration:    backfill,
				AssumeValidBefore:   tt.assumeValidBefore,
				MessageExpiryWindow: uint64(DefaultMessageExpiryWindow.Seconds()),
				PollInterval:        time.Second,
				ValidationInterval:  time.Second,
				RPCConcurrency:      1,
				FetchConcurrency:    1,
			}
			err := cfg.Check()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestConfigCheck_MessageExpiryWindow(t *testing.T) {
	cfg := &Config{
		L2RPCs:             []string{"http://localhost:8545"},
		RollupConfigs:      map[eth.ChainID]*rollup.Config{eth.ChainIDFromUInt64(901): {}},
		BackfillDuration:   168 * time.Hour,
		PollInterval:       time.Second,
		ValidationInterval: time.Second,
		RPCConcurrency:     1,
		FetchConcurrency:   1,
	}
	cfg.MessageExpiryWindow = uint64(DefaultMessageExpiryWindow.Seconds())
	require.NoError(t, cfg.Check())
	cfg.MessageExpiryWindow = uint64(DefaultMessageExpiryWindow.Seconds()) + 1
	require.ErrorContains(t, cfg.Check(), "message-expiry-window must not exceed 7 days")
}
