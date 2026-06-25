package rollup

import (
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/superchain"
	"github.com/stretchr/testify/require"
)

func TestApplyHardforks_NoForks(t *testing.T) {
	cfg := Config{}
	hardforks := superchain.HardforkConfig{}
	applyHardforks(&cfg, hardforks)
	requireAllHardforksSetCorrectly(t, cfg, hardforks)
}

func TestApplyHardforks(t *testing.T) {
	cfg := Config{}
	hardforkCfg := superchain.HardforkConfig{}

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

func requireAllHardforksSetCorrectly(t *testing.T, cfg Config, hardforkCfg superchain.HardforkConfig) {
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
