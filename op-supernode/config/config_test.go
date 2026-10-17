package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIConfig_Check_interopLogBackfill(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *CLIConfig
		wantErr string
	}{
		{
			name: "positive depth",
			cfg:  &CLIConfig{L1NodeAddr: "http://x", InteropLogBackfillDepth: time.Hour},
		},
		{
			name:    "negative depth",
			cfg:     &CLIConfig{L1NodeAddr: "http://x", InteropLogBackfillDepth: -time.Second},
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
