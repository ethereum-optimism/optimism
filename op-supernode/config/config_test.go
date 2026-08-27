package config

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	snflags "github.com/ethereum-optimism/optimism/op-supernode/flags"
)

func TestNewConfigSharedBeaconSlotDurationOverride(t *testing.T) {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range snflags.Flags {
		require.NoError(t, f.Apply(set))
	}
	require.NoError(t, set.Parse([]string{
		"--l1=http://l1",
		"--l1.beacon=http://beacon",
		"--l1.beacon.slot-duration-override=6",
	}))
	cfg := NewConfig(cli.NewContext(&cli.App{}, set, nil))
	require.Equal(t, uint64(6), cfg.L1BeaconSlotDurationOverride)
}

func TestCLIConfig_Check_interopLogBackfill(t *testing.T) {
	ptr := func(u uint64) *uint64 { return &u }
	tests := []struct {
		name    string
		cfg     *CLIConfig
		wantErr string
	}{
		{
			name: "ok with activation and depth",
			cfg:  &CLIConfig{L1NodeAddr: "http://x", InteropActivationTimestamp: ptr(1), InteropLogBackfillDepth: time.Hour},
		},
		{
			// No CLI activation here is fine at the Check() layer; the
			// rollup-derived path is a valid activation source, and the
			// pairing is re-checked in supernode.New after resolution.
			name: "depth without CLI activation is allowed at Check; resolved later",
			cfg:  &CLIConfig{L1NodeAddr: "http://x", InteropLogBackfillDepth: time.Hour},
		},
		{
			name:    "negative depth",
			cfg:     &CLIConfig{L1NodeAddr: "http://x", InteropActivationTimestamp: ptr(1), InteropLogBackfillDepth: -time.Second},
			wantErr: "interop.log-backfill-depth must be >= 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Check()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
