package registry

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-service/ptr"

	"github.com/stretchr/testify/require"
)

func TestDefaultHardforkSchedule(t *testing.T) {
	sched := DefaultHardforkSchedule()
	require.NotSame(t, sched, DefaultHardforkSchedule())
	require.NotNil(t, sched.RegolithTime(0))
	require.NotNil(t, sched.CanyonTime(0))
	require.NotNil(t, sched.DeltaTime(0))
	require.NotNil(t, sched.EcotoneTime(0))
	require.NotNil(t, sched.FjordTime(0))
	require.NotNil(t, sched.GraniteTime(0))
	require.NotNil(t, sched.HoloceneTime(0))
	require.NotNil(t, sched.IsthmusTime(0))
	require.NotNil(t, sched.JovianTime(0))
	require.NotNil(t, sched.KarstTime(0))
	// The default stops at maxDefaultHardfork even when OP Mainnet schedules Lagoon.
	require.Nil(t, sched.LagoonTime(0))
}

func TestDefaultHardforkFor(t *testing.T) {
	tests := []struct {
		name   string
		config superchain.HardforkConfig
		want   forks.Name
	}{
		{
			name: "fork up to the cap is kept",
			config: superchain.HardforkConfig{
				CanyonTime: ptr.New(uint64(0)),
				JovianTime: ptr.New(uint64(0)),
			},
			want: forks.Jovian,
		},
		{
			name: "fork after the cap is capped",
			config: superchain.HardforkConfig{
				KarstTime:  ptr.New(uint64(0)),
				LagoonTime: ptr.New(uint64(1)),
			},
			want: forks.Karst,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, defaultHardforkFor(tt.config))
		})
	}
}

func TestLatestScheduledMainlineFork(t *testing.T) {
	tests := []struct {
		name   string
		config superchain.HardforkConfig
		want   forks.Name
	}{
		{name: "regolith", want: forks.Regolith},
		{
			name: "delta",
			config: superchain.HardforkConfig{
				CanyonTime: ptr.New(uint64(0)),
				DeltaTime:  ptr.New(uint64(0)),
			},
			want: forks.Delta,
		},
		{
			name: "latest mainline fork wins",
			config: superchain.HardforkConfig{
				CanyonTime: ptr.New(uint64(0)),
				LagoonTime: ptr.New(uint64(0)),
			},
			want: forks.Lagoon,
		},
		{
			name: "optional fork is ignored",
			config: superchain.HardforkConfig{
				PectraBlobScheduleTime: ptr.New(uint64(0)),
			},
			want: forks.Regolith,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, latestScheduledMainlineFork(tt.config))
		})
	}
}
