package params_test

import (
	"encoding/json"
	"math/big"
	"testing"

	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// timestampForks lists the timestamp-based forks the ChainConfig carries a field
// for, paired with the field each one maps to. It drives the round-trip test that
// keeps ActivationTime and SetActivationTime in agreement.
var timestampForks = []struct {
	fork  forks.Name
	field func(*opparams.ChainConfig) *uint64
}{
	{forks.Regolith, func(c *opparams.ChainConfig) *uint64 { return c.RegolithTime }},
	{forks.Canyon, func(c *opparams.ChainConfig) *uint64 { return c.CanyonTime }},
	{forks.Ecotone, func(c *opparams.ChainConfig) *uint64 { return c.EcotoneTime }},
	{forks.Fjord, func(c *opparams.ChainConfig) *uint64 { return c.FjordTime }},
	{forks.Granite, func(c *opparams.ChainConfig) *uint64 { return c.GraniteTime }},
	{forks.Holocene, func(c *opparams.ChainConfig) *uint64 { return c.HoloceneTime }},
	{forks.Isthmus, func(c *opparams.ChainConfig) *uint64 { return c.IsthmusTime }},
	{forks.Jovian, func(c *opparams.ChainConfig) *uint64 { return c.JovianTime }},
	{forks.Karst, func(c *opparams.ChainConfig) *uint64 { return c.KarstTime }},
	{forks.Lagoon, func(c *opparams.ChainConfig) *uint64 { return c.InteropTime }},
}

// TestActivationTimeSetRoundTrip asserts the ActivationTime and SetActivationTime
// switches map each fork to the same field. Setting a fork's time and reading it
// back must return the same value, and must not disturb any other fork's field.
func TestActivationTimeSetRoundTrip(t *testing.T) {
	for i, tf := range timestampForks {
		t.Run(string(tf.fork), func(t *testing.T) {
			cfg := &opparams.ChainConfig{}
			want := uint64(1000 + i)
			cfg.SetActivationTime(tf.fork, &want)

			// SetActivationTime wrote the field ActivationTime reads.
			require.NotNil(t, tf.field(cfg))
			require.Equal(t, want, *tf.field(cfg))
			require.NotNil(t, cfg.ActivationTime(tf.fork))
			require.Equal(t, want, *cfg.ActivationTime(tf.fork))

			// No other fork's field was touched.
			for _, other := range timestampForks {
				if other.fork == tf.fork {
					continue
				}
				require.Nilf(t, other.field(cfg), "%s field set by SetActivationTime(%s)", other.fork, tf.fork)
			}
		})
	}
}

func TestActivationTimeUnsupportedForkPanics(t *testing.T) {
	cfg := &opparams.ChainConfig{}
	require.Panics(t, func() { cfg.ActivationTime(forks.Bedrock) })
	require.Panics(t, func() { cfg.ActivationTime(forks.Delta) })
	require.Panics(t, func() { cfg.SetActivationTime(forks.Delta, ptr.New(uint64(1))) })
}

func TestForkPredicates(t *testing.T) {
	// Every timestamp predicate is active at or after its activation time, and
	// inactive before it or when unscheduled.
	predicates := map[forks.Name]func(*opparams.ChainConfig, uint64) bool{
		forks.Regolith: (*opparams.ChainConfig).IsRegolith,
		forks.Canyon:   (*opparams.ChainConfig).IsCanyon,
		forks.Ecotone:  (*opparams.ChainConfig).IsEcotone,
		forks.Fjord:    (*opparams.ChainConfig).IsFjord,
		forks.Granite:  (*opparams.ChainConfig).IsGranite,
		forks.Holocene: (*opparams.ChainConfig).IsHolocene,
		forks.Isthmus:  (*opparams.ChainConfig).IsIsthmus,
		forks.Jovian:   (*opparams.ChainConfig).IsJovian,
		forks.Karst:    (*opparams.ChainConfig).IsKarst,
		forks.Lagoon:   (*opparams.ChainConfig).IsInterop,
	}
	for fork, pred := range predicates {
		t.Run(string(fork), func(t *testing.T) {
			cfg := &opparams.ChainConfig{}
			cfg.SetActivationTime(fork, ptr.New(uint64(100)))
			require.False(t, pred(cfg, 99), "inactive before activation")
			require.True(t, pred(cfg, 100), "active at activation")
			require.True(t, pred(cfg, 101), "active after activation")

			unscheduled := &opparams.ChainConfig{}
			require.False(t, pred(unscheduled, 100), "inactive when unscheduled")
		})
	}
}

func TestIsBedrock(t *testing.T) {
	cfg := &opparams.ChainConfig{BedrockBlock: big.NewInt(5)}
	require.False(t, cfg.IsBedrock(big.NewInt(4)))
	require.True(t, cfg.IsBedrock(big.NewInt(5)))
	require.True(t, cfg.IsBedrock(big.NewInt(6)))

	require.False(t, (&opparams.ChainConfig{}).IsBedrock(big.NewInt(5)), "nil BedrockBlock is never forked")
	require.False(t, cfg.IsBedrock(nil), "nil head is never forked")
}

// TestOptimismConfigJSONWireCompat asserts the OP-core OptimismConfig serialises
// byte-for-byte identically to op-geth's, and round-trips through op-geth's type.
// This guards the wire format of rollup.Config.ChainOpConfig.
func TestOptimismConfigJSONWireCompat(t *testing.T) {
	cases := map[string]*uint64{
		"with canyon denominator":    ptr.New(uint64(250)),
		"without canyon denominator": nil,
	}
	for name, canyon := range cases {
		t.Run(name, func(t *testing.T) {
			op := &opparams.OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50, EIP1559DenominatorCanyon: canyon}
			geth := &gethparams.OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50, EIP1559DenominatorCanyon: canyon}

			opJSON, err := json.Marshal(op)
			require.NoError(t, err)
			gethJSON, err := json.Marshal(geth)
			require.NoError(t, err)
			require.Equal(t, string(gethJSON), string(opJSON))

			var back gethparams.OptimismConfig
			require.NoError(t, json.Unmarshal(opJSON, &back))
			require.Equal(t, *geth, back)
		})
	}
}
