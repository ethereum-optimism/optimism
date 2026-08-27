package rollup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// multiBlockConfig returns a valid config with the multi-blocks feature scheduled at
// genesis + 10 block times and a group limit of 4.
func multiBlockConfig() *Config {
	cfg := randConfig()
	cfg.KarstTime = ptr.New(uint64(0))
	cfg.MultiBlockTime = ptr.New(cfg.Genesis.L2Time + 10*cfg.BlockTime)
	cfg.MaxMultiBlocks = ptr.New(uint64(4))
	return cfg
}

func TestConfig_MultiBlockJSON(t *testing.T) {
	cfg := multiBlockConfig()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.Contains(t, string(data), `"multi_block_time"`)
	require.Contains(t, string(data), `"max_multi_blocks"`)

	var roundTripped Config
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	require.Equal(t, cfg, &roundTripped)
}

func TestConfig_MultiBlockJSONOmittedWhenUnset(t *testing.T) {
	data, err := json.Marshal(randConfig())
	require.NoError(t, err)
	require.NotContains(t, string(data), "multi_block_time")
	require.NotContains(t, string(data), "max_multi_blocks")
}

func TestConfig_MultiBlockActivationTime(t *testing.T) {
	var cfg Config
	ts := uint64(4242)
	cfg.SetActivationTime(forks.MultiBlock, &ts)
	require.Equal(t, &ts, cfg.MultiBlockTime)
	require.Equal(t, &ts, cfg.ActivationTime(forks.MultiBlock))
	require.False(t, cfg.IsMultiBlock(ts-1))
	require.True(t, cfg.IsMultiBlock(ts))
}

func TestConfig_MaxMultiBlocksOrDefault(t *testing.T) {
	var cfg Config
	require.Equal(t, uint64(1), cfg.MaxMultiBlocksOrDefault())
	cfg.MaxMultiBlocks = ptr.New(uint64(16))
	require.Equal(t, uint64(16), cfg.MaxMultiBlocksOrDefault())
}

func TestConfig_SiblingsAllowed(t *testing.T) {
	cfg := multiBlockConfig()
	activation := *cfg.MultiBlockTime
	bt := cfg.BlockTime
	// A later fork lands inside the multi-block era: siblings must not fall on its activation.
	cfg.LagoonTime = ptr.New(activation + 5*bt)

	require.False(t, cfg.SiblingsAllowed(activation-bt), "before activation")
	require.False(t, cfg.SiblingsAllowed(activation), "at the activation timestamp")
	require.True(t, cfg.SiblingsAllowed(activation+bt))
	require.False(t, cfg.SiblingsAllowed(*cfg.LagoonTime), "at a later fork's activation timestamp")
	require.True(t, cfg.SiblingsAllowed(*cfg.LagoonTime+bt))

	var unset Config
	require.False(t, unset.SiblingsAllowed(1000))
}

func TestConfig_CheckMultiBlock(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		expErr error
	}{
		{name: "valid", mutate: func(*Config) {}},
		{
			name:   "explicit zero max",
			mutate: func(c *Config) { c.MaxMultiBlocks = ptr.New(uint64(0)) },
			expErr: ErrInvalidMaxMultiBlocks,
		},
		{
			name:   "before Karst",
			mutate: func(c *Config) { c.KarstTime = ptr.New(*c.MultiBlockTime + c.BlockTime) },
			expErr: ErrMultiBlockBeforeKarst,
		},
		{
			name:   "Karst unset",
			mutate: func(c *Config) { c.KarstTime = nil },
			expErr: ErrMultiBlockBeforeKarst,
		},
		{
			name:   "misaligned with block time",
			mutate: func(c *Config) { c.MultiBlockTime = ptr.New(*c.MultiBlockTime + 1) },
			expErr: ErrMultiBlockMisaligned,
		},
		{
			name:   "before genesis",
			mutate: func(c *Config) { c.MultiBlockTime = ptr.New(c.Genesis.L2Time - c.BlockTime) },
			expErr: ErrMultiBlockMisaligned,
		},
		{
			name:   "max without activation is allowed",
			mutate: func(c *Config) { c.MultiBlockTime = nil },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := multiBlockConfig()
			test.mutate(cfg)
			err := cfg.Check()
			if test.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, test.expErr)
			}
		})
	}
}
