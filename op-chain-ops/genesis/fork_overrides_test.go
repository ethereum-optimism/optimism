package genesis

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
)

func TestForkOverridesAtGenesis(t *testing.T) {
	t.Log("NOTE: if this test fails, it is probably because there is a new hardfork that should be included in the expected result.")
	zero := hexutil.Uint64(0)

	tests := []struct {
		name string
		fork forks.Name
		want map[string]*hexutil.Uint64
	}{
		{
			name: "bedrock",
			fork: forks.Bedrock,
			want: map[string]*hexutil.Uint64{
				"l2GenesisRegolithTimeOffset": nil,
				"l2GenesisCanyonTimeOffset":   nil,
				"l2GenesisDeltaTimeOffset":    nil,
				"l2GenesisEcotoneTimeOffset":  nil,
				"l2GenesisFjordTimeOffset":    nil,
				"l2GenesisGraniteTimeOffset":  nil,
				"l2GenesisHoloceneTimeOffset": nil,
				"l2GenesisIsthmusTimeOffset":  nil,
				"l2GenesisJovianTimeOffset":   nil,
				"l2GenesisKarstTimeOffset":    nil,
				"l2GenesisLagoonTimeOffset":   nil,
			},
		},
		{
			name: "isthmus",
			fork: forks.Isthmus,
			want: map[string]*hexutil.Uint64{
				"l2GenesisRegolithTimeOffset": &zero,
				"l2GenesisCanyonTimeOffset":   &zero,
				"l2GenesisDeltaTimeOffset":    &zero,
				"l2GenesisEcotoneTimeOffset":  &zero,
				"l2GenesisFjordTimeOffset":    &zero,
				"l2GenesisGraniteTimeOffset":  &zero,
				"l2GenesisHoloceneTimeOffset": &zero,
				"l2GenesisIsthmusTimeOffset":  &zero,
				"l2GenesisJovianTimeOffset":   nil,
				"l2GenesisKarstTimeOffset":    nil,
				"l2GenesisLagoonTimeOffset":   nil,
			},
		},
		{
			name: "lagoon",
			fork: forks.Lagoon,
			want: map[string]*hexutil.Uint64{
				"l2GenesisRegolithTimeOffset": &zero,
				"l2GenesisCanyonTimeOffset":   &zero,
				"l2GenesisDeltaTimeOffset":    &zero,
				"l2GenesisEcotoneTimeOffset":  &zero,
				"l2GenesisFjordTimeOffset":    &zero,
				"l2GenesisGraniteTimeOffset":  &zero,
				"l2GenesisHoloceneTimeOffset": &zero,
				"l2GenesisIsthmusTimeOffset":  &zero,
				"l2GenesisJovianTimeOffset":   &zero,
				"l2GenesisKarstTimeOffset":    &zero,
				"l2GenesisLagoonTimeOffset":   &zero,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ForkOverridesAtGenesis(tt.fork))
		})
	}
}
