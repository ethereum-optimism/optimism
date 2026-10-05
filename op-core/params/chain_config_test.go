package params_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"slices"
	"testing"

	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// TestForks checks how every mainline fork in forks.All is represented on ChainConfig.
// Bedrock is block-based, with the BedrockBlock field and IsBedrock. Forks outside
// forks.AllEL are consensus-layer only and have no field. Every other fork is
// timestamp-based, with a <Fork>Time field, an Is<Fork> predicate and an
// ActivationTime/SetActivationTime case.
func TestForks(t *testing.T) {
	for _, fork := range forks.All {
		t.Run(string(fork), func(t *testing.T) {
			switch {
			case fork == forks.Bedrock:
				checkBedrock(t)
			case !slices.Contains(forks.AllEL, fork):
				_, ok := reflect.TypeFor[opparams.ChainConfig]().FieldByName(forkstest.Title(fork) + "Time")
				require.Falsef(t, ok, "%s has no execution-layer field", fork)
				requireUnsupported(t, fork)
			default:
				checkTimestampFork(t, fork)
			}
		})
	}
	t.Run("unknown", func(t *testing.T) { requireUnsupported(t, "unknown") })
}

func checkBedrock(t *testing.T) {
	requireField(t, "BedrockBlock", reflect.TypeFor[*big.Int](), "bedrockBlock,omitempty")
	requireUnsupported(t, forks.Bedrock)

	cfg := &opparams.ChainConfig{BedrockBlock: big.NewInt(5)}
	require.False(t, cfg.IsBedrock(big.NewInt(4)))
	require.True(t, cfg.IsBedrock(big.NewInt(5)))
	require.True(t, cfg.IsBedrock(big.NewInt(6)))
	require.False(t, cfg.IsBedrock(nil), "nil head is never forked")
	require.False(t, (&opparams.ChainConfig{}).IsBedrock(big.NewInt(5)), "nil BedrockBlock is never forked")
}

func checkTimestampFork(t *testing.T, fork forks.Name) {
	field := requireField(t, forkstest.Title(fork)+"Time", reflect.TypeFor[*uint64](), string(fork)+"Time,omitempty")
	isActive := requirePredicate(t, "Is"+forkstest.Title(fork))

	activation := uint64(100)
	cfg := &opparams.ChainConfig{}
	require.Nil(t, cfg.ActivationTime(fork))
	require.False(t, isActive(cfg, activation), "unscheduled")
	require.False(t, cfg.IsForkActive(fork, activation), "unscheduled")

	cfg.SetActivationTime(fork, &activation)
	var want opparams.ChainConfig
	reflect.ValueOf(&want).Elem().FieldByIndex(field.Index).Set(reflect.ValueOf(&activation))
	require.Equal(t, want, *cfg, "SetActivationTime must set %s and nothing else", field.Name)
	require.Same(t, &activation, cfg.ActivationTime(fork))

	for _, at := range []uint64{activation - 1, activation, activation + 1} {
		require.Equal(t, at >= activation, isActive(cfg, at), "Is%s(%d)", forkstest.Title(fork), at)
		require.Equal(t, at >= activation, cfg.IsForkActive(fork, at), "IsForkActive(%s, %d)", fork, at)
	}
}

func requireField(t *testing.T, name string, typ reflect.Type, jsonTag string) reflect.StructField {
	f, ok := reflect.TypeFor[opparams.ChainConfig]().FieldByName(name)
	require.Truef(t, ok, "ChainConfig lacks field %s", name)
	require.Equal(t, typ, f.Type, name)
	require.Equal(t, jsonTag, f.Tag.Get("json"), name)
	return f
}

func requirePredicate(t *testing.T, name string) func(*opparams.ChainConfig, uint64) bool {
	m, ok := reflect.TypeFor[*opparams.ChainConfig]().MethodByName(name)
	require.Truef(t, ok, "ChainConfig lacks method %s", name)
	pred, ok := m.Func.Interface().(func(*opparams.ChainConfig, uint64) bool)
	require.Truef(t, ok, "%s has type %s, want func(uint64) bool", name, m.Type)
	return pred
}

func requireUnsupported(t *testing.T, fork forks.Name) {
	cfg := &opparams.ChainConfig{}
	require.Panics(t, func() { cfg.ActivationTime(fork) })
	require.Panics(t, func() { cfg.SetActivationTime(fork, ptr.New(uint64(1))) })
}

// TestGethChainConfig is a direct unit test of ChainConfig.GethChainConfig: it
// asserts the OP→go-ethereum mapping — the Ethereum fork schedule derived from the
// OP schedule (Shanghai=Canyon, Cancun=Ecotone, Prague=Isthmus, Osaka=Karst), the
// OP fork timestamps carried through unchanged, and the OptimismConfig mapping.
// Distinct fork times are used so a mis-wired mapping can't pass.
func TestGethChainConfig(t *testing.T) {
	cfg := &opparams.ChainConfig{
		ChainID:      big.NewInt(1234),
		BedrockBlock: big.NewInt(7),
		Optimism:     &opparams.OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50, EIP1559DenominatorCanyon: ptr.New(uint64(250))},
	}
	for i, fork := range forkstest.TimestampELForks {
		requireField(t, forkstest.Title(fork)+"Time", reflect.TypeFor[*uint64](), string(fork)+"Time,omitempty")
		cfg.SetActivationTime(fork, ptr.New(uint64(10*(i+1))))
	}

	geth := cfg.GethChainConfig()

	require.Equal(t, cfg.ChainID, geth.ChainID)
	require.Equal(t, cfg.BedrockBlock, geth.BedrockBlock)

	// Ethereum fork schedule is derived from the OP schedule.
	require.Equal(t, cfg.CanyonTime, geth.ShanghaiTime, "Shanghai activates with Canyon")
	require.Equal(t, cfg.EcotoneTime, geth.CancunTime, "Cancun activates with Ecotone")
	require.Equal(t, cfg.IsthmusTime, geth.PragueTime, "Prague activates with Isthmus")
	require.Equal(t, cfg.KarstTime, geth.OsakaTime, "Osaka activates with Karst")

	gethFields := reflect.ValueOf(geth).Elem()
	for _, fork := range forkstest.TimestampELForks {
		name := forkstest.Title(fork) + "Time"
		f := gethFields.FieldByName(name)
		require.Truef(t, f.IsValid(), "go-ethereum ChainConfig lacks field %s", name)
		require.Equal(t, cfg.ActivationTime(fork), f.Interface(), name)
	}

	// OptimismConfig maps across.
	require.NotNil(t, geth.Optimism)
	require.Equal(t, cfg.Optimism.EIP1559Elasticity, geth.Optimism.EIP1559Elasticity)
	require.Equal(t, cfg.Optimism.EIP1559Denominator, geth.Optimism.EIP1559Denominator)
	require.Equal(t, cfg.Optimism.EIP1559DenominatorCanyon, geth.Optimism.EIP1559DenominatorCanyon)

	// Non-OP-Mainnet chains get no pre-Bedrock block overrides (base values stay 0).
	require.Equal(t, int64(0), geth.BerlinBlock.Int64())
	require.Equal(t, int64(0), geth.LondonBlock.Int64())

	t.Run("op-mainnet pre-bedrock overrides", func(t *testing.T) {
		opMainnet := &opparams.ChainConfig{ChainID: big.NewInt(opparams.OPMainnetChainID), BedrockBlock: big.NewInt(opparams.OPMainnetGenesisBlockNum)}
		geth := opMainnet.GethChainConfig()
		require.Equal(t, int64(3950000), geth.BerlinBlock.Int64())
		require.Equal(t, int64(opparams.OPMainnetGenesisBlockNum), geth.LondonBlock.Int64())
	})
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
