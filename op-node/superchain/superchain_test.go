package superchain

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	registry "github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/stretchr/testify/require"
)

// fullyPopulatedChainConfig returns a chain config with every optional field set to a non-zero
// value, so the registry→Config conversion has a source for every Config field. Extend it whenever
// ChainConfig gains a field, so TestRollupConfigFromRegistry_AllFieldsSet keeps covering the whole
// conversion.
func fullyPopulatedChainConfig() *registry.ChainConfig {
	portal := common.HexToAddress("0x1111111111111111111111111111111111111111")
	sysCfgProxy := common.HexToAddress("0x2222222222222222222222222222222222222222")
	daChallenge := common.HexToAddress("0x3333333333333333333333333333333333333333")

	return &registry.ChainConfig{
		ChainID:           10,
		BatchInboxAddr:    common.HexToAddress("0xff00000000000000000000000000000000000010"),
		BlockTime:         2,
		SeqWindowSize:     3600,
		MaxSequencerDrift: 600,
		Optimism: &registry.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New(uint64(250)),
		},
		AltDA: &registry.AltDAConfig{
			DaChallengeContractAddress: daChallenge,
			DaChallengeWindow:          160,
			DaResolveWindow:            160,
			DaCommitmentType:           "KeccakCommitment",
		},
		Genesis: registry.GenesisConfig{
			L2Time: 1234,
			L1:     registry.GenesisRef{Hash: common.HexToHash("0xaa"), Number: 100},
			L2:     registry.GenesisRef{Hash: common.HexToHash("0xbb")},
			SystemConfig: registry.SystemConfig{
				BatcherAddr: common.HexToAddress("0x4444444444444444444444444444444444444444"),
				Overhead:    common.HexToHash("0xbc"),
				Scalar:      common.HexToHash("0xa6fe0"),
				GasLimit:    30_000_000,
			},
		},
		Addresses: registry.AddressesConfig{
			OptimismPortalProxy: &portal,
			SystemConfigProxy:   &sysCfgProxy,
		},
		Hardforks: registry.HardforkConfig{
			CanyonTime:             ptr.New(uint64(1)),
			DeltaTime:              ptr.New(uint64(2)),
			EcotoneTime:            ptr.New(uint64(3)),
			FjordTime:              ptr.New(uint64(4)),
			GraniteTime:            ptr.New(uint64(5)),
			HoloceneTime:           ptr.New(uint64(6)),
			IsthmusTime:            ptr.New(uint64(7)),
			JovianTime:             ptr.New(uint64(8)),
			KarstTime:              ptr.New(uint64(9)),
			KeepKarstUpgradeGas:    true,
			LagoonTime:             ptr.New(uint64(10)),
			PectraBlobScheduleTime: ptr.New(uint64(11)),
		},
	}
}

// TestRollupConfigFromRegistry exercises the registry→rollup.Config conversion in isolation and
// checks that key registry fields — including the keep_karst_upgrade_gas behavioral flag — reach
// the rollup Config.
func TestRollupConfigFromRegistry(t *testing.T) {
	chConfig := fullyPopulatedChainConfig()
	cfg := rollupConfigFromRegistry(chConfig, registry.Superchain{L1: registry.L1Config{ChainID: 1}})

	require.Equal(t, big.NewInt(10), cfg.L2ChainID)
	require.Equal(t, big.NewInt(1), cfg.L1ChainID)
	require.Equal(t, uint64(2), cfg.BlockTime)
	require.Equal(t, *chConfig.Addresses.OptimismPortalProxy, cfg.DepositContractAddress)
	require.Equal(t, *chConfig.Addresses.SystemConfigProxy, cfg.L1SystemConfigAddress)
	require.Equal(t, uint64(9), *cfg.KarstTime)
	require.True(t, cfg.KeepKarstUpgradeGas,
		"keep_karst_upgrade_gas from the registry must be carried into rollup.Config")
}

// TestRollupConfigFromRegistry_AllFieldsSet feeds a fully-populated chain config through the
// conversion and asserts every rollup Config field ends up non-zero. A newly added registry-derived
// field that the conversion forgets to map stays at its zero value and fails here — the completeness
// guard for the whole Config, not just the hardforks.
func TestRollupConfigFromRegistry_AllFieldsSet(t *testing.T) {
	cfg := rollupConfigFromRegistry(fullyPopulatedChainConfig(), registry.Superchain{L1: registry.L1Config{ChainID: 1}})

	// The multi-blocks feature has no registry representation: it is configured per deployment, so
	// the conversion has nothing to map it from.
	notFromRegistry := map[string]bool{
		"MultiBlockTime": true,
		"MaxMultiBlocks": true,
	}

	v := reflect.ValueOf(*cfg)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if notFromRegistry[typ.Field(i).Name] {
			continue
		}
		require.Falsef(t, v.Field(i).IsZero(),
			"rollup Config field %q is zero after conversion: map it in rollupConfigFromRegistry, and "+
				"make sure fullyPopulatedChainConfig provides its source", typ.Field(i).Name)
	}
}

func TestApplyHardforks_NoForks(t *testing.T) {
	cfg := rollup.Config{}
	hardforks := registry.HardforkConfig{}
	applyHardforks(&cfg, hardforks)
	requireAllHardforksSetCorrectly(t, cfg, hardforks)
}

func TestApplyHardforks(t *testing.T) {
	cfg := rollup.Config{}
	hardforkCfg := registry.HardforkConfig{}

	// Set all hardforks
	hardforkVal := reflect.ValueOf(&hardforkCfg).Elem()
	for i := 0; i < hardforkVal.NumField(); i++ {
		field := hardforkVal.Field(i)
		switch field.Kind() {
		case reflect.Ptr: // *uint64 fork-activation times
			val := uint64(i + 10) // +10 just so they're all arbitrary non-zero values
			field.Set(reflect.ValueOf(&val))
		case reflect.Bool: // behavioral flags, e.g. KeepKarstUpgradeGas
			field.SetBool(true)
		default:
			t.Fatalf("unexpected hard fork field kind %v for %v", field.Kind(), hardforkVal.Type().Field(i).Name)
		}
	}

	applyHardforks(&cfg, hardforkCfg)

	requireAllHardforksSetCorrectly(t, cfg, hardforkCfg)
}

func requireAllHardforksSetCorrectly(t *testing.T, cfg rollup.Config, hardforkCfg registry.HardforkConfig) {
	hardforkType := reflect.TypeOf(hardforkCfg)
	hardforkVal := reflect.ValueOf(hardforkCfg)
	cfgVal := reflect.ValueOf(&cfg).Elem()
	for i := 0; i < hardforkVal.NumField(); i++ {
		hardforkField := hardforkType.Field(i)
		cfgField := cfgVal.FieldByName(hardforkField.Name)
		if hardforkField.Type.Kind() == reflect.Ptr {
			require.Equalf(t, hardforkVal.Field(i).Elem(), cfgField.Elem(), "missing hard fork field %v", hardforkField.Name)
		} else {
			require.Equalf(t, hardforkVal.Field(i).Interface(), cfgField.Interface(), "missing hard fork field %v", hardforkField.Name)
		}
	}
	// Regolith is always activated at genesis
	require.NotNil(t, cfg.RegolithTime)
	require.Zero(t, *cfg.RegolithTime)
}
