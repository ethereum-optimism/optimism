package rollup

import (
	"encoding/json"
	"os"
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

// On a multi-block chain a timestamp no longer identifies one block, so callers must be pushed to
// read the chain instead of computing a block number that would be wrong past the activation.
func TestConfig_TargetBlockNumberRejectsMultiBlock(t *testing.T) {
	cfg := multiBlockConfig()
	activation := *cfg.MultiBlockTime

	for _, ts := range []uint64{cfg.Genesis.L2Time, activation - cfg.BlockTime, activation, activation + 100*cfg.BlockTime} {
		_, err := cfg.TargetBlockNumber(ts)
		require.ErrorIsf(t, err, ErrMultiBlockNoBlockNumberForTimestamp, "timestamp %d", ts)
	}

	// without the activation the mapping is a bijection again
	cfg.MultiBlockTime = nil
	num, err := cfg.TargetBlockNumber(activation)
	require.NoError(t, err)
	require.Equal(t, cfg.Genesis.L2.Number+10, num)
	require.Equal(t, activation, cfg.TimestampForBlock(num))
}

// An activation time of 0 means active at genesis, the convention every scheduleable fork uses, so
// a deploy config with a zero offset produces a config that validates.
func TestConfig_MultiBlockAtGenesis(t *testing.T) {
	cfg := multiBlockConfig()
	cfg.MultiBlockTime = ptr.New(uint64(0))
	require.NoError(t, cfg.Check())

	require.True(t, cfg.IsMultiBlock(0))
	require.True(t, cfg.IsMultiBlock(cfg.Genesis.L2Time))

	// the activation timestamp itself never allows siblings
	require.False(t, cfg.SiblingsAllowed(0))
	require.True(t, cfg.SiblingsAllowed(cfg.Genesis.L2Time))
}

func TestConfig_MaxMultiBlocksUpperBound(t *testing.T) {
	cfg := multiBlockConfig()
	cfg.MaxMultiBlocks = ptr.New(uint64(MaxMultiBlocksLimit))
	require.NoError(t, cfg.Check())

	cfg.MaxMultiBlocks = ptr.New(uint64(MaxMultiBlocksLimit + 1))
	require.ErrorIs(t, cfg.Check(), ErrMaxMultiBlocksTooLarge)
}

// multiBlockRollupFixture is the rollup config op-chain-ops/genesis emits for a multi-blocks chain.
// kona parses the same bytes and makes the same assertions, so the two clients cannot drift on
// which timestamps allow siblings. Keep the file unchanged.
const multiBlockRollupFixture = "../../op-chain-ops/genesis/testdata/rollup-multiblock.json"

// TestConfig_SiblingsAllowedFixture pins the sibling exclusion set on the shared fixture: no fork
// activation timestamp scheduled past the multi-blocks activation may carry siblings, because the
// predicates that recognize an activation block do so from its timestamp alone.
func TestConfig_SiblingsAllowedFixture(t *testing.T) {
	data, err := os.ReadFile(multiBlockRollupFixture)
	require.NoError(t, err)
	var cfg Config
	require.NoError(t, json.Unmarshal(data, &cfg))
	require.NoError(t, cfg.Check())
	require.NotNil(t, cfg.MultiBlockTime)

	excluded := 0
	for _, fork := range scheduleableForks {
		activation := cfg.ActivationTime(fork)
		if activation == nil || *activation <= *cfg.MultiBlockTime {
			continue
		}
		excluded++
		require.Falsef(t, cfg.SiblingsAllowed(*activation), "siblings at the %s activation (%d)", fork, *activation)
		require.Truef(t, cfg.SiblingsAllowed(*activation-cfg.BlockTime), "no siblings one block before the %s activation", fork)
		require.Truef(t, cfg.SiblingsAllowed(*activation+cfg.BlockTime), "no siblings one block after the %s activation", fork)
	}
	require.Positive(t, excluded, "the fixture must schedule a fork past the multi-blocks activation")
}
