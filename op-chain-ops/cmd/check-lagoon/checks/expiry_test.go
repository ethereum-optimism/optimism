package checks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestDefaultExpiryPeriodIsEightDays(t *testing.T) {
	require.Equal(t, uint64(8*24*60*60), DefaultExpiryPeriod)
}

func TestExpiryStateVerify(t *testing.T) {
	chain := eth.ChainIDFromUInt64(10)
	ok := expiryState{ChainID: chain, ExpiryPeriod: DefaultExpiryPeriod}
	synced := ok
	synced.L1Read, synced.L1ChainID, synced.L1Paused, synced.L2Paused = true, chain, true, true

	tests := []struct {
		name    string
		state   expiryState
		wantErr []string
	}{
		{name: "production period, L1 not compared", state: ok},
		{name: "production period, pause in sync", state: synced},
		{
			name:    "short period",
			state:   expiryState{ChainID: chain, ExpiryPeriod: 30},
			wantErr: []string{"expiryPeriod() is 30s, want 691200s"},
		},
		{
			name: "L2 unpaused while L1 paused",
			state: func() expiryState {
				s := synced
				s.L2Paused = false
				return s
			}(),
			wantErr: []string{"ETHLiquidity.paused() is false but the L1 pause is true"},
		},
		{
			name: "messenger of another chain",
			state: func() expiryState {
				s := synced
				s.L1ChainID = eth.ChainIDFromUInt64(130)
				return s
			}(),
			wantErr: []string{"belongs to chain 130, not 10"},
		},
		{
			name: "every mismatch is reported",
			state: func() expiryState {
				s := synced
				s.ExpiryPeriod = 1
				s.L2Paused = false
				return s
			}(),
			wantErr: []string{"expiryPeriod() is 1s", "ETHLiquidity.paused() is false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.verify(DefaultExpiryPeriod)
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			for _, want := range tt.wantErr {
				require.ErrorContains(t, err, want)
			}
		})
	}
}
