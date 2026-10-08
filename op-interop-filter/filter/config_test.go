package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-interop-filter/flags"
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
	cfg.MessageExpiryWindow = depset.MessageExpiryTimeSecondsInterop
	require.NoError(t, cfg.Check())
	cfg.MessageExpiryWindow = depset.MessageExpiryTimeSecondsInterop + 1
	require.ErrorContains(t, cfg.Check(), "message-expiry-window 604801s exceeds protocol window 604800s")
}

func TestNewConfig_MessageExpiryWindowAboveProtocolWindow(t *testing.T) {
	app := cli.NewApp()
	app.Flags = flags.Flags
	var cfgErr error
	app.Action = func(ctx *cli.Context) error {
		_, cfgErr = NewConfig(ctx, "test")
		return nil
	}
	// The flag is checked before it is truncated to whole seconds, so half a second over 168h is
	// rejected too.
	require.NoError(t, app.Run([]string{"op-interop-filter", "--message-expiry-window=168h0m0.5s"}))
	require.ErrorContains(t, cfgErr, "message-expiry-window 168h0m0.5s exceeds protocol window 168h0m0s")
}
